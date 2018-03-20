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
  "net/http"
  "html/template"
  "fmt"
  "golang.org/x/oauth2"
  "github.com/google/go-github/github"
  "context"
  "strings"
)

func add(a, b int) int {
  return a + b
}

func render(w http.ResponseWriter, name string, s interface{}) {
  rootTmpl := template.New("").Funcs(template.FuncMap{
    "add": add,
  })

  tmpl, err := rootTmpl.ParseFiles(
    "templates/header.html",
    fmt.Sprintf("templates/%s", name),
    "templates/footer.html",
  ); if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, "Wasn't able to parse the template")
    return
  }

  err = tmpl.ExecuteTemplate(w, name, s)
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, "Wasn't able to execute the template")
    return
  }
}

func indexPage(w http.ResponseWriter, r *http.Request) {
  var repos Repos
  err := repos.FindAll()
  if err != nil {
    logger.Println(err)
    render(w, "error.html", "No repositories found")
    return
  }

  render(w, "index.html", repos)
}

func resultPage(w http.ResponseWriter, r *http.Request) {
  accessToken := r.URL.Query().Get("access_token")
  repo := r.URL.Query().Get("repo")
  project := r.URL.Query().Get("project")

  if accessToken != "" && repo != "" && project != "" {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
      &oauth2.Token{AccessToken: accessToken},
    )
    tc := oauth2.NewClient(ctx, ts)
    client := github.NewClient(tc)

    repoSlice := strings.Split(repo, "/")
    if len(repoSlice) < 2 {
      logger.Println("invalid repository string")
      render(w, "error.html", "Invalid repository string")
      return
    }

    secret := Secret(16)
    repo := Repo{
      Project: project,
      Slug: repo,
      Token: accessToken,
      Secret: secret,
    }

    // set the repo to opt-in
    if strings.ToUpper(r.URL.Query().Get("optin")) == "ON" {
      repo.OptIn = true
    }
    // define custom opt-in flag
    if r.URL.Query().Get("optinFlag") != "" {
      repo.OptInFlag = r.URL.Query().Get("optinFlag")
    }
    // define custom opt-out flag
    if r.URL.Query().Get("optoutFlag") != "" {
      repo.OptOutFlag = r.URL.Query().Get("optoutFlag")
    }

    name := "web"
    newHook := github.Hook{
      Name: &name, Events: []string{"pull_request"},
      Config: map[string]interface{}{
        "content_type": "json",
        "url": serverDomain + "/hook",
        "secret": secret,
      },
    }

    if !devMode {
      hooks, _, err := client.Repositories.ListHooks(ctx,
        repoSlice[0], repoSlice[1], &github.ListOptions{})
      if err != nil {
        logger.Println(err)
        render(w, "error.html", "Cannot list repository hooks")
        return
      }

      var hookExists = false
      hookURL, _ := newHook.Config["url"].(string)
      for _, hook := range hooks {
        curHookURL, _ := hook.Config["url"].(string)
        if hookURL == curHookURL {
          hookExists = true
        }
      }

      if !hookExists {
        _, _, err := client.Repositories.CreateHook(ctx,
          repoSlice[0], repoSlice[1], &newHook)
        if err != nil {
          logger.Println(err)
          render(w, "error.html", "Cannot create repository hooks")
          return
        }
      }
    }
    err := repo.CreateOrUpdate()
    if err != nil {
      logger.Println(err)
      render(w, "error.html", "Cannot insert/update the database record")
      return
    }

    render(w, "result.html", repo.Slug)
  } else {
    render(w, "error.html", "Missing parameters: access_token, repo or project")
  }
}

func authenticationPage(w http.ResponseWriter, r *http.Request) {
  code := r.URL.Query().Get("code")
  if code != "" {
    tok, err := conf.Exchange(context.Background(), code)
    if !devMode && err != nil {
      render(w, "error.html", "Invalid token")
    } else {
      var token string = "1234"
      if !devMode {
        token = tok.AccessToken
      }
      render(w, "auth.html", token)
    }
  } else {
    url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
    http.Redirect(w, r, url, http.StatusMovedPermanently)
  }
}
