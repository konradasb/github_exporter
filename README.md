
# Github exporter for Prometheus

[![main](https://github.com/konradasb/github_exporter/actions/workflows/main.yml/badge.svg?branch=master)](https://github.com/konradasb/github_exporter/actions/workflows/main.yml)

This is a Github exporter for Prometheus metrics exposed by Github API. Written in Go with pluggable metrics collectors.

## Status

Project is highly development, only a limited amount of collectors are implemented, starting from the ones that are needed the most at the time.

## Features

* Pluggable collectors, easy to integrate
* Rate limit implementation as recommended by [Github](https://docs.github.com/en/rest/guides/best-practices-for-integrators#dealing-with-rate-limits):
  * Response caching / Response revalidation
  * Conditional requests
  * Burst safeguards
  * Throttling

## Usage

For more detailed usage instructions and flags, check out the help (`--help`) flag.

```bash
export GH_PRIVATE_KEY=""
export GH_APP_ID=""
export GH_INS_ID=""

./exporter --help
Prometheus exporter for Github metrics.
Written in Go, with love ❤️

Usage:
  exporter [flags]

Flags:
      --host string                Host on which to expose metrics (HOST) (default "0.0.0.0")
      --port string                Port on which to expose metrics (PORT) (default "9042")
      --log-level string           Output log level severity (LOG_LEVEL) (default "debug")
      --web-metrics-path string    Path to HTTP metrics (WEB_METRICS_PATH) (default "/metrics")
      --web-healthz-path string    Path to HTTP healthz (WEB_HEALTHZ_PATH) (default "/healthz")
      --web-version-path string    Path to HTTP version (WEB_VERSION_PATH) (default "/version")
      --gh-organizations strings   List of Github organizations to scrape (GH_ORGANIZATIONS)
      --gh-repositories strings    List of Github repositories to scrape (GH_REPOSITORIES)
      --gh-private-key string      Github App Private Key (required) (GH_PRIVATE_KEY)
      --gh-app-id int              Github App application ID (required) (GH_APP_ID)
      --gh-ins-id int              Github App instalation ID (required) (GH_INS_ID)
  -h, --help                       help for exporter
  -v, --version                    version for exporter
```

## Usage (Docker)

You can run this exporter using a Docker image as well. Example:

```bash
docker pull konradasb/github_exporter
docker run -d -p 9024:9024 \
  -e GH_PRIVATE_KEY="" \
  -e GH_APP_ID="" \
  -e GH_INS_ID="" \
  konradasb/github_exporter [<flags>]
```

## Environment variables

These environment variables (or their `--flag` counterparts) are required:

* GH_PRIVATE_KEY
* GH_APP_ID
* GH_INS_ID

For more information on Github App(s) see [here](https://docs.github.com/en/developers/apps/building-github-apps)

## Configuration

Example Prometheus scrape job configuration:

```yaml
scrape_configs:
  - job_name: github_exporter
    scrape_interval: 60s
    scrape_timeout: 30s
    static_configs:
      - targets:
        - 127.0.0.1:9024
```

Adjust `scrape_interval` and `scrape_timeout` as needed. It might not be the right values depending on configuration - count of organizations, repositories, cache expiration times and etc.

## Collectors

List of collectors, descriptions and wether they are enabled by default

  Name    |             Description              | Enabled
----------|--------------------------------------|----------
actions   | collector for Github Actions service | true
ratelimit | collector for Github ratelimits      | true

## Development

Prerequisites:

* Go >= 1.17

Building:

```bash
git clone https://github.com/konradasb/github_exporter.git
cd github_exporter
make build
./github_exporter --help
```

## Contributing

Pull requests are always welcome. For any major changes, open an issue discussing the changes first before opening a pull request.

Update tests as appropriate.

## License

[License - MIT](https://choosealicense.com/licenses/mit/)
