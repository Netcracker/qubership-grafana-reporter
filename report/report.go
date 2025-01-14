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
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Netcracker/grafana-reporter/dashboard"
	"github.com/Netcracker/grafana-reporter/timerange"

	"golang.org/x/sync/errgroup"
	yaml "gopkg.in/yaml.v3"
)

var (
	reportsDir = path.Join(os.TempDir(), "reports/")
)

type GrafanaInstance struct {
	http.Client
	Endpoint        string
	Credentials     string
	DefaultFrom     string
	DefaultTo       string
	DefaultTemplate string
	Templates       map[string][]byte
	RenderCollapsed bool
}

type Credentials struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Token    string `yaml:"apiKey"`
}

func RunGenerateReport(addr, credentialsFile, dashboardUID, variables string, templates map[string][]byte, defaultTemplate, defaultFrom, defaultTo string, renderCollapsed bool, tlsConfig *tls.Config, user, password, token string) error {
	slog.Info("Generation started...")

	if len(dashboardUID) == 0 {
		return fmt.Errorf("dashboard UID can not be empty")
	}

	transportConf := http.DefaultTransport.(*http.Transport).Clone()
	transportConf.TLSClientConfig = tlsConfig
	g := &GrafanaInstance{
		DefaultTemplate: defaultTemplate,
		Templates:       templates,
		DefaultFrom:     defaultFrom,
		DefaultTo:       defaultTo,
		Endpoint:        addr,
		Credentials:     credentialsFile,
		RenderCollapsed: renderCollapsed,
		Client: http.Client{
			Transport: transportConf,
		},
	}
	startTime := time.Now()
	timerangeFrom := g.DefaultFrom
	timerangeTo := g.DefaultTo
	texTemplate := g.DefaultTemplate

	vars, err := url.ParseQuery(variables)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when parsing variables. Error: %v", err))
		return err
	}
	for k := range vars {
		if !strings.HasPrefix(k, "var-") {
			err = fmt.Errorf("could not read var-* parameter. Name of parameter %q is not valid", k)
			slog.Error(fmt.Sprintf("Error occurred when parsing variables. Error: %v", err))
			return err
		}
	}

	authHeader, err := g.getAuthHeaderFromParameters(user, password, token)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when getting authorization header. Error: %v", err))
		return err
	}

	timestampFrom, err := timerange.RelativeTimeToTimestamp(startTime, timerangeFrom, "from")
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when converting parameter %q to timestamp. Error: %v", "from", err))
		return err
	}
	timestampTo, err := timerange.RelativeTimeToTimestamp(startTime, timerangeTo, "to")
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when converting parameter %q to timestamp. Error: %v", "to", err))
		return err
	}
	timerangeData := &timerange.TimerangeData{
		From:     timerangeFrom,
		To:       timerangeTo,
		DateFrom: timestampFrom,
		DateTo:   timestampTo,
	}
	requestId := generateUniqueRequestId(dashboardUID, timerangeFrom, timerangeTo, !g.RenderCollapsed)
	slog.Info(fmt.Sprintf("Generating report %q with parameters: dashboardId=%s, from=%v, to=%v, template=%s, vars=%s", requestId, dashboardUID, timerangeFrom, timerangeTo, texTemplate, vars.Encode()))
	report, err := g.generateReport(dashboardUID, timerangeData, texTemplate, vars, requestId, authHeader, g.RenderCollapsed)
	duration := time.Since(startTime).String()
	slog.Info(fmt.Sprintf("The job took %s", duration))
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when generating report. Error: %v", err))
		return err
	}
	fileName := fmt.Sprintf("%s.pdf", requestId)
	fileReport, err := os.Create(path.Join(reportsDir, fileName))
	if err != nil {
		return fmt.Errorf("failed to create report file. Error: %w", err)
	}
	defer fileReport.Close()
	_, err = fileReport.Write(report)
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Report generation is succeeded. File name: %s", fileName))
	return nil
}

