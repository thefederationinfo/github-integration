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
  "github.com/jinzhu/gorm"
)

type Repo struct {
  gorm.Model
  Project string
  Slug string
  Token string
  Secret string
  OptIn bool
}

type Repos []Repo
