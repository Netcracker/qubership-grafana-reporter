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

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Netcracker/grafana-reporter/handle"
	"github.com/Netcracker/grafana-reporter/report"
)

// @title				Grafana Reporter REST API
// @version			1.0
// @description		This document describes REST API for Grafana Reporter.
// @license.name		Qubership
// @license.url		https://www.qubership.org/
// @tag.name			Generate
// @tag.description	Create reports of Grafana dashboard with the set of parameters
// @tag.name			General
// @tag.description	Get application information
// @host				GRAFANA_REPORTER:8881
func main() {
	logLevel := flag.String("logLevel", "info", "Log level of the application")
	port := flag.String("port", ":8881", "Application port")
	grafanaAddress := flag.String("grafana", "http://grafana-service:3000", "Grafana endpoint to get dashboard information from")
	credentialsFile := flag.String("credentials", "/grafana/auth/credentials.yaml", "Path to yaml file that contains credentials for Grafana (for basic or token authentication)")
	renderCollapsed := flag.Bool("renderCollapsed", false, "Enable rendering collapsed panels. If true, all collapsed panels will be expanded and rendered")
	defaultTemplate := flag.String("template", "gridTemplate", "Tex Template name to layout panels by default")
	defaultFrom := flag.String("defaultFrom", "now-30m", "Default time range will be used if the parameter is not set in request parameters")
	defaultTo := flag.String("defaultTo", "now", "Default time range will be used if the parameter is not set in request parameters")
	templatesPath := flag.String("templates", "templates", "Default templates path")
	customTemplatesPath := flag.String("customTemplates", "templates/custom", "Custom templates path")
	insecureSkipVerify := flag.Bool("insecureSkipVerify", true, "Verify Grafana certificates or not")
	ca := flag.String("ca", "/grafana/certificates/ca.pem", "Name of Certificate Authority file. It should be mounted in /grafana/certificates/ directory")
	crt := flag.String("cert", "/grafana/certificates/cert.crt", "Name of public Certificate file. It should be mounted in /grafana/certificates/ directory")
	pKey := flag.String("pKey", "/grafana/certificates/cert.key", "Name of private key file. It should be mounted in /grafana/certificates/ directory")
	dashboardUID := flag.String("dashboard", "", "Dashboard UID to generate report.")
	// parameters only for command line execution
	vars := flag.String("vars", "", "All variables separated by `&`")
	user := flag.String("user", "", "Credentials for Grafana user")
	password := flag.String("password", "", "Credentials for Grafana user")
	token := flag.String("token", "", "Credentials for Grafana user")

	httpServiceMode := flag.Bool("httpServiceMode", false, "Mode of the application. It can be run as HTTP service or make one report and return")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: getLogLevel(*logLevel), ReplaceAttr: replaceAttrs, AddSource: true}))

	slog.SetDefault(logger)

	slog.Info(fmt.Sprintf("Grafana address: %s", *grafanaAddress))
	slog.Debug(fmt.Sprintf("The parameter renderCollapsed is %t", *renderCollapsed))

	templates, err := readTemplates(*defaultTemplate, *templatesPath, *customTemplatesPath)
	if err != nil {
		slog.Error(fmt.Sprintf("Error happened when reading available templates: %s", err))
		os.Exit(1)
	}

	tlsConfig, err := getTLSConfig(*insecureSkipVerify, *ca, *crt, *pKey)
	if err != nil {
		slog.Error(fmt.Sprintf("Error happened when getting TLS certificates. Error: %s", err))
		os.Exit(1)
	}
	if *httpServiceMode {
		baseCtx, cancel := context.WithCancel(context.Background())
		srvBaseCtx := context.WithValue(baseCtx, ContextKey, ContextMain)
		srv := &http.Server{
			Addr:              *port,
			Handler:           handle.RegisterEndpoints(*grafanaAddress, *credentialsFile, templates, *defaultTemplate, *defaultFrom, *defaultTo, *renderCollapsed, tlsConfig),
			TLSConfig:         nil,
			ReadHeaderTimeout: time.Second * 15,
			WriteTimeout:      time.Minute * 15,
			ReadTimeout:       time.Second * 15,
			IdleTimeout:       time.Minute * 15,
			BaseContext: func(_ net.Listener) context.Context {
				return srvBaseCtx
			},
		}

		slog.Info("Starting HTTP Server...")

		go func() {
			if err := srv.ListenAndServe(); err != nil {
				if !errors.Is(err, http.ErrServerClosed) {
					slog.Error(fmt.Sprintf("Error Occurred while starting HTTP Server: %s", err))
					cancel()
				}
			}
		}()

		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		slog.Info("Stopping application")

		if err = Shutdown(srvBaseCtx, 30*time.Second,
			func(ctx context.Context) {
				if err = srv.Shutdown(ctx); err != nil {
					slog.Error("Failed to shut down HTTP server gracefully in time", "error", err)
					slog.Info("Force closing HTTP server", "error", srv.Close())
				}
				slog.Info("HTTP server is shut down")
			},
		); err != nil {
			slog.Error("Failed to shutdown gracefully", "error", err)
		}
	} else {
		err = report.RunGenerateReport(*grafanaAddress, *credentialsFile, *dashboardUID, *vars, templates, *defaultTemplate, *defaultFrom, *defaultTo, *renderCollapsed, tlsConfig, *user, *password, *token)
		if err != nil {
			slog.Error(fmt.Sprintf("Error occurred while generating report: %s", err))
			os.Exit(1)
		}
	}
}
func getTLSConfig(insecureSkipVerify bool, ca string, crt string, pKey string) (*tls.Config, error) {
	var err error
	var tlsConf *tls.Config

	if insecureSkipVerify {
		tlsConf = &tls.Config{
			InsecureSkipVerify: true,
		}
		return tlsConf, err
	} else {
		caCert, err := os.ReadFile(ca)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to read Certificate Authority file. Error: %s", err))
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		ok := caCertPool.AppendCertsFromPEM(caCert)
		if !ok {
			err = fmt.Errorf("could not parse Certificate from PEM")
			slog.Error(fmt.Sprintf("Failed to parse Certificate Authority. Error: %v", err))
			return nil, err
		}

		certCrt, certErr := os.ReadFile(crt)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to read Certificate file. Error: %s", err))
		}
		pKey, pKeyErr := os.ReadFile(pKey)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to read Private Key file. Error: %s", err))
		}

		if len(pKey) > 0 && len(certCrt) > 0 && certErr == nil && pKeyErr == nil {
			var clientCert tls.Certificate
			clientCert, err = tls.X509KeyPair(certCrt, pKey)
			if err != nil {
				slog.Error(fmt.Sprintf("Failed to parse Certificate. Error: %v", err))
				return nil, err
			}
			tlsConf = &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{clientCert},
			}
		} else {
			tlsConf = &tls.Config{
				RootCAs: caCertPool,
			}
		}
	}
	return tlsConf, nil
}