func (g *GrafanaInstance) generateReport(dashboardID string, timerangeData *timerange.TimerangeData, templateName string, vars url.Values, requestId string, authHeader string, renderCollapsed bool) ([]byte, error) {
	//get dashboard
	structuredDashboard, err := g.getDashboard(dashboardID, authHeader, renderCollapsed)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred while getting Grafana dashboard: %s", err))
		return nil, err
	}
	structuredDashboard.RequestId = requestId
	//get panels
	ok, err := g.getPanels(structuredDashboard, timerangeData.From, timerangeData.To, vars, requestId, authHeader)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred while getting panels: %s", err))
		return nil, err
	}
	if !ok {
		return nil, err
	}

	//generate report from images and template
	err = generateFile(string(g.Templates[templateName]), structuredDashboard, timerangeData, vars)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred while generating report file: %s", err))
		return nil, err
	}
	//get tex file from reportsDir
	report, err := getReport(requestId)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred while generating PDF report. Error: %v", err))
		return nil, err
	}
	return report, err

}

// HandleGetTemplatesList godoc
//
//	@Summary		Get names of available tex templates
//	@Description	Get names of all available tex templates
//	@Tags			General
//	@id             getTexTemplates
//	@Produce		json
//	@Success		200	{object}	string "OK"
//	@Router			/api/v1/templates [get]
func (g *GrafanaInstance) HandleGetTemplatesList(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	var templatesList []string
	for templateName := range g.Templates {
		templatesList = append(templatesList, templateName)
	}
	err := json.NewEncoder(writer).Encode(templatesList)
	if err != nil {
		slog.Error("Could not encode templates list", "error", err)
	}
}

// HandleGetTemplate godoc
//
//	@Summary		Get tex template by name
//	@Description	Get tex template by name
//	@Tags			General
//	@id             getTexTemplate
//	@Param          template   path string true "PDF tex template name"
//	@Produce		json
//	@Success		200	{object}	map[string]string "OK"
//	@Failure		400	{string}    string "Bad Request"
//	@Failure		404	{string}    string "Not Found"
//	@Router			/api/v1/report/{template} [get]
func (g *GrafanaInstance) HandleGetTemplate(writer http.ResponseWriter, request *http.Request) {
	urlPath := strings.Split(request.URL.Path, "/")
	if len(urlPath) != 5 || urlPath[4] == "" {
		writer.WriteHeader(http.StatusBadRequest)
		_, err := writer.Write([]byte("error"))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		slog.Error(fmt.Sprintf("Handle of invalid URL path. Path: %s", request.URL.Path))
		return
	}

	templateName := urlPath[4]
	texTemplate, ok := g.Templates[templateName]
	if !ok {
		writer.WriteHeader(http.StatusNotFound)
		_, err := writer.Write([]byte("template does not exist"))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		slog.Warn(fmt.Sprintf("Could not get template. Template %q does not exist", templateName))
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	err := json.NewEncoder(writer).Encode(map[string]string{templateName: string(texTemplate)})
	if err != nil {
		slog.Error("Could not encode tex template", "error", err)
	}
}

// HandleGetDefaultParameters godoc
//
//	@Summary		Get values of default parameters
//	@Description	Get values of default parameters such as default template and time range
//	@Tags			General
//	@id             getDefaults
//	@Produce		json
//	@Success		200	{object}	string "OK"
//	@Router			/api/v1/defaults [get]
func (g *GrafanaInstance) HandleGetDefaultParameters(writer http.ResponseWriter) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	defaults := map[string]string{
		"template": g.DefaultTemplate,
		"from":     g.DefaultFrom,
		"to":       g.DefaultTo,
	}
	err := json.NewEncoder(writer).Encode(defaults)
	if err != nil {
		slog.Error("Could not encode default parameters", "error", err)
	}
}

