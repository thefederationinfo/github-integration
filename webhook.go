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
  "github.com/google/go-github/github"
  "io/ioutil"
  "encoding/json"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "strings"
)

func webhook(w http.ResponseWriter, r *http.Request) {
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, `{"error":"database error"}`)
    return
  }
  defer db.Close()
  defer r.Body.Close()

  b, err := ioutil.ReadAll(r.Body)
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, `{"error":"invalid body"}`)
    return
  }

  var event github.PullRequestReviewEvent
  err = json.Unmarshal(b, &event)
  pr := event.PullRequest
  if err != nil || pr == nil {
    logger.Println("Not supported event type", string(b))
    fmt.Fprintf(w, `{"error":"unsupported event type"}`)
    return
  }

  // skip all events except for open PRs
  if *pr.State != "open" {
    logger.Println("Ignore closed pull request")
    fmt.Fprintf(w, `{}`)
    return
  }

  var repo Repo
  err = db.Where("slug = ?", *pr.Base.Repo.FullName).Find(&repo).Error
  if err != nil {
    logger.Println(err, *pr.Base.Repo.FullName)
    fmt.Fprintf(w, `{"error":"repo not registered"}`)
    return
  }

  // XXX validate payload
  //_, err = github.ValidatePayload(r, []byte(repo.Secret))
  //if err != nil {
  //  logger.Println(err, repo.Secret)
  //  fmt.Fprintf(w, `{"error":"invalid signature"}`)
  //  return
  //}

  var flagExists = false
  var buildFlag = "ci skip"
  if repo.OptIn {
    buildFlag = "ci"
  }

  // check PR title and body for [ci] or [ci skip] flag
  if pr.Title != nil && pr.Body != nil &&
  strings.Contains(strings.ToLower(*pr.Title),
      fmt.Sprintf("[%s]", buildFlag)) &&
  strings.Contains(strings.ToLower(*pr.Body),
      fmt.Sprintf("[%s]", buildFlag)) {
    flagExists = true
  }

  if !flagExists {
    // check labels for build flag if we haven't already found it
    for _, label := range pr.Labels {
      if label.Name != nil && strings.Contains(
          strings.ToLower(*label.Name), buildFlag) {
        flagExists = true
        break
      }
    }
  }

  // ignoring pull-request! Repository is set
  // to opt-in and no build flag was found
  if repo.OptIn && !flagExists {
    fmt.Fprintf(w, `{}`)
    return
  }

  // ignoring pull-request! Repository is set
  // to opt-out and a skip flag was found
  if !repo.OptIn && flagExists {
    fmt.Fprintf(w, `{}`)
    return
  }

  build := Build{
    RepoID: repo.ID,
    Matrix: fmt.Sprintf(
      `"PROJECT=%s PRREPO=%s PRSHA=%s"`,
      repo.Project,
      *pr.Head.Repo.CloneURL,
      *pr.Head.SHA,
    ),
    PRUser: *pr.Head.User.Login,
    PRRepo: *pr.Head.Repo.Name,
    PRSha: *pr.Head.SHA,
  }
  err = db.Create(&build).Error
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, `{"error":"database error"}`)
    return
  }

  fmt.Fprintf(w, `{}`)
}
