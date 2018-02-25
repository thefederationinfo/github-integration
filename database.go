package main

import (
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
)

func query(query string, value *string) error {
  db, err := sql.Open("sqlite3", "./server.db")
  if err != nil {
    return err
  }
  defer db.Close()

  stmt, err := db.Prepare(query)
  if err != nil {
    return err
  }
  defer stmt.Close()

  err = stmt.QueryRow().Scan(value)
  if err != nil {
    return err
  }
  return nil
}
