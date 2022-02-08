package collectors

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/go-github/v42/github"
	"github.com/pkg/errors"
	metrics "github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	defaultNamespace = "github"
)

// CollectorFactory is a common function for creating new Collectors
type CollectorFactory func(client *github.Client) (Collector, error)

var (
	Collectors         = make([]string, 0)
	collectorFactories = make(map[string]CollectorFactory)
)

// Collector collects Prometheus metrics
type Collector interface {
	// Update collects Prometheus metrics of collector
	Update(ctx context.Context, ch chan<- metrics.Metric) error
}

func registerCollector(name string, enabled bool, factory CollectorFactory) {
	_, ok := collectorFactories[name]
	if ok {
		log.Fatalf("collector '%v' is already registered", name)
		return
	}
	collectorFactories[name] = factory
	if enabled {
		Collectors = append(Collectors, name)
	}
}

// GithubCollector is the main collector which collects Prometheus metrics
// from other registered collectors
type GithubCollector struct {
	Collectors map[string]Collector

	scrapeDurationDesc *metrics.Desc
	scrapeSuccessDesc  *metrics.Desc

	client *github.Client
	logger *zap.Logger
	wg     *sync.WaitGroup
	mu     *sync.Mutex
}

// NewGithubCollector initializes a new *GithubCollector instance
func NewGithubCollector(client *github.Client, logger *zap.Logger) (*GithubCollector, error) {
	scrapeDurationDesc := metrics.NewDesc(
		metrics.BuildFQName(defaultNamespace, "scrape", "collector_duration_seconds"),
		"collector: Duration of collector scrape",
		[]string{"collector"},
		nil,
	)
	scrapeSuccessDesc := metrics.NewDesc(
		metrics.BuildFQName(defaultNamespace, "scrape", "collector_success"),
		"collector: Success collector scrapes count",
		[]string{"collector"},
		nil,
	)

	c := &GithubCollector{
		Collectors:         make(map[string]Collector),
		scrapeDurationDesc: scrapeDurationDesc,
		scrapeSuccessDesc:  scrapeSuccessDesc,
		logger:             logger,
		client:             client,
		wg:                 &sync.WaitGroup{},
		mu:                 &sync.Mutex{},
	}

	for _, name := range viper.GetStringSlice("collectors") {
		collectorFn, ok := collectorFactories[name]
		if !ok {
			logger.Info(
				"collector is not implemented or registed, ignoring",
				zap.String("collector", name),
			)
			continue
		}
		collector, err := collectorFn(client)
		if err != nil {
			return nil, errors.Wrap(err, "error creating collector")
		}
		c.Collectors[name] = collector
	}

	return c, nil
}

// Describe implements Prometheus Collector interface
func (c *GithubCollector) Describe(ch chan<- *metrics.Desc) {
	ch <- c.scrapeDurationDesc
	ch <- c.scrapeSuccessDesc
}

// Collect implements Prometheus Collector interface
func (c *GithubCollector) Collect(ch chan<- metrics.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for name := range c.Collectors {
		c.wg.Add(1)
		go func(name string) {
			c.collect(ctx, name, ch)
		}(name)
	}

	c.wg.Wait()
}

func (c *GithubCollector) collect(ctx context.Context, collector string, ch chan<- metrics.Metric) {
	now := time.Now()

	done := make(chan error, 1)
	go func() {
		err := c.Collectors[collector].Update(ctx, ch)
		done <- err
	}()

	success := 1.0
	select {
	case err := <-done:
		if err != nil {
			success = 0
			c.logger.Error(
				"collector scrape failure",
				zap.String("collector", collector),
				zap.String("took", time.Since(now).String()),
				zap.Error(err),
			)
		}
	case <-ctx.Done():
		success = 0
		c.logger.Debug(
			"collector scrape timeout",
			zap.String("collector", collector),
			zap.String("took", time.Since(now).String()),
		)
	}

	if success == 1.0 {
		c.logger.Debug(
			"collector scrape successfull",
			zap.String("collector", collector),
			zap.String("took", time.Since(now).String()),
		)
	}

	ch <- metrics.MustNewConstMetric(c.scrapeSuccessDesc, metrics.GaugeValue, success, collector)
	ch <- metrics.MustNewConstMetric(c.scrapeDurationDesc, metrics.GaugeValue, float64(time.Since(now).Seconds()), collector)

	c.wg.Done()
}