// HandleGenerateReport godoc
//
//	@Summary		Generate Grafana dashboard report
//	@Description	Generate Grafana dashboard report in PDF file. You can set time range, tex template and other parameters `var-` from Grafana
//	@Tags			Generate
//	@id             generateReport
//	@Param			Authorization	header		string	true	"Authentication header"
//	@Param          dashboard_uid   path string true "Dashboard UID"
//	@Param          template   query string false "PDF tex template name"
//	@Param          from   query string false "The start of time range"
//	@Param          to   query string false "The end of time range"
//	@Produce		octet-stream
//	@Success		200	{object}    string "OK"
//	@Failure		400	{string}    string "Bad Request"
//	@Failure		401	{string}    string "Unauthorized"
//	@Router			/api/v1/report/{dashboard_uid} [get]
//	@Router			/api/v1/report/{dashboard_uid} [post]
func (g *GrafanaInstance) HandleGenerateReport(writer http.ResponseWriter, request *http.Request) {
	startTime := time.Now()

	urlPath := strings.Split(request.URL.Path, "/")
	if len(urlPath) != 5 || urlPath[4] == "" {
		urlErr := fmt.Sprintf("Handle of invalid URL path. Path: %s", request.URL.Path)
		slog.Error(urlErr)
		writer.WriteHeader(http.StatusBadRequest)
		_, err := writer.Write([]byte(urlErr))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		return
	}
	dashboardID := urlPath[4]
	timerangeFrom := getParameterFromRequest(request, "from", g.DefaultFrom)
	timerangeTo := getParameterFromRequest(request, "to", g.DefaultTo)
	texTemplate := getParameterFromRequest(request, "template", g.DefaultTemplate)
	renderCollapsed := getBoolParameterFromRequest(request, "renderCollapsed", g.RenderCollapsed)

	vars := url.Values{}
	for k, values := range request.URL.Query() {
		if strings.HasPrefix(k, "var-") {
			for _, value := range values {
				vars.Add(k, value)
			}
		}
	}

	authHeader, err := g.getAuthHeaderFromRequest(request)
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when getting authorization header. Error: %v", err))
		writer.WriteHeader(http.StatusUnauthorized)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		return
	}

	timestampFrom, err := timerange.RelativeTimeToTimestamp(startTime, timerangeFrom, "from")
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when converting parameter %q to timestamp. Error: %v", "from", err))
		writer.WriteHeader(http.StatusBadRequest)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		return
	}
	timestampTo, err := timerange.RelativeTimeToTimestamp(startTime, timerangeTo, "to")
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when converting parameter %q to timestamp. Error: %v", "to", err))
		writer.WriteHeader(http.StatusBadRequest)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		return
	}
	timerangeData := &timerange.TimerangeData{
		From:     timerangeFrom,
		To:       timerangeTo,
		DateFrom: timestampFrom,
		DateTo:   timestampTo,
	}
	requestId := generateUniqueRequestId(dashboardID, timerangeFrom, timerangeTo, !renderCollapsed)
	slog.Info(fmt.Sprintf("Generating report %q with parameters: dashboardId=%s, from=%v, to=%v, template=%s, vars=%s", requestId, dashboardID, timerangeFrom, timerangeTo, texTemplate, vars.Encode()))
	//generateReport as a job and return immediate requestId
	report, err := g.generateReport(dashboardID, timerangeData, texTemplate, vars, requestId, authHeader, renderCollapsed)
	duration := time.Since(startTime).String()
	writer.Header().Set("Duration", duration)
	slog.Info(fmt.Sprintf("The request %s took %s", request.RequestURI, duration))
	//writer.Write([]byte(requestId))
	if err != nil {
		slog.Error(fmt.Sprintf("Error occurred when generating report. Error: %v", err))
		writer.WriteHeader(http.StatusInternalServerError)
		_, err = writer.Write([]byte(err.Error()))
		if err != nil {
			slog.Error("Could not write response", "error", err)
		}
		return
	}
	writer.Header().Set("Content-Type", "application/pdf")
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", requestId))
	writer.WriteHeader(http.StatusOK)
	_, err = writer.Write(report)
	if err != nil {
		slog.Error("Could not write response", "error", err)
	}
}

