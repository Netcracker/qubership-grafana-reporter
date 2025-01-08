# Grafana-reporter

This application is an HTTP server that generates Grafana dashboard reports and saves it in PDF format.
It provides REST API and parameters to get dashboard data that you need.

## Table of Content

* [Grafana-reporter](#grafana-reporter)
  * [Table of Content](#table-of-content)
  * [Overview](#overview)
  * [Repository structure](#repository-structure)
  * [Configuration](#configuration)
    * [Environment variables](#environment-variables)
    * [Command line arguments](#command-line-arguments)
      * [Templates](#templates)
      * [Default time range](#default-time-range)
  * [How to start](#how-to-start)
    * [Build](#build)
    * [Definition of done](#definition-of-done)
    * [Deploy](#deploy)
      * [Prerequisites](#prerequisites)
      * [Local](#local)
        * [Run as job in Docker container](#run-as-job-in-docker-container)
          * [Mounts](#mounts)
        * [Run as REST API Service](#run-as-rest-api-service)
          * [Authentication](#authentication)
          * [Query parameters](#query-parameters)
          * [Time range](#time-range)
          * [Variables](#variables)
          * [Template](#template)
      * [Deploy with helm](#deploy-with-helm)
    * [How to debug](#how-to-debug)
    * [How to troubleshoot](#how-to-troubleshoot)
      * [Rendering errors](#rendering-errors)
  * [CI/CD](#cicd)
  * [Evergreen strategy](#evergreen-strategy)
  * [Useful links](#useful-links)

## Overview

The application is useful in cases when you need to attach or share Grafana visualizations of metrics, and you want to
make the visualizations available even if data rotates or not available anymore.
Grafana-reporter allows you to generate report from Grafana dashboard with panels that shows data in the time range and
parameters you need.

There is a brief description how Grafana-reporter works. It is a RESTful application that accepts Grafana dashboard UID,
time range, dashboard variables. It gets Grafana dashboard information via `/api/dashboards/uid/{uid}`. The response
includes information about panels on the dashboard. The application sends requests to [grafana-image-renderer]
and gets rendered panels with data in FullHD resolution.
For PDF document generation Grafana-reporter uses tex command line tools and tex templates. It inserts in tex template
the panels and then generates PDF document according to tex file. More information about templates can be found [here](docs/public/configuration.md/#template)

## Repository structure

* `./docs` - any documentation related to grafana-reporter
* `./dashboard` - main structured entities for dashboard generation
* `./handle` - REST API registration
* `./report` - report rendering logic
* `./templates` - default TeX templates for reports
* `./timerange` - Grafana timeranges parsing logic
* `./main.go` - application entrypoint

Files for microservice build:

* `./Dockerfile` - to build Docker image

## Configuration

### Environment variables

There is a list of environment variables:

<!-- markdownlint-disable line-length -->
| Name                             | Description                                                                                                                       | Default |
| -------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | ------- |
| `MAX_CONCURRENT_RENDER_REQUESTS` | Maximum concurrent requests to grafana-image-renderer to request panels at the same time. It is not recommended to set high value | 4       |
| `SAVE_TEMP_IMAGES`               | By default all panels images after report generated will be deleted. To save images set `true`                                    | false   |
<!-- markdownlint-enable line-length -->

### Command line arguments

<!-- markdownlint-disable line-length -->
| Name               | Mandatory | Description                                                                         | Default value                  |
| ------------------ | --------- | ----------------------------------------------------------------------------------- | ------------------------------ |
| logLevel           | no        | Log level of the application.                                                       | info                           |
| grafana            | yes       | Grafana endpoint to get dashboard information from.                                 | localhost                      |
| dashboard          | yes       | Dashboard UID to generate report for.                                               |                                |
| user               | yes       | Credentials for Grafana user. You can set basic auth credentials or token (api key) |                                |
| password           | yes       | Credentials for Grafana user. You can set basic auth credentials or token (api key) |                                |
| token              | yes       | Credentials for Grafana user. You can set basic auth credentials or token (api key) |                                |
| vars               | no        | Dashboard variables separated by `&`.                                               |                                |
| insecureSkipVerify | no        | Verify Grafana certificates or not.                                                 | false                          |
| ca                 | no        | Name of Certificate Authority file                                                  | /grafana/certificates/ca.pem   |
| cert               | no        | Name of public Certificate file                                                     | /grafana/certificates/cert.crt |
| pKey               | no        | Name of private key file                                                            | /grafana/certificates/cert.key |
| template           | no        | Tex Template name to layout panels by default.                                      | simpleTemplate                 |
| defaultFrom        | no        | Time range begin of report.                                                         | now-30m                        |
| defaultTo          | no        | Time range end of report.                                                           | now                            |
<!-- markdownlint-enable line-length -->

#### Templates

There are two types of pre-defined tex templates for different purposes:

* [simpleTemplate](./templates/simpleTemplate) - The standard size of page (A4), each panel is placed under the
previous. This template can be used in case if you need to print the data.
* [gridTemplate](./templates/gridTemplate) - (default) The template copies layout of panels in the original Grafana dashboard.
   The size of the page tries to render panel in a beautified way.
* [pngTemplate](./templates/pngTemplate) - The same as gridTemplate but returns file in png format.

Also, you can use your own custom tex template as default. To do this, place your tex template under
`/templates/custom/` directory and set the name of the file to `template` parameter.

#### Default time range

Parameters `defaultFrom` and `defaultTo` are used when its are not set in the request to Grafana-reporter.
Its values can be timestamp or Grafana time range.

## How to start

If you are developer and need to make changes to this repository make sure that you read recommendations below.

### Build

Each time when you push your changes to repository CI pipeline is started.
Usually one of the latest step is application build and Docker image build.
So you do not need to run build job manually. You can find the image in `build-service` link in job.

If you need to build grafana-reporter manually, you can:

1. Run CI pipeline.
2. Run only build job.
   Parameters:

   * `REPOSITORY_NAME` - `qubership-grafana-reporter`
   * `LOCATION` - your branch

### Definition of done

After you made changes related to a task do next steps:

1. Check if there are any dependencies versions that can be upgraded in `go.mod`. Upgrade if it is possible.
2. Create tests if you modified behavior of application or fixed a bug (especially if it is not covered by tests).
3. Build grafana-reporter Docker image.
4. Check that all pipeline is succeeded (linter, build, deploy & test jobs are passed).
5. Deploy grafana-reporter
   Check that your feature works fine in possible cases.
6. Create merge request using merge request template. Name your MR `<TICKET-ID>: <SHORT-DESCRIPTION>`. Describe and
   explain your changes in MR. There you can add any information about the changes (how it was tested, details
   of aim of changes, examples and so on) to make it clear to the reviewers.

### Deploy

Grafana-reporter can be installed as a part of Platform Monitoring. It is included in the manifest.

#### Prerequisites

It is required to have access to Grafana with grafana-image-renderer installed. See installation documentation provided [above](#deploy).

There are some useful configurations for Grafana and
grafana-image-renderer:

```yaml
grafana:
  install: true
  extraVars:
    GF_RENDERING_CONCURRENT_RENDER_REQUEST_LIMIT: "90"
    GF_RENDERING_RENDER_KEY_LIFETIME: "10m"
    GF_ALERTING_CONCURRENT_RENDER_LIMIT: 7
    GF_PLUGIN_GRAFANA_IMAGE_RENDERER_RENDERING_CLUSTERING_TIMEOUT: 70
    GF_PLUGIN_GRAFANA_IMAGE_RENDERER_RENDERING_MODE: "clustered"
    GF_UNIFIED_ALERTING_SCREENSHOTS_CAPTURE_TIMEOUT: "20s"
    GF_ALERTING_NOTIFICATION_TIMEOUT_SECONDS: "60"
  imageRenderer:
    install: true
    resources:
      requests:
        cpu: 300m
        memory: 800Mi
      limits:
        cpu: 800m
        memory: 1500Mi
    extraEnvs:
      LOG_LEVEL: info
      RENDERING_ARGS: --no-sandbox,--disable-setuid-sandbox,--disable-dev-shm-usage,--disable-accelerated-2d-canvas,--disable-gpu,--window-size=1920x1080
      IGNORE_HTTPS_ERRORS: true
      RENDERING_CLUSTERING_TIMEOUT: 100
      RENDERING_MODE: "clustered"
  reporter:
    install: true
    ingress:
      install: true
      host: grafana-reporter.cloud.org
      annotations:
        nginx.ingress.kubernetes.io/proxy-connect-timeout: '300'
        nginx.ingress.kubernetes.io/proxy-read-timeout: '300'
        nginx.ingress.kubernetes.io/proxy-send-timeout: '300'
```

#### Local

##### Run as job in Docker container

Grafana-reporter can be launched directly as docker container from your machine.
To run grafana-reporter, execute the command:

```bash
docker run -d --name grafana-reporter \
  -v <path_to_save_report>:/reports:rw \
  -v <grafana_certificates>:/grafana/certificates/:ro \
  -v <path_to_custom_template_dir>:/templates/custom:ro \
  <docker_image> <parameters>
```

The list of parameters described [below](#command-line-parameters). About mounts you can read [here](#mounts).

The example:

```bash
docker run -d --name grafana-reporter \
  -v <path_to_save_report>:/reports:rw \
  -v <grafana_certificates>:/grafana/certificates:ro \
  -v <path_to_custom_template_dir>:/templates/custom:ro \
  grafana-reporter:latest -logLevel debug -grafana https://10.10.10.10/grafana \
  -dashboard monitoring-k8s-pod-resources -token glsa_9244xlVFZK0j8Lh4fU8Cz6Z5tO664zIi_7a762939 \
  -template gridTemplate -defaultFrom now-15m -defaultTo now -vars "var-datasource=default&var-cluster=&var-namespace=ingress-nginx&var-pod=ingress-nginx-controller-b25hj"
```

###### Mounts

<!-- markdownlint-disable line-length -->
| Mount point            | Mandatory | Access | Description                                                                               |
| ---------------------- | --------- | ------ | ----------------------------------------------------------------------------------------- |
| /reports               | yes       | rw     | Directory where generated report will be saved                                            |
| /grafana/certificates/ | no        | ro     | Certificates must be placed in the directory in `ca.pem`, `cery.crt` and `cert.key` files |
| /templates/custom      | no        | ro     | If you want to create report generated on custom template, you should mount directory     |
<!-- markdownlint-enable line-length -->

##### Run as REST API Service

Grafana-reporter can be launched directly as docker container from your machine as HTTP server.
It will listen for the requests to generate reports.
To run grafana-reporter, execute the command:

```bash
docker run -d --name grafana-reporter \
  -v <path_to_save_report>:/reports:rw \
  -v <grafana_certificates>:/grafana/certificates/:ro \
  -v <path_to_custom_template_dir>:/templates/custom:ro \
  <docker_image> -httpServiceMode=true <parameters>
```

###### Authentication

To generate dashboard report execute the command with basic authentication:

```bash
curl http://<user>:<password>@<grafana_reporter>:<port>/api/v1/report/<uid> --output /report.pdf
```

To generate dashboard report execute the command with API Key token:

```bash
curl http://<grafana_reporter>:<port>/api/v1/report/<uid> -H "Authorization: Bearer <api_key>" --output /report.pdf
```

###### Query parameters

There are parameters that you can add to query to customize report results:

<!-- markdownlint-disable line-length -->
| Name            | Description                                                                                    | If does not set                              |
| --------------- | ---------------------------------------------------------------------------------------------- | -------------------------------------------- |
| template        | Tex Template name to layout panels.                                                            | Value of application parameter `template`    |
| from            | Time range of the request to render panels data.                                               | Value of application parameter `defaultFrom` |
| to              | Time range of the request to render panels data.                                               | Value of application parameter `defaultTo`   |
| renderCollapsed | Enable rendering collapsed panels. If true, all collapsed panels will be expanded and rendered | false                                        |
| vars-*          | Grafana variables                                                                              | -                                            |
<!-- markdownlint-enable line-length -->

###### Time range

Its values can be:

* timestamp, for example `from=1706562000000&to=1706734799000`,
* Grafana time range, for example `from=now-30m&to=now-15m`.

For example:

```bash
curl 'http://<grafana_reporter>:<port>/api/v1/report/monitoring-k8s-cluster-overview?from=now-30m&to=now-15m' --output report.pdf
```

```bash
curl 'http://<grafana_reporter>:<port>/api/v1/report/monitoring-k8s-cluster-overview?from=1706562000000&to=1706734799000' --output report.pdf
```

Parameters `defaultFrom` and `defaultTo` are used when its are not set in the request to Grafana-reporter.

###### Variables

When you need to filter the data, you can set variables as it is set in Grafana, for example:

```bash
curl 'http://<user>:<password>@<grafana_reporter>:<port>/api/v1/report/api/v1/report/monitoring-govm-processes?var-cluster=&var-namespace=monitoring&var-pod=node-exporter-pct2b&var-container=node-exporter' --output /report.pdf
```

###### Template

There is a default template set in the parameters of application, but if you need to render PDF report in a certain
tex template, you can set the template name to use.

For example:

```bash
curl 'http://<user>:<password>@<grafana_reporter>:<port>/api/v1/report/<uid>?template=simpleTemplate' --output /report.pdf
```

You can learn about templates [here](configuration.md#template).

#### Deploy with helm

To deploy grafana-reporter clone repository. Modify locally grafana subchart [`values.yaml`]
and any other parameters that are necessary for deploy.

Set parameters:

```yaml
# charts/grafana-operator/values.yaml
install: true
image: <grafana-image>
operator:
  install: true
  image: <grafana-operator-image>
imageRenderer:
  install: true
  image: <grafana-image-renderer-image>
reporter:
  install: true
  image: <grafana-reporter-image>
  args:
    - "-grafana=http://grafana-service:3000/"
```

To uninstall deployment run command:

```helm
helm uninstall <any-release-name> --namespace <namespace>
```


### How to debug

You can debug grafana-reporter locally with default or custom parameters in your IDE.
The only thing that you need to cnfigure cli flags and have the instance of
Grafana with grafana-image-renderer installed (cloud or VM).

### How to troubleshoot

There are no well-defined rules for troubleshooting, as each task is unique, but there are some tips that can do:

* See deployment parameters and cli flags
* See logs of grafana-reporter
* See logs and configuration of grafana-image-renderer
* See logs and configuration (for example, timeouts) of Grafana

#### Rendering errors

If panels got successfully from grafana-image-renderer, but report generation failed with an error:
`Error occurred when generating tex file`, it means that something when wrong with `.tex` file.
To investigate why the error happened you can see `/reports/<dashboarduid-timerange>.log`.

More likely there will be line like

```bash
l.19 ...degraphics[width=0.995\textwidth]{185.png}
```

It means that the error occurred in line 19. To see `.tex` file generated by grafana-reporter look at
`/reports/<dashboarduid-timerange>.tex`.

## CI/CD

The main CI/CD pipeline designed to automize all basic developer routine start from code quality and finish with
deploying to stand k8s cluster. There are described stages in pipeline:

1. `lint` - stage with jobs that run different linter to check code & documentation.
2. `tests` - stage with jobs with units tests and other go code checks.
3. `build` - stage with jobs that build docker image of grafana-reporter.

## Evergreen strategy

To keep the component up to date, the following activities should be performed regularly:

* Vulnerabilities fixing, dependencies update
* Bug-fixing, improvement and feature implementation

## Useful links

* [Configuration parameters](docs/public/configuration.md)
* [Usage guide](docs/public/usage.md)
* [Swagger 2.0 REST API](docs/swagger.json)
* [Usage and examples](docs/public/usage.md)