func getLogLevel(logLevel string) slog.Level {
	var lvl slog.LevelVar
	if err := lvl.UnmarshalText([]byte(logLevel)); err != nil {
		return slog.LevelInfo
	}
	return lvl.Level()
}

// readTemplate reads all templates from directories to map
func readTemplates(defaultTemplate string, templatesPath string, customTemplatesPath string) (map[string][]byte, error) {
	var templates = map[string][]byte{}
	paths := []string{templatesPath, customTemplatesPath}
	for _, dirPath := range paths {
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				template, err := os.ReadFile(path.Join(dirPath, entry.Name()))
				if err != nil {
					return nil, err
				}
				templates[entry.Name()] = template
			}
		}
	}
	if templates[defaultTemplate] == nil {
		return nil, fmt.Errorf("could not find default template in directories %s and %s", templatesPath, customTemplatesPath)
	}
	return templates, nil
}

func replaceAttrs(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.TimeKey && len(groups) == 0 {
		actualTime := a.Value.Any().(time.Time)
		timeValue := actualTime.Format("2006-01-02T15:04:05.999")
		return slog.Attr{Key: slog.TimeKey, Value: slog.StringValue(timeValue)}
	}
	if a.Key == slog.SourceKey {
		source := a.Value.Any().(*slog.Source)
		source.File = filepath.Base(source.File)
		return slog.Attr{
			Key:   slog.SourceKey,
			Value: slog.StringValue(fmt.Sprintf("%s:%v", filepath.Base(source.File), source.Line)),
		}
	}
	return a
}