func (g *GrafanaInstance) getDashboard(dashboardUID string, authHeader string, renderCollapsed bool) (*dashboard.StructuredDashboard, error) {
	urlString, err := url.JoinPath(g.Endpoint, "/api/dashboards/uid/", dashboardUID)
	if err != nil {
		return nil, fmt.Errorf("could not create URL for request Grafana dashboard :%w", err)
	}
	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request to get Grafana dashboard :%w", err)
	}
	req.Header.Set("Authorization", authHeader)
	res, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to Grafana failed: %w", err)
	}
	slog.Debug(fmt.Sprintf("Response %s %q received", http.MethodGet, urlString), "status", res.Status)
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get Grafana dashboard, status code = %v", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	//slog.Debug(fmt.Sprintf("Response body for '%s' received: %s", urlString, body))
	err = res.Body.Close()
	if err != nil {
		slog.Error("Could not close body response", "error", err)
	}

	var dashboardEntity *dashboard.DashboardEntity
	if err = json.Unmarshal(body, &dashboardEntity); err != nil {
		return nil, err
	}
	structured, err := dashboardEntity.GetStructuredDashboard(renderCollapsed)
	if err != nil {
		return nil, err
	}
	return structured, nil
}

type PanelRequestInfo struct {
	ImageName string
	URL       string
}

func (g *GrafanaInstance) getPanels(structuredDashboard *dashboard.StructuredDashboard, from string, to string, vars url.Values, requestId string, authHeader string) (bool, error) {
	panelRequestInfos, err := getPanelsURLs(g.Endpoint, structuredDashboard, from, to, vars)
	if err != nil {
		return false, err
	}

	attempts := 3
	var wg sync.WaitGroup
	wg.Add(len(panelRequestInfos))
	concurrentRequests := getMaxConcurrentRequests()
	concurrencyLimiter := make(chan struct{}, concurrentRequests)
	isFailed := false

	for _, requestInfo := range panelRequestInfos {
		concurrencyLimiter <- struct{}{}
		go func(panelInfo *PanelRequestInfo) {
			defer func() {
				<-concurrencyLimiter
				wg.Done()
			}()
			if isFailed {
				return
			}
			for i := 0; i < attempts; i++ {
				err := g.requestAndSaveGetPanel(panelInfo.URL, panelInfo.ImageName, requestId, authHeader)
				if err != nil {
					slog.Error(fmt.Sprintf("Error occurred when requesting for panel. The request will be sent again in 5 seconds: %s", err), "panelId", panelInfo.ImageName)
					time.Sleep(time.Second * 5)
					slog.Info(fmt.Sprintf("Requesting for the panel again. URL: %q. Remaining attempts: %v", panelInfo.URL, attempts-i), "panelId", panelInfo.ImageName)
				} else {
					break
				}
			}
			if err != nil {
				isFailed = true
			}
		}(requestInfo)
	}
	wg.Wait()
	close(concurrencyLimiter)
	if isFailed {
		err = fmt.Errorf("could not get all panels successfully")
		slog.Error(err.Error())
		return false, err
	}
	slog.Debug(fmt.Sprintf("All the panels successfully saved to tmp/%s/", requestId))
	return true, nil
}

func getPanelsURLs(grafanaEndpoint string, structuredDashboard *dashboard.StructuredDashboard, from string, to string, vars url.Values) ([]*PanelRequestInfo, error) {
	errGroup, _ := errgroup.WithContext(context.Background())

	mutex := sync.Mutex{}
	var panelRequestInfos []*PanelRequestInfo
	for _, rows := range structuredDashboard.Rows {
		for _, panel := range rows.Panels {
			panelc := panel
			errGroup.Go(func() error {
				varsLocal := url.Values{}
				for k, values := range vars {
					for _, value := range values {
						varsLocal.Add(k, value)
					}
				}
				varsLocal.Add("panelId", strconv.Itoa(panelc.Id))
				varsLocal.Add("theme", theme)
				varsLocal.Add("from", from)
				varsLocal.Add("to", to)
				//add width and height
				var width, height int
				height = panelc.GetPxHeight(screenResolutionWidth)
				width = panelc.GetPxWidth(screenResolutionWidth)

				varsLocal.Add("width", strconv.Itoa(width))
				varsLocal.Add("height", strconv.Itoa(height))

				urlString, err := url.JoinPath(grafanaEndpoint, "/render/d-solo/", structuredDashboard.Uid, structuredDashboard.Slug)
				if err != nil {
					err = fmt.Errorf("could not create URL for request Grafana panel :%w", err)
				}
				mutex.Lock()
				panelRequestInfos = append(panelRequestInfos, &PanelRequestInfo{
					URL:       fmt.Sprintf("%s?%s", urlString, varsLocal.Encode()),
					ImageName: fmt.Sprintf("%s.png", strconv.Itoa(panelc.Id)),
				})
				mutex.Unlock()
				return err
			})
		}
	}
	if err := errGroup.Wait(); err != nil {
		return nil, err
	}
	return panelRequestInfos, nil
}

