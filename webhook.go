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
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "io/ioutil"
)

func webhook(w http.ResponseWriter, r *http.Request) {
  db, err := sql.Open("sqlite3", "./server.db")
  if err != nil {
    fmt.Println(err)
    fmt.Fprintf(w, `{"error":"database error"}`)
  }
  defer db.Close()

  defer r.Body.Close()
  b, err := ioutil.ReadAll(r.Body)
  if err != nil {
    fmt.Println(err)
    fmt.Fprintf(w, `{"error":"invalid body"}`)
  }

  event, err := github.ParseWebHook(github.WebHookType(r), b)
  if err != nil {
    fmt.Println(err)
    fmt.Fprintf(w, `{"error":"invalid payload"}`)
    return
  }

  switch event := event.(type) {
  case *github.PullRequest:
    stmt, err := db.Prepare("select secret from repos where slug like ?")
    if err != nil {
      fmt.Println(err)
      fmt.Fprintf(w, `{"error":"database error"}`)
      return
    }
    defer stmt.Close()

    var secret string
    err = stmt.QueryRow(event.Base.Repo.FullName).Scan(&secret)
    if err != nil {
      fmt.Println(err)
      fmt.Fprintf(w, `{"error":"repo not registered"}`)
      return
    }

    // validate payload
    _, err = github.ValidatePayload(r, []byte(secret))
    if err != nil {
      fmt.Println(err)
      fmt.Fprintf(w, `{"error":"invalid signature"}`)
      return
    }

    matrix := []string{fmt.Sprintf(
      `"PRREPO=%s PRSHA=%s"`,
      event.Head.Repo.CloneURL,
      event.Head.SHA,
    )}
    go triggerTravisBuild(matrix)
    fmt.Fprintf(w, `{}`)
  default:
    fmt.Println("Not supported event type", event)
    fmt.Fprintf(w, `{"error":"unsupported event type"}`)
  }
}
