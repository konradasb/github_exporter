package collectors

import (
	"log"
	"sync"
	"time"

	"github.com/google/go-github/v42/github"
	metrics "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

const (
	defaultNamespace = "github"
)

// CollectorFactory is a common function for creating new Collectors
type CollectorFactory func(client *github.Client, logger *zap.Logger) (Collector, error)

var (
	collectorFactories = make(map[string]CollectorFactory)
)

// Collector collects Prometheus metrics
type Collector interface {
	// Update collects Prometheus metrics of collector
	Update(ch chan<- metrics.Metric) error
}

func registerCollector(name string, factory CollectorFactory) {
	_, ok := collectorFactories[name]
	if ok {
		log.Fatalf("collector '%v' is already registered", name)
		return
	}
	collectorFactories[name] = factory
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

type GithubCollectorOpts struct {
	Client        *github.Client
	Logger        *zap.Logger
	Organizations []string
	Repositories  []string
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

	for name, collector := range collectorFactories {
		result, err := collector(client, logger)
		if err != nil {
			return nil, err
		}
		c.Collectors[name] = result
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
	for name := range c.Collectors {
		c.wg.Add(1)
		go func(name string) {
			c.collect(name, ch)
			c.wg.Done()
		}(name)
	}
	c.wg.Wait()
}

func (c *GithubCollector) collect(collector string, ch chan<- metrics.Metric) {
	now := time.Now()
	err := c.Collectors[collector].Update(ch)
	ch <- metrics.MustNewConstMetric(c.scrapeDurationDesc, metrics.GaugeValue, float64(time.Since(now).Seconds()), collector)
	if err != nil {
		c.logger.Error(
			"collector scrape error",
			zap.String("collector", collector),
			zap.Error(err),
		)
		ch <- metrics.MustNewConstMetric(c.scrapeSuccessDesc, metrics.GaugeValue, 0, collector)
		return
	}
	c.logger.Debug(
		"collector scrape successfull",
		zap.String("collector", collector),
	)
	ch <- metrics.MustNewConstMetric(c.scrapeSuccessDesc, metrics.GaugeValue, 1, collector)
}
