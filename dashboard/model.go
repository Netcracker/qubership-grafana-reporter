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
	"fmt"
	"math"
	"strings"
)

const (
	grafanaResolutionWidth = 24
	rowsLimit              = 500
	panelsLimit            = 1000
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

func (de *DashboardEntity) GetStructuredDashboard(renderCollapsed bool) (*StructuredDashboard, error) {
	var rows []*Row
	panelsCount := 0
	for _, rowOrPanel := range de.Panels {
		if strings.EqualFold(rowOrPanel.Type, "row") {
			if rowOrPanel.Collapsed == renderCollapsed {
				rows = append(rows, &Row{
					Title:   rowOrPanel.Title,
					GridPos: rowOrPanel.GridPos,
					Panels:  rowOrPanel.Panels,
				})
				panelsCount += len(rowOrPanel.Panels)
			}
		} else if !strings.EqualFold(rowOrPanel.Type, "row") && (len(rows) == 0 || !rowOrPanel.IsAddedToPreviousRow(*rows[len(rows)-1])) {
			rows = append(rows, &Row{
				Title:   "",
				GridPos: rowOrPanel.GridPos,
				Panels:  []Panel{rowOrPanel},
			})
			panelsCount++
		} else {
			rows[len(rows)-1].Panels = append(rows[len(rows)-1].Panels, rowOrPanel)
			panelsCount++
		}
	}

	if len(rows) > rowsLimit || panelsCount > panelsLimit {
		return nil, fmt.Errorf("grafana dashboard contains too many rows/panels: rows=%d (limit=%d); panels=%d (limit=%d)", len(rows), rowsLimit, panelsCount, panelsLimit)
	}

	dsh := &StructuredDashboard{
		Uid:   de.Uid,
		Title: de.Title,
		Slug:  de.Slug,
		Rows:  rows,
	}
	return dsh, nil
}

func (p *Panel) IsTheFirst() bool {
	return p.X == 0
}

func (p *Panel) IsTheLast() bool {
	return p.X+p.W == grafanaResolutionWidth
}

func (p *Panel) GetPxWidth(screenResolutionWidth int) int {
	return p.W * (screenResolutionWidth / grafanaResolutionWidth)
}

func (p *Panel) GetPxHeight(screenResolutionWidth int) int {
	return p.H * (screenResolutionWidth / grafanaResolutionWidth)
}

func (p *Panel) GetRelativeWidth(screenResolutionWidth int) float64 {
	return roundFloat(float64(p.GetPxWidth(screenResolutionWidth))/float64(screenResolutionWidth), 3) - 0.005
}

func (p *Panel) IsAddedToPreviousRow(row Row) bool {
	rowWidth := p.W
	for _, panel := range row.Panels {
		rowWidth += panel.W
	}
	return rowWidth <= grafanaResolutionWidth
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
