package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/v42/github"
	"github.com/gorilla/mux"
	"github.com/konradasb/github_exporter/build"
	"github.com/konradasb/github_exporter/collectors"
	"github.com/konradasb/github_exporter/log"
	"github.com/konradasb/github_exporter/transport"
	"github.com/konradasb/github_exporter/validators"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	c := &cobra.Command{
		Use:          "exporter",
		Short:        "Prometheus exporter for Github metrics.\nWritten in Go, with love ❤️",
		SilenceUsage: true,
		Args:         cobra.ExactArgs(0),
	}

	c.SetVersionTemplate(build.String())
	c.Version = build.Version

	// https://github.com/spf13/viper/issues/397
	cobra.OnInitialize(func() {
		c.Flags().VisitAll(func(f *pflag.Flag) {
			if viper.IsSet(f.Name) && viper.GetString(f.Name) != "" {
				if !f.Changed {
					c.Flags().Set(f.Name, viper.GetString(f.Name))
				}
			}
		})
	})

	c.Flags().SortFlags = false

	c.Flags().String("host", "0.0.0.0", "Host on which to expose metrics (HOST)")
	c.Flags().String("port", "9024", "Port on which to expose metrics (PORT)")
	c.Flags().String("log-level", "debug", "Output log level severity (LOG_LEVEL)")
	c.Flags().StringSlice("collectors", collectors.Collectors, "List of enabled collectors")
	c.Flags().String("web-metrics-path", "/metrics", "Path to HTTP metrics (WEB_METRICS_PATH)")
	c.Flags().String("web-healthz-path", "/healthz", "Path to HTTP healthz (WEB_HEALTHZ_PATH)")
	c.Flags().String("web-version-path", "/version", "Path to HTTP version (WEB_VERSION_PATH)")
	c.Flags().StringSlice("gh-organizations", []string{}, "List of Github organizations to scrape (GH_ORGANIZATIONS)")
	c.Flags().StringSlice("gh-repositories", []string{}, "List of Github repositories to scrape (GH_REPOSITORIES)")
	c.Flags().String("gh-private-key", "", "Github App Private Key (required) (GH_PRIVATE_KEY)")
	c.Flags().Int64("gh-app-id", 0, "Github App application ID (required) (GH_APP_ID)")
	c.Flags().Int64("gh-ins-id", 0, "Github App instalation ID (required) (GH_INS_ID)")

	c.Flags().AddFlagSet(collectors.ActionsFlagset)

	c.MarkFlagRequired("gh-private-key")
	c.MarkFlagRequired("gh-app-id")
	c.MarkFlagRequired("gh-ins-id")

	viper.BindPFlags(c.Flags())
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv()

	c.RunE = func(cmd *cobra.Command, args []string) error {
		logger, err := log.NewZapLogger(viper.GetString("log-level"))
		if err != nil {
			return errors.Wrap(err, "error initializing logger")
		}

		rt, err := ghinstallation.New(
			http.DefaultTransport, viper.GetInt64("gh-app-id"), viper.GetInt64("gh-ins-id"),
			[]byte(viper.GetString("gh-private-key")),
		)
		if err != nil {
			return errors.Wrap(err, "error initializing Github client transport")
		}

		// TODO: Make this configurable
		v := validators.NewRegexpValidator(
			map[*regexp.Regexp]time.Duration{
				regexp.MustCompile(`\/repos$`):           10 * time.Minute,
				regexp.MustCompile(`\/workflows$`):       10 * time.Minute,
				regexp.MustCompile(`\/actions/runners$`): 1 * time.Minute,
				regexp.MustCompile(`\/actions/runs$`):    5 * time.Minute,
			},
		)

		// TODO: Make this configurable
		transport := transport.NewTransport(rt).
			WithRatelimit().WithThrottle(nil).WithRevalidation(v).WithCache(nil)

		client := github.NewClient(
			&http.Client{
				Transport: transport,
			},
		)

		c, err := collectors.NewGithubCollector(client, logger)
		if err != nil {
			return err
		}

		registry := prometheus.NewRegistry()
		err = registry.Register(c)
		if err != nil {
			return err
		}

		template := `
		<html>
		<head><title>Github Exporter</title></head>
		<body>
		<h1>Github Exporter</h1>
		<p><a href="` + viper.GetString("web-metrics-path") + `">Metrics</a></p>
		</body>
		</html>
		`

		router := mux.NewRouter()

		router.Handle(viper.GetString("web-metrics-path"), promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		router.Handle("/", http.HandlerFunc(
			func(rw http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(rw, template)
			}),
		)
		router.Handle(viper.GetString("web-healthz-path"), http.HandlerFunc(
			func(rw http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(rw, "OK")
			}),
		)
		router.Handle(viper.GetString("web-version-path"), http.HandlerFunc(
			func(rw http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(rw, build.String())
			}),
		)

		srv := &http.Server{
			Addr:         fmt.Sprintf("%s:%s", viper.GetString("host"), viper.GetString("port")),
			WriteTimeout: 120 * time.Second,
			ReadTimeout:  120 * time.Second,
			IdleTimeout:  120 * time.Second,
			Handler:      router,
		}

		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

		go func() {
			logger.Info(
				"starting server",
				zap.String("host", viper.GetString("host")),
				zap.String("port", viper.GetString("port")),
				zap.Error(err),
			)
			err := srv.ListenAndServe()
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal(
					"error starting server",
					zap.String("host", viper.GetString("host")),
					zap.String("port", viper.GetString("port")),
					zap.Error(err),
				)
			}
		}()

		<-shutdown

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = srv.Shutdown(ctx)
		if err != nil {
			logger.Error(
				"error stopping server",
				zap.String("host", viper.GetString("host")),
				zap.String("port", viper.GetString("port")),
				zap.Error(err),
			)
			return err
		}
		return nil
	}

	if err := c.Execute(); err != nil {
		os.Exit(1)
	}

}
