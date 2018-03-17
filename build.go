//
// TheFederation Github Integration Server
// Copyright (C) 2018 Lukas Matt <lukas@zauberstuhl.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
package main

import (
  "fmt"
  "net/http"
  "strings"
  "encoding/json"
  "golang.org/x/oauth2"
  "github.com/google/go-github/github"
  "io/ioutil"
  "context"
  "time"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
)

const (
  STATUS_ERROR = "error"
  STATUS_FAIL = "failure"
  STATUS_PENDING = "pending"
  STATUS_SUCCESS = "success"

  BUILD_NOT_STARTED = 0
  BUILD_PENDING = 1
  BUILD_FINISHED = 2
  BUILD_FINISHED_ERROR = 3
)

type Build struct {
  gorm.Model
  RepoID uint
  Matrix string
  TravisType string
  TravisRequestID int64
  TravisRepositoryID int64
  PRUser string
  PRRepo string
  PRSha string
  Status int `gorm:"default:0"`

  Repo Repo
}

type Builds []Build

type TravisStatus struct {
  State string `json:"state"`
  Builds []struct {
    ID int64 `json:"id"`
    State string `json:"state"`
  } `json:"builds"`
}

type TravisRequest struct {
  Type string `json:"@type"`
  Request struct {
    ID int64 `json:"id"`
    Repository struct {
      ID int64 `json:"id"`
    } `json:"repository"`
  } `json:"request"`
}

var (
  travisTestEndpoint = "https://travis-ci.org/thefederationinfo/federation-tests/builds/%d"
  travisTestDescription = "Continuous integration tests for the federation network"
  travisTestContext = "Federation Suite"
  travisEndpoint = "https://api.travis-ci.org/repo/"
  travisSlug = "thefederationinfo%2Ffederation-tests"
  travisRequests = travisEndpoint + travisSlug + "/requests"
)

func (build *Build) AfterFind(db *gorm.DB) error {
  return db.Model(build).Related(&build.Repo).Error
}

func BuildAgent() {
  logger.Println("Started build agent")
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    panic("failed to connect database")
  }
  defer db.Close()

  for {
    // sleep for a bit before continuing
    time.Sleep(10 * time.Second)

    var builds Builds
    err := db.Find(&builds).Error
    if err != nil {
      //logger.Printf("Cannot fetch new builds: %+v\n", err)
      continue
    }
    for _, build := range builds {
      if build.Status == BUILD_NOT_STARTED {
        build.Status = BUILD_PENDING
        err = db.Save(&build).Error
        if err != nil {
          logger.Printf("#%d: cannot update status: %+v\n", build.ID, err)
          continue
        }
        logger.Printf("#%d: starting new build\n", build.ID)
        go build.Run(false)
      }
    }
  }
}

func (build *Build) Run(watch bool) {
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    logger.Println(err)
    return
  }
  defer db.Close()

  ts := oauth2.StaticTokenSource(
    &oauth2.Token{AccessToken: build.Repo.Token},
  )
  tc := oauth2.NewClient(context.Background(), ts)
  client := github.NewClient(tc)

  if !watch {
    status := build.TriggerTravis()
    logger.Printf("#%d: travis build triggered\n", build.ID)
    build.UpdateStatus(client, status)
    if status == STATUS_ERROR {
      (*build).Status = BUILD_FINISHED
      err := db.Save(&build).Error
      if err != nil {
        logger.Printf("#%d: cannot update status: %+v\n", build.ID, err)
      }
      return
    }
  }

  var statusHref string
  started := time.Now()
  timeout := started.Add(-1 * time.Hour)
  for {
    status := build.FetchStatus()
    logger.Printf("#%d: request status: %+v", build.ID, status)
    if status.State == "finished" {
      var failure bool
      var passed int
      for _, build := range status.Builds {
        // canceled, passed, errored, started
        switch build.State {
        case "canceled":
          fallthrough
        case "errored":
          fallthrough
        case "failed":
          failure = true
        case "passed":
          passed += 1
        }
      }
      if failure {
        build.UpdateStatus(client, STATUS_FAIL, statusHref)
        (*build).Status = BUILD_FINISHED_ERROR
        err := db.Save(&build).Error
        if err != nil {
          logger.Printf("#%d: cannot update status: %+v\n", build.ID, err)
        }
        break
      } else if len(status.Builds) == passed {
        build.UpdateStatus(client, STATUS_SUCCESS, statusHref)
        (*build).Status = BUILD_FINISHED
        err := db.Save(&build).Error
        if err != nil {
          logger.Printf("#%d: cannot update status: %+v\n", build.ID, err)
        }
        break
      }
      // update the status line in the PR once
      if len(status.Builds) > 0 && statusHref == "" {
        statusHref = fmt.Sprintf(travisTestEndpoint, status.Builds[0].ID)
        build.UpdateStatus(client, STATUS_PENDING, statusHref)
      }
    }

    if time.Now().Before(timeout) {
      build.UpdateStatus(client, STATUS_ERROR, statusHref)
      logger.Printf("#%d: Timeout..\n", build.ID)
      (*build).Status = BUILD_FINISHED
      err := db.Save(&build).Error
      if err != nil {
        logger.Printf("#%d: cannot update status: %+v\n", build.ID, err)
      }
      break
    }
    time.Sleep(1 * time.Minute)
  }
  logger.Printf("#%d: Travis build finished\n", build.ID)
}

