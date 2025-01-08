// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package dashboard

import (
	"math"
	"strings"
)

type DashboardEntity struct {
	Dashboard `json:"dashboard"`
	Meta      `json:"meta"`
}

type Meta struct {
	Slug string `json:"slug"`
}
type Dashboard struct {
	Title  string  `json:"title"`
	Panels []Panel `json:"panels"`
	Uid    string  `json:"uid"`
}
type Panel struct {
	Id        int  `json:"id"`
	Collapsed bool `json:"collapsed"`
	GridPos   `json:"gridPos"`
	Title     string  `json:"title"`
	Type      string  `json:"type"`
	Panels    []Panel `json:"panels"`
}
type GridPos struct {
	H int `json:"h"`
	W int `json:"w"`
	X int `json:"x"`
	Y int `json:"y"`
}

type StructuredDashboard struct {
	Uid       string
	Title     string
	Slug      string
	Rows      []*Row
	Panels    []Panel
	RequestId string
}

type Row struct {
	Title string
	GridPos
	Panels []Panel
}
type PanelStructured struct {
	Id    string
	Title string
	GridPos
	Type string
}

func (de *DashboardEntity) GetStructuredDashboard(renderCollapsed bool) *StructuredDashboard {
	var rows []*Row
	for _, rowOrPanel := range de.Panels {
		if strings.EqualFold(rowOrPanel.Type, "row") {
			if rowOrPanel.Collapsed == renderCollapsed {
				rows = append(rows, &Row{
					Title:   rowOrPanel.Title,
					GridPos: rowOrPanel.GridPos,
					Panels:  rowOrPanel.Panels,
				})
			}
		} else if !strings.EqualFold(rowOrPanel.Type, "row") && len(rows) == 0 {
			rows = append(rows, &Row{
				Title:   "",
				GridPos: rowOrPanel.GridPos,
				Panels:  []Panel{rowOrPanel},
			})
		} else {
			rows = append(rows, &Row{
				Title:   "",
				GridPos: rowOrPanel.GridPos,
				Panels:  []Panel{rowOrPanel},
			})
		}
	}

	dsh := &StructuredDashboard{
		Uid:   de.Uid,
		Title: de.Title,
		Slug:  de.Slug,
		Rows:  rows,
	}
	return dsh
}

func (p *Panel) IsTheFirst() bool {
	return p.X == 0
}

func (p *Panel) IsTheLast() bool {
	return p.X+p.W == 24
}

func (p *Panel) GetPxWidth(screenResolutionWidth int) int {
	return p.W * (screenResolutionWidth / 24)
}

func (p *Panel) GetPxHeight(screenResolutionWidth int) int {
	return p.H * (screenResolutionWidth / 24)
}

func (p *Panel) GetRelativeWidth(screenResolutionWidth int) float64 {
	return roundFloat(float64(p.GetPxWidth(screenResolutionWidth))/float64(screenResolutionWidth), 3) - 0.005
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
