# Github exporter for Prometheus

This is a Github exporter for Prometheus metrics exposed by Github API. Written in Go with pluggable metrics collectors.

## Status

Project is highly development, only a limited amount of collectors are implemented, starting from the ones that are needed the most at the time.

## Features

* Rate limit implementation as recommended by [Github](https://docs.github.com/en/rest/guides/best-practices-for-integrators#dealing-with-rate-limits)
* Pluggable collectors, easy to integrate
* Configurable caching of responses

## Usage

For more detailed usage instructions and flags, check out the help (`--help`) flag.

```bash
export GITHUB_PRIVATE_KEY=""
export GITHUB_APP_ID=""
export GITHUB_INS_ID=""

./exporter [<flags>]
```

## Usage (Docker)

You can run this exporter using a Docker image as well. Example:

```bash
docker pull konradasb/github_exporter
docker run -d -p 9024:9024 \
  -e GITHUB_PRIVATE_KEY="" \
  -e GITHUB_APP_ID="" \
  -e GITHUB_INS_ID="" \
  konradasb/github_exporter [<flags>]
```

## Environment variables

These environment variables (or their `--flag` counterparts) are required:

* GITHUB_PRIVATE_KEY
* GITHUB_APP_ID
* GITHUB_INS_ID

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

  Name  |             Description              | Enabled
--------|--------------------------------------|----------
actions | collector for Github Actions service | true

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
