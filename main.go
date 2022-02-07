package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v42/github"
	"github.com/gorilla/mux"
	"github.com/konradasb/github_exporter/collectors"
	"github.com/konradasb/github_exporter/log"
	"github.com/konradasb/github_exporter/transport"
	metrics "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	// defaultLogLevel defines the the default log level severity
	defaultLogLevel = "debug"

	// defaultWebListenHost defines the default HTTP server listen host
	defaultWebListenHost = "0.0.0.0"

	// defaultWebListenPort defines the default HTTP server listen port
	defaultWebListenPort = "9024"

	// defaultCollectors defines the deault enabled collectors
	defaultCollectors = "actions,ratelimit"
)

func main() {
	type options struct {
		logLevel            string
		collectors          string
		webMetricsPath      string
		webVersionPath      string
		webHealthzPath      string
		webListenHost       string
		webListenPort       string
		githubPrivateyKey   string
		githubAppID         int64
		githubInsID         int64
		githubOrganizations string
		githubRepositories  string
		serverReadTimeout   time.Duration
		serverIdleTimeout   time.Duration
		serverWriteTimeout  time.Duration
	}

	opts := &options{}

	kingpin.Flag("log.level", "Output log serverity").Default(defaultLogLevel).Envar("LOG_LEVEL").StringVar(&opts.logLevel)
	kingpin.Flag("web.listen.host", "Host on which to expose metrics").Default(defaultWebListenHost).Envar("WEB_LISTEN_HOST").StringVar(&opts.webListenHost)
	kingpin.Flag("web.listen.port", "Port on which to expose metrics").Default(defaultWebListenPort).Envar("WEB_LISTEN_HOST").StringVar(&opts.webListenPort)
	kingpin.Flag("web.metrics.path", "Path to HTTP metrics").Default("/metrics").Envar("WEB_METRICS_PATH").StringVar(&opts.webMetricsPath)
	kingpin.Flag("web.healthz.path", "Path to HTTP healthz").Default("/healthz").Envar("WEB_HEALTHZ_PATH").StringVar(&opts.webHealthzPath)
	kingpin.Flag("web.version.path", "Path to HTTP version").Default("/version").Envar("WEB_VERSION_PATH").StringVar(&opts.webVersionPath)
	kingpin.Flag("collectors", "List of enabled collectors").Default(defaultCollectors).Envar("COLLECTORS").StringVar(&opts.collectors)
	kingpin.Flag("server.read-timeout", "Server read timeout duration").Default("30s").Envar("SERVER_READ_TIMEOUT").DurationVar(&opts.serverReadTimeout)
	kingpin.Flag("server.idle-timeout", "Server idle timeout duration").Default("30s").Envar("SERVER_IDLE_TIMEOUT").DurationVar(&opts.serverIdleTimeout)
	kingpin.Flag("server.write-timeout", "Server write timeout duration").Default("30s").Envar("SERVER_WRITE_TIMEOUT").DurationVar(&opts.serverWriteTimeout)
	kingpin.Flag("github.organizations", "Github organizations to scrape").Default("").Envar("GITHUB_ORGANIZATIONS").StringVar(&opts.githubOrganizations)
	kingpin.Flag("github.repositories", "Github repositories to scrape").Default("").Envar("GITHUB_REPOSITORIES").StringVar(&opts.githubRepositories)
	kingpin.Flag("github.private-key", "Github App private key (required) (GITHUB_PRIVATE_KEY)").Required().Envar("GITHUB_PRIVATE_KEY").StringVar(&opts.githubPrivateyKey)
	kingpin.Flag("github.app-id", "Github App application ID (required) (GITHUB_APP_ID)").Required().Envar("GITHUB_APP_ID").Int64Var(&opts.githubAppID)
	kingpin.Flag("github.ins-id", "Github App instalation ID (required) (GITHUB_INS_ID)").Required().Envar("GITHUB_INS_ID").Int64Var(&opts.githubInsID)

	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger, err := log.NewZapLogger(opts.logLevel)
	if err != nil {
		fmt.Printf("error initializing logger: %v", err)
		return
	}

	rt, err := ghinstallation.New(
		http.DefaultTransport, opts.githubAppID, opts.githubInsID,
		[]byte(opts.githubPrivateyKey),
	)
	if err != nil {
		fmt.Printf("error initializing Github client transport: %v", err)
		return
	}

	client := github.NewClient(
		&http.Client{
			Transport: transport.NewTransport(rt, nil),
		},
	)

	c, err := collectors.NewGithubCollector(client, logger)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	registry := metrics.NewRegistry()
	err = registry.Register(c)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	template := `
	<html>
	<head><title>Github Exporter</title></head>
	<body>
	<h1>Github Exporter</h1>
	<p><a href="` + opts.webMetricsPath + `">Metrics</a></p>
	</body>
	</html>
	`

	router := mux.NewRouter()

	router.Handle(opts.webMetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	router.Handle("/", http.HandlerFunc(
		func(rw http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(rw, template)
		}),
	)
	router.Handle(opts.webHealthzPath, http.HandlerFunc(
		func(rw http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(rw, "OK")
		}),
	)
	router.Handle(opts.webVersionPath, http.HandlerFunc(
		func(rw http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(rw, "1.0.0")
		}),
	)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", opts.webListenHost, opts.webListenPort),
		WriteTimeout: opts.serverWriteTimeout,
		ReadTimeout:  opts.serverReadTimeout,
		IdleTimeout:  opts.serverIdleTimeout,
		Handler:      router,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		logger.Info(
			"starting server",
			zap.String("host", opts.webListenHost),
			zap.String("port", opts.webListenPort),
			zap.Error(err),
		)
		err := srv.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal(
				"error starting server",
				zap.String("host", opts.webListenHost),
				zap.String("port", opts.webListenPort),
				zap.Error(err),
			)
		}
	}()

	<-shutdown

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(ctx)
	if err != nil {
		logger.Fatal(
			"error stopping server",
			zap.String("host", opts.webListenHost),
			zap.String("port", opts.webListenPort),
			zap.Error(err),
		)
	}

}
