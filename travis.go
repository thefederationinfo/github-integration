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
)

const (
  STATUS_ERROR = "error"
  STATUS_FAIL = "failure"
  STATUS_PENDING = "pending"
  STATUS_SUCCESS = "success"
)

var (
  travisTestEndpoint = "https://travis-ci.org/thefederationinfo/federation-tests/builds/%d"
  travisTestDescription = "Continuous integration tests for the federation network"
  travisTestContext = "Federation Suite"
  travisEndpoint = "https://api.travis-ci.org/repo/"
  travisSlug = "thefederationinfo%2Ffederation-tests"
  travisRequests = travisEndpoint + travisSlug + "/requests"
)

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

func (request *TravisRequest) Run(token string, matrix []string, pr *github.PullRequest) {
  ts := oauth2.StaticTokenSource(
    &oauth2.Token{AccessToken: token},
  )
  tc := oauth2.NewClient(context.Background(), ts)
  client := github.NewClient(tc)

  status := request.Build(matrix)
  request.UpdateStatus(client, pr, status)
  if status == STATUS_ERROR {
    return
  }

  startedJobs := 0
  started := time.Now()
  timeout := started.Add(-1 * time.Hour)
  logger.Printf("#%d: Travis build triggered\n", request.Request.ID)
  for {
    status := request.Status()
    if status.State == "finished" {
      var failure bool
      var passed int
      for _, build := range status.Builds {
        // canceled, passed, errored, started
        if build.State == "canceled" || build.State == "errored" {
          failure = true
        }
        if build.State == "passed" {
          passed += 1
        }
      }
      if failure {
        request.UpdateStatus(client, pr, STATUS_FAIL)
      } else if len(status.Builds) == passed {
        request.UpdateStatus(client, pr, STATUS_SUCCESS)
        break
      }
      // update the status lin in the PR once
      if len(status.Builds) > 0 && startedJobs == 0 {
        request.UpdateStatus(client, pr, STATUS_PENDING,
          fmt.Sprintf(travisTestEndpoint, status.Builds[0].ID))
        startedJobs = len(status.Builds)
      }
    }

    if time.Now().Before(timeout) {
      logger.Printf("#%d: Timeout..\n", request.Request.ID)
      break
    }
    time.Sleep(1 * time.Minute)
  }
  logger.Printf("#%d: Travis build finished\n", request.Request.ID)
}

func (request *TravisRequest) UpdateStatus(client *github.Client, pr *github.PullRequest, params... string) {
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
  if _, _, err := client.Repositories.CreateStatus(
    context.Background(), *pr.Head.User.Login, *pr.Head.Repo.Name,
    *pr.Head.SHA, &repoStatus); err != nil {
    fmt.Println("#%d: Cannot update status: %+v", request.Request.ID, err)
  }
}

func (request *TravisRequest) Build(matrix []string) string {
  var requestJson = `{"request":{"branch":"continuous_integration","config":{"env":{"matrix":[%s]}}}}`
  resp, err := request.fetch("POST", travisRequests,
    fmt.Sprintf(requestJson, strings.Join(matrix, ",")))
  if err != nil {
    fmt.Println("#%d: Cannot create request: %+v", request.Request.ID, err)
    return STATUS_ERROR
  }
  defer resp.Body.Close()

  b, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    fmt.Println("#%d: Cannot read status body: %+v", request.Request.ID, err)
    return STATUS_ERROR
  }

  err = json.Unmarshal(b, &request)
  if err != nil {
    fmt.Println("#%d: Cannot unmarshal body: %+v <> %s",
      request.Request.ID, err, string(b))
    return STATUS_ERROR
  }
  return STATUS_PENDING
}

func (request *TravisRequest) Status() (status TravisStatus) {
  resp, err := request.fetch("GET", fmt.Sprintf(
    "%s%d%s%d", travisEndpoint,
    request.Request.Repository.ID,
    "/request/", request.Request.ID), "")
  if err != nil {
    logger.Printf("#%d: Cannot fetch build status: %+v\n", request.Request.ID, err)
    return
  }
  defer resp.Body.Close()

  b, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    logger.Printf("#%d: Cannot read status body: %+v\n", request.Request.ID, err)
    return
  }

  err = json.Unmarshal(b, &status)
  if err != nil {
    logger.Printf("#%d: Cannot unmarshal body: %+v <> %s\n",
      request.Request.ID, err, string(b))
    return
  }
  return
}

func (request *TravisRequest) fetch(method, url, body string) (*http.Response, error) {
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