func (g *GrafanaInstance) requestAndSaveGetPanel(urlString string, imageName string, requestId string, header string) error {
	req, err := http.NewRequest(http.MethodGet, urlString, nil)
	if err != nil {
		return fmt.Errorf("could not create request to get Grafana panel :%w", err)
	}
	slog.Info(fmt.Sprintf("Requesting panel by url: %s", req.URL))
	req.Header.Set("Authorization", header)
	res, err := g.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request to Grafana failed: %w", err)
	}
	slog.Info(fmt.Sprintf("Response %s %q received", http.MethodGet, urlString), "status", res.Status)
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get Grafana panel: Status code is %v", res.StatusCode)
	}

	panelsDirPath := getPanelsDirPath(requestId)
	if err = os.MkdirAll(panelsDirPath, 0777); err != nil {
		return fmt.Errorf("could not create directory for panel on path %q. Error: %w", panelsDirPath, err)
	}
	fullFilePath := path.Join(panelsDirPath, imageName)
	f, err := os.Create(fullFilePath)
	if err != nil {
		return fmt.Errorf("could not create file %q. Error: %w", imageName, err)
	}
	defer f.Close()
	_, err = f.ReadFrom(res.Body)
	if err != nil {
		return fmt.Errorf("could not save image of panel from response to file %q. Error: %w", imageName, err)
	}
	slog.Info("Panel successfully saved to file", "path", fullFilePath)
	return nil
}

func getPanelsDirPath(requestId string) string {
	return path.Join(os.TempDir(), requestId)
}

func (g *GrafanaInstance) getAuthHeaderFromRequest(request *http.Request) (string, error) {
	authHeader := request.Header.Get("Authorization")
	if authHeader == "" {
		if g.Credentials != "" {
			data, err := os.ReadFile(g.Credentials)
			if err != nil {
				return "", err
			}
			var creds Credentials
			err = yaml.Unmarshal(data, &creds)
			if err != nil {
				return "", err
			}
			return creds.getAuthHeader()
		} else {
			return "", fmt.Errorf("credentials are not provided")
		}
	}
	return authHeader, nil
}
func (creds *Credentials) getAuthHeader() (string, error) {
	var authHeader string
	if creds.Token != "" {
		authHeader = fmt.Sprintf("Bearer %s", creds.Token)
	} else if creds.User != "" && creds.Password != "" {
		auth := fmt.Sprintf("%s:%s", creds.User, creds.Password)
		authHeader = fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth)))
	} else {
		return "", fmt.Errorf("credentials are not provided")
	}
	return authHeader, nil
}

func (g *GrafanaInstance) getAuthHeaderFromParameters(user, password, token string) (string, error) {
	creds := &Credentials{
		User:     user,
		Password: password,
		Token:    token,
	}
	return creds.getAuthHeader()
}

func getParameterFromRequest(r *http.Request, name string, defaultValue string) string {
	if len(name) != 0 {
		if r.URL.Query().Has(name) {
			return r.URL.Query().Get(name)
		} else {
			return defaultValue
		}
	}
	return ""
}

func getBoolParameterFromRequest(r *http.Request, name string, defaultValue bool) bool {
	if len(name) != 0 {
		if r.URL.Query().Has(name) {
			value, err := strconv.ParseBool(r.URL.Query().Get(name))
			if err != nil {
				return defaultValue
			}
			return value
		} else {
			return defaultValue
		}
	}
	return false
}

func getMaxConcurrentRequests() int {
	maxRequestsEnv, found := os.LookupEnv("MAX_CONCURRENT_RENDER_REQUESTS")
	if found {
		maxRequests, err := strconv.Atoi(maxRequestsEnv)
		if err != nil {
			slog.Warn("Could not parse MAX_CONCURRENT_RENDER_REQUESTS", "error", err.Error())
		} else {
			return maxRequests
		}
	}
	return 4
}
