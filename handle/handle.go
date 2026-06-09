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

package handle

import (
	"crypto/tls"
	"github.com/Netcracker/grafana-reporter/report"
	"log/slog"
	"net/http"
)

func RegisterEndpoints(addr, credentialsFile string, templates map[string][]byte, defaultTemplate, defaultFrom, defaultTo string, renderCollapsed bool, tlsConfig *tls.Config) http.Handler {
	slog.Debug("Registering handlers...")
	mux := http.NewServeMux()

	transportConf := http.DefaultTransport.(*http.Transport).Clone()
	transportConf.TLSClientConfig = tlsConfig
	GrafanaInstance := &report.GrafanaInstance{
		DefaultTemplate: defaultTemplate,
		Templates:       templates,
		DefaultFrom:     defaultFrom,
		DefaultTo:       defaultTo,
		Endpoint:        addr,
		Credentials:     credentialsFile,
		RenderCollapsed: renderCollapsed,
		Client: http.Client{
			Timeout:   0,
			Transport: transportConf,
		},
	}
	mux.HandleFunc("/api/v1/report/", func(writer http.ResponseWriter, request *http.Request) {
		GrafanaInstance.HandleGenerateReport(writer, request)
	})
	mux.HandleFunc("/api/v1/templates", func(writer http.ResponseWriter, request *http.Request) {
		GrafanaInstance.HandleGetTemplatesList(writer)
	})
	mux.HandleFunc("/api/v1/template/", func(writer http.ResponseWriter, request *http.Request) {
		GrafanaInstance.HandleGetTemplate(writer, request)
	})
	mux.HandleFunc("/api/v1/defaults", func(writer http.ResponseWriter, request *http.Request) {
		GrafanaInstance.HandleGetDefaultParameters(writer)
	})

	return mux
}
