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
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "fmt"
)

func frontend(w http.ResponseWriter, r *http.Request) {
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, "Database Error :(")
    return
  }
  defer db.Close()

  var repos Repos
  err = db.Find(&repos).Error
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, "Database Query Error :(")
    return
  }

  fmt.Fprintf(w, `<!DOCTYPE html>
  <html>
  <head>
    <title>Federation Testsuite</title>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/css/bootstrap.min.css" integrity="sha384-Gn5384xqQ1aoWXA+058RXPxPg6fy4IWvTNh0E263XmFcJlSAwiGgFAW/dAiS6JXm" crossorigin="anonymous">
  </head>
  <body>
    <div class="container">
    <h1 class="pt-5">Federation Testsuite</h1>
    <h4>How to add your project to the list</h2>
    <p class="mb-0">Follow the <a href="https://github.com/thefederationinfo/federation-tests/blob/master/README.md">instructions</a> and add your own image to the repository.</p>
    <p>Then register your project with the <a href="/auth">integration service</a>.</p>
    <h4>Covered Projects</h2>
    <p>Following projects are covered by our <a href="https://github.com/thefederationinfo/federation-tests">continuous integration tests</a>:</p>
    <table class="table">
      <thead>
        <tr>
          <th scope="col">#</th>
          <th scope="col">Slug</th>
          <th scope="col">Project</th>
        </tr>
      </thead>
      <tbody>`)
    for i, repo := range repos {
      fmt.Fprintf(w, `
        <tr>
          <th scope="row">%d</th>
          <td>%s</td>
          <td>
            <a href="https://github.com/%s">Details</a>
          </td>
        </tr>`, i+1, repo.Slug, repo.Slug)
    }
    fmt.Fprintf(w, `
      </tbody>
    </table>
    </div>
    <script src="https://code.jquery.com/jquery-3.2.1.slim.min.js" integrity="sha384-KJ3o2DKtIkvYIK3UENzmM7KCkRr/rE9/Qpg6aAZGJwFDMVNA/GpGFF93hXpG5KkN" crossorigin="anonymous"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.12.9/umd/popper.min.js" integrity="sha384-ApNbgh9B+Y1QKtv3Rn7W3mgPxhU9K/ScQsAP7hUibX39j7fakFPskvXusvfa0b4Q" crossorigin="anonymous"></script>
    <script src="https://maxcdn.bootstrapcdn.com/bootstrap/4.0.0/js/bootstrap.min.js" integrity="sha384-JZR6Spejh4U02d8jOt6vLEHfe/JQGiRRSQQxSfFWpi1MquVdAyjUar5+76PVCmYl" crossorigin="anonymous"></script>
  </body>
  </html>`)
}
