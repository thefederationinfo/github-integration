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
  _ "github.com/jinzhu/gorm/dialects/sqlite"
)

type Repo struct {
  gorm.Model
  Project string
  Slug string
  Token string
  Secret string
  OptIn bool
  OptInFlag string `gorm:"default:'ci'"`
  OptOutFlag string `gorm:"default:'ci skip'"`
}

type Repos []Repo

func (repos *Repos) FindAll() error {
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    return err
  }
  defer db.Close()

  return db.Find(repos).Error
}

func (repo *Repo) CreateOrUpdate() error {
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    return err
  }
  defer db.Close()

  var oldRecord Repo
  err = db.Where(
    "project = ? and slug = ?", repo.Project, repo.Slug,
  ).Find(&oldRecord).Error
  if err == gorm.ErrRecordNotFound {
    err = db.Create(repo).Error
    if err != nil {
      return err
    }
  } else if err == nil {
    // NOTE you have to specify opt_in as extra update field
    // since gorm will not update if bool is false
    // see http://gorm.io/docs/update.html
    err = db.Model(repo).Where("id = ?", oldRecord.ID).
      Update(repo).Update("opt_in", repo.OptIn).Error
    if err != nil {
      return err
    }
  }
  return err
}
