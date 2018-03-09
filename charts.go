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
  "github.com/wcharczuk/go-chart"
  "github.com/wcharczuk/go-chart/drawing"
  "github.com/jinzhu/gorm"
  _ "github.com/jinzhu/gorm/dialects/sqlite"
  "time"
  "fmt"
  "sort"
)

type DataSet struct {
  X []time.Time
  Y []float64
}

func (d DataSet) Len() int {
  return len(d.X)
}

func (d DataSet) Swap(i, j int) {
  d.X[i], d.X[j] = d.X[j], d.X[i]
  d.Y[i], d.Y[j] = d.Y[j], d.Y[i]
}

func (d DataSet) Less(i, j int) bool {
  return d.X[i].Before(d.X[j])
}

func buildsPNG(w http.ResponseWriter, r *http.Request) {
  db, err := gorm.Open(databaseDriver, databaseDSN)
  if err != nil {
    logger.Println(err)
    return
  }
  defer db.Close()

  var builds Builds
  err = db.Find(&builds).Error
  if err != nil {
    logger.Println(err)
    fmt.Fprintf(w, "ups.. something went wrong :(")
    return
  }

  var passed = make(map[time.Time]float64)
  var failed = make(map[time.Time]float64)
  for _, build := range builds {
    year, month, day := build.CreatedAt.Date()
    date := time.Date(year, month, day,0, 0, 0, 0, time.Local)
    if build.Status == BUILD_FINISHED {
      passed[date] += 1
    } else if build.Status == BUILD_FINISHED_ERROR {
      failed[date] += 1
    }
  }

  var maxX float64
  var p, f DataSet
  for k, v := range passed {
    p.X = append(p.X, k)
    p.Y = append(p.Y, v)
    if v > maxX {
      maxX = v
    }
  }
  for k, v := range failed {
    f.X = append(f.X, k)
    f.Y = append(f.Y, v)
    if v > maxX {
      maxX = v
    }
  }

  sort.Sort(p)
  sort.Sort(f)

  graph := chart.Chart{
    Background: chart.Style{
      FontSize: 30,
    },
    XAxis: chart.XAxis{
      Name: "Day",
      NameStyle: chart.StyleShow(),
      Style: chart.Style{
        Show: true,
      },
    },
    YAxis: chart.YAxis{
      Name: "Count",
      NameStyle: chart.StyleShow(),
      Style: chart.Style{
        Show: true,
      },
      Range: &chart.ContinuousRange{
        Min: 0.0,
        Max: maxX + 2,
      },
    },
    Series: []chart.Series{
      chart.TimeSeries{
        Name: "failed",
        Style: chart.Style{
          Show: true,
          StrokeWidth: 3,
          StrokeColor: drawing.ColorFromHex("A93226"),
          FillColor: drawing.ColorFromHex("CD6155").WithAlpha(64),
        },
        YValues: f.Y,
        XValues: f.X,
      },
      chart.TimeSeries{
        Name: "passed",
        Style: chart.Style{
          Show: true,
          StrokeWidth: 3,
          StrokeColor: drawing.ColorFromHex("229954"),
          FillColor: drawing.ColorFromHex("27AE60").WithAlpha(64),
        },
        YValues: p.Y,
        XValues: p.X,
      },
    },
  }

  w.Header().Set("Content-Type", "image/png")

  graph.Elements = []chart.Renderable{
    chart.Legend(&graph),
  }

  graph.Render(chart.PNG, w)
}
