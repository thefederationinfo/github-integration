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
  oauth2Github "golang.org/x/oauth2/github"
  "time"
  "math/rand"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "flag"
)

var (
  runes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
  travisToken, serverDomain string
  travisEndpoint = "https://api.travis-ci.org/repo/"
  travisSlug = "thefederationinfo%2Ffederation-tests"
  travisRequests = travisEndpoint + travisSlug + "/requests"
  conf = &oauth2.Config{
    Scopes: []string{"admin:repo_hook"},
    Endpoint: oauth2Github.Endpoint,
  }
)

func init() {
  rand.Seed(time.Now().UnixNano())

  flag.StringVar(&serverDomain, "server-domain", "localhost:8080",
    "Specify the endpoint your server is running on. " +
    "This is important for e.g. github callbacks!")
  flag.StringVar(&travisToken, "travis-token", "",
    "Specify the travis token for triggering builds (required)")
  flag.StringVar(&conf.ClientID, "github-id", "",
    "Specify the github client id (required)")
  flag.StringVar(&conf.ClientSecret, "github-secret", "",
    "Specify the github client secret (required)")

  db, err := sql.Open("sqlite3", "./server.db")
  if err != nil {
    panic(err)
  }
  defer db.Close()

  _, err = db.Exec(`create table repos(slug text, token text, secret text);`)
  if err != nil {
    fmt.Println(err)
  }
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
  if travisToken == "" || conf.ClientID == "" || conf.ClientSecret == "" {
    flag.Usage()
    return
  }

  http.HandleFunc("/", frontend)
  http.HandleFunc("/hook", webhook)

  fmt.Println("Running webserver on :8080")
  fmt.Println(http.ListenAndServe(":8080", nil))
}