func (build *Build) TriggerTravis() string {
  var requestJson = `{"request": {
    "branch": "continuous_integration",
    "config": {
      "merge_mode": "replace",
      "sudo": "required",
      "language": "go",
      "services": ["postgresql", "docker", "redis"],
      "env": {"matrix": [%s]},
      "install": "bash scripts/install.sh",
      "script": "bats --tap $(find . -name $PROJECT'*.bats')"
    }
  }}`

  resp, err := build.fetch("POST", travisRequests,
    fmt.Sprintf(requestJson, build.Matrix))
  if err != nil {
    fmt.Printf("#%d: Cannot create request: %+v\n", build.ID, err)
    return STATUS_ERROR
  }
  defer resp.Body.Close()

  b, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    fmt.Printf("#%d: Cannot read status body: %+v\n", build.ID, err)
    return STATUS_ERROR
  }

  var request TravisRequest
  err = json.Unmarshal(b, &request)
  if err != nil {
    fmt.Printf("#%d: Cannot unmarshal body: %+v <> %s\n", build.ID, err, string(b))
    return STATUS_ERROR
  }

  build.TravisType = request.Type
  build.TravisRequestID = request.Request.ID
  build.TravisRepositoryID = request.Request.Repository.ID

  return STATUS_PENDING
}

func (build *Build) UpdateStatus(client *github.Client, params... string) {
  if len(params) <= 0 {
    panic("state is mandatory")
  }

  repoStatus := github.RepoStatus{
    State: &params[0],
    Description: &travisTestDescription,
    Context: &travisTestContext,
  }
  if len(params) >= 2 {
    repoStatus.TargetURL = &params[1]
  }

  slug := strings.Split(build.Repo.Slug, "/")
  if len(slug) <= 1 {
    fmt.Printf("#%d: Invalid repor slug: %s\n", build.ID, build.Repo.Slug)
    return
  }

  if _, _, err := client.Repositories.CreateStatus(context.Background(),
  slug[0], slug[1], build.PRSha, &repoStatus); err != nil {
    fmt.Printf("#%d: Cannot update status: %+v\n", build.ID, err)
  }
}

func (build *Build) FetchStatus() (status TravisStatus) {
  resp, err := build.fetch("GET", fmt.Sprintf(
    "%s%d%s%d", travisEndpoint, build.TravisRepositoryID,
    "/request/", build.TravisRequestID), "")
  if err != nil {
    logger.Printf("#%d: Cannot fetch build status: %+v\n", build.ID, err)
    return
  }
  defer resp.Body.Close()

  b, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    logger.Printf("#%d: Cannot read status body: %+v\n", build.ID, err)
    return
  }

  err = json.Unmarshal(b, &status)
  if err != nil {
    logger.Printf("#%d: Cannot unmarshal body: %+v <> %s\n", build.ID, err, string(b))
    return
  }
  return
}

func (build *Build) fetch(method, url, body string) (*http.Response, error) {
  req, err := http.NewRequest(method, url, strings.NewReader(body))
  if err != nil {
    return nil, err
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Accept", "application/json")
  req.Header.Set("Travis-API-Version", "3")
  req.Header.Set("Authorization", "token " + travisToken)

  client := &http.Client{}
  return client.Do(req)
}
