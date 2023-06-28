# Sorarqube badge proxy
[![Go Report Card](https://goreportcard.com/badge/github.com/shdubna/sonar-badge-proxy)](https://goreportcard.com/report/github.com/shdubna/sonar-badge-proxy)
[![GitHub CodeQL](https://github.com/shdubna/sonar-badge-proxy/workflows/CodeQL/badge.svg)](https://github.com/shdubna/sonar-badge-proxy/actions?query=workflow%3CodeQL)
[![GitHub Release](https://github.com/shdubna/sonar-badge-proxy/workflows/Release/badge.svg)](https://github.com/shdubna/sonar-badge-proxy/actions?query=workflow%3ARelease)
[![GitHub license](https://img.shields.io/github/license/shdubna/sonar-badge-proxy.svg)](https://github.com/shdubna/sonar-badge-proxy/blob/main/LICENSE)
[![GitHub tag](https://img.shields.io/github/v/tag/shdubna/sonar-badge-proxy?label=latest)](https://github.com/shdubna/sonar-badge-proxy/releases)

This is simple reverse proxy that allow to get badges from sonarqube using user token. 
For example you with this proxy you can configure badges on group level in Gitlab.

## Flags
| Flag              | Description                                       | Type     | Default                |
|:------------------|:--------------------------------------------------|:---------|:-----------------------| 
| `listen_address`  | Address to listen requests                        | `string` | `:8080`                |
| `listen_endpoint` | Path under which proxy response to SonarQube      | `string` | `/proxy/bages/measure` |
| `insecure`        | Allow insecure requests                           | `bool`   | `false`                |
| `proxy_token`     | Check authorization by proxy_token param in query | `string` | ``                     |
| `debug`           | Enable debug logging                              | `bool`   | `false `               |

## How to use
1. Create sonar token
2. Run and configure proxy
3. Send request to proxy
  ```bash
  curl <sonar-badge-proxy>/<listen_endpoint>?project=<project_name>&token=<sonar user token>&metric=alert_status
  ```

