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
  "golang.org/x/oauth2"
  "github.com/google/go-github/github"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "context"
)

func frontend(w http.ResponseWriter, r *http.Request) {
  ctx := context.Background()
  accessToken := r.URL.Query().Get("access_token")
  repo := r.URL.Query().Get("repo")
  if accessToken != "" && repo != "" {
    db, err := sql.Open("sqlite3", "./server.db")
    if err != nil {
      logger.Println(err)
      fmt.Fprintf(w, "Database Failure :(")
      return
    }
    defer db.Close()

    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
      &oauth2.Token{AccessToken: accessToken},
    )
    tc := oauth2.NewClient(ctx, ts)
    client := github.NewClient(tc)

    name := "web"
    secret := Secret(16)
    hook := github.Hook{
      Name: &name, Events: []string{"pull_request"},
      Config: map[string]interface{}{
        "url": serverDomain + "/hook",
        "secret": secret,
      },
    }

    _, _, err = client.Repositories.CreateHook(ctx, "ganggo", "ganggo", &hook)
    if err != nil {
      logger.Println(err)
      fmt.Fprintf(w, "Create Hook Failure :(")
      return
    }

    _, err = db.Exec(fmt.Sprintf(`insert into repos(slug, token, secret)
      values('%s', '%s', '%s');`, repo, accessToken, secret,
    )); if err != nil {
      logger.Println(err)
      fmt.Fprintf(w, "Database Insert Failure :(")
      return
    }

    fmt.Fprintf(w, "Success :) You can undo it by simply revoking permissions on Github.")
    return
  }

  code := r.URL.Query().Get("code")
  if code != "" {
    tok, err := conf.Exchange(ctx, code)
    if err != nil {
      fmt.Println(err)
      fmt.Fprintf(w, "Token Failure :(")
    } else {
      fmt.Fprintf(w, `<!DOCTYPE html>
        <html>
        <body>
        <p>You are authenticated :) Please add your repository:</p>
        <form method="GET">
          <input type="hidden" name="access_token" value="%s" />
          <input type="text" name="repo" placeholder="user/repository" />
        </form>
        </body>
        </html>`, tok.AccessToken)
    }
  } else {
    url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
    http.Redirect(w, r, url, http.StatusMovedPermanently)
  }
}
