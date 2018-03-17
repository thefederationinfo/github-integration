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
  "log"
  "net/http"
  "golang.org/x/oauth2"
  oauth2Github "golang.org/x/oauth2/github"
  "time"
  "math/rand"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "flag"
  "os"
)

var (
  devMode = false
  databaseDriver = "sqlite3"
  databaseDSN = "./server.db"
  logger = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
  runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
  travisToken, serverDomain string
  conf = &oauth2.Config{
    Scopes: []string{"admin:repo_hook", "repo:status"},
    Endpoint: oauth2Github.Endpoint,
  }
)

func init() {
  rand.Seed(time.Now().UnixNano())

  flag.BoolVar(&devMode, "devMode", devMode,
    "If devMode is set all parameters become optional " +
    "and authentication is disabled!")
  flag.StringVar(&serverDomain, "server-domain", "localhost:8080",
    "Specify the endpoint your server is running on. " +
    "This is important for e.g. github callbacks!")
  flag.StringVar(&travisToken, "travis-token", "",
    "Specify the travis token for triggering builds (required)")
  flag.StringVar(&conf.ClientID, "github-id", "",
    "Specify the github client id (required)")
  flag.StringVar(&conf.ClientSecret, "github-secret", "",
    "Specify the github client secret (required)")

  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    panic("failed to connect database")
  }
  defer db.Close()

  build := &Build{}
  db.Model(build).AddUniqueIndex("index_builds_on_repo_id_and_matrix", "repo_id", "matrix")
  db.AutoMigrate(build)

  repo := &Repo{}
  db.Model(repo).AddUniqueIndex("index_repos_on_project_and_slug", "project", "slug")
  db.Model(repo).AddIndex("index_repos_on_slug", "slug")
  db.AutoMigrate(repo)
}

func Secret(n int) string {
  b := make([]rune, n)
  for i := range b {
    b[i] = runes[rand.Intn(len(runes))]
  }
  return string(b)
}

func main() {
  flag.Parse()
  if !devMode && (travisToken == "" || conf.ClientID == "" || conf.ClientSecret == "") {
    flag.Usage()
    return
  }

  // start build agent
  go BuildAgent()

  http.HandleFunc("/", indexPage)
  http.HandleFunc("/images/stats/builds.png", buildsPNG)
  http.HandleFunc("/auth", authenticationPage)
  http.HandleFunc("/result", resultPage)
  http.HandleFunc("/hook", webhook)

  logger.Println("Running webserver on :8181")
  logger.Println(http.ListenAndServe(":8181", nil))
}
