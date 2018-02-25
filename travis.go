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
  "io/ioutil"
)

type TravisRequest struct {
  Type string `json:"@type"`
}

func triggerTravisBuild(matrix []string) bool {
  var requestJson = `{
    "request": {
      "branch": "master",
      "config": {
        "env": {
          "matrix": [%s]
        }
      }
    }
  }`
  // travis trigger build
  req, err := http.NewRequest("POST", travisRequests,
    strings.NewReader(fmt.Sprintf(
      requestJson, strings.Join(matrix, ","),
    )))
  if err != nil {
    fmt.Println(err)
    return false
  }
  req.Header.Set("Content-Type", "application/json")
  req.Header.Set("Accept", "application/json")
  req.Header.Set("Travis-API-Version", "3")
  req.Header.Set("Authorization", "token " + travisToken)

  client := &http.Client{}
  resp, err := client.Do(req)
  if err != nil {
    fmt.Println(err)
    return false
  }
  defer resp.Body.Close()

  b, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    fmt.Println(err)
    return false
  }

  var request TravisRequest
  err = json.Unmarshal(b, &request)
  if err != nil {
    fmt.Println(string(b), err)
    return false
  }
  return request.Type == "pending"
}
