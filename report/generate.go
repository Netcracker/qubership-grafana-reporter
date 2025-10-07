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

package report

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"text/template"

	"github.com/Netcracker/grafana-reporter/dashboard"
	"github.com/Netcracker/grafana-reporter/timerange"
)

type pdfData struct {
	StructDashboard *dashboard.StructuredDashboard
	From            string
	To              string
	TimestampFrom   string
	TimestampTo     string
	Vars            string
}

func generatePdf(templateBody string, structuredDashboard *dashboard.StructuredDashboard, timerangeData *timerange.TimerangeData, vars url.Values) error {
	funcMap := template.FuncMap{
		"decrm": func(i int) int {
			return i - 1
		},
		"rmdlr": func(s string) string {
			return strings.ReplaceAll(s, "$", "")
		},
	}
	templateObj, err := template.New("pdf_report").Funcs(funcMap).Delims("[[", "]]").Parse(templateBody)
	if err != nil {
		return fmt.Errorf("failed to create pdf template. Error: %w", err)
	}

	if err = os.MkdirAll(reportsDir, 0777); err != nil {
		return fmt.Errorf("failed to create reports directory. Error: %w", err)
	}
	fileTexName := fmt.Sprintf("%s.tex", structuredDashboard.RequestID)
	fileTex, err := os.Create(path.Join(reportsDir, fileTexName))
	if err != nil {
		return fmt.Errorf("failed to create report file. Error: %w", err)
	}
	defer func() {
		if err := fileTex.Close(); err != nil {
			slog.Error("Failed to close report file", "error", err)
		}
	}()

	data := pdfData{
		StructDashboard: structuredDashboard,
		From:            timerangeData.From,
		To:              timerangeData.To,
		TimestampFrom:   timerangeData.DateFrom.Format(timerange.Format),
		TimestampTo:     timerangeData.DateTo.Format(timerange.Format),
		Vars:            strings.ReplaceAll(vars.Encode(), "&", " "),
	}

	if err = templateObj.Execute(fileTex, data); err != nil {
		slog.Error(fmt.Sprintf("Error occurred when generating tex file. More details in %s%s and .log files", reportsDir, fileTexName), "err", err)
		return err
	}

	command := exec.Command("pdflatex", fmt.Sprintf("--output-dir=%s", reportsDir), path.Join(reportsDir, fileTexName))
	output, err := command.CombinedOutput()
	if err != nil {
		slog.Error("Error occurred when tex command executing", "err", err)
		return err
	}
	if output != nil {
		slog.Debug(fmt.Sprintf("Output of exec: %s", output))
	}

	return nil
}

func generateFile(templateBody string, structuredDashboard *dashboard.StructuredDashboard, timerangeData *timerange.TimerangeData, vars url.Values) error {
	defer func(requestID string) {
		save, found := os.LookupEnv("SAVE_TEMP_IMAGES")
		if found {
			toSaveImages, err := strconv.ParseBool(save)
			if err == nil && toSaveImages {
				return
			}
		}
		// delete all images from tmp directory
		dir := getPanelsDirPath(requestID)
		entries, err := os.ReadDir(dir)
		if err != nil {
			slog.Error("Could not read tmp directory of images", "error", err, "path", dir)
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".png") {
				imageFile := path.Join(dir, entry.Name())
				if err = os.Remove(imageFile); err != nil {
					slog.Error("Could not successfully delete image file", "error", err, "file", imageFile)
				}
			}
		}
	}(structuredDashboard.RequestID)

	err := generatePdf(templateBody, structuredDashboard, timerangeData, vars)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred while generating PDF report. Error: %v", err))
		return err
	}
	return nil
}

func getReport(requestID string) ([]byte, error) {
	report, err := os.ReadFile(path.Join(reportsDir, fmt.Sprintf("%s.pdf", requestID)))
	if err != nil {
		return nil, err
	}
	if len(report) < 1 {
		return nil, fmt.Errorf("report is empty")
	}
	return report, err
}

func generateUniqueRequestID(uid, from, to string, isCollapsed bool) string {
	var collapsed string
	if isCollapsed {
		collapsed = ""
	} else {
		collapsed = "_expanded"
	}
	return fmt.Sprintf("%s_report_%s-%s%s", uid, from, to, collapsed)
}
