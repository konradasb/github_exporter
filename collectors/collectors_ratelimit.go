package collectors

import (
	"context"

	"github.com/google/go-github/v42/github"
	metrics "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type ratelimitCollector struct {
	limitRemaining *metrics.Desc
	limitTotal     *metrics.Desc

	logger *zap.Logger
	client *github.Client
}

func newRatelimitCollector(client *github.Client, logger *zap.Logger) (Collector, error) {
	c := &ratelimitCollector{
		limitRemaining: metrics.NewDesc(
			metrics.BuildFQName(defaultNamespace, "ratelimit", "limit_remaining"),
			"Total amount limit of requests", []string{"resource"}, nil,
		),
		limitTotal: metrics.NewDesc(
			metrics.BuildFQName(defaultNamespace, "ratelimit", "limit_total"),
			"Total amount of requests remaining", []string{"resource"}, nil,
		),
		logger: logger,
		client: client,
	}

	return c, nil
}

func (c *ratelimitCollector) Update(ch chan<- metrics.Metric) error {
	results, _, err := c.client.RateLimits(context.Background())
	if err != nil {
		return err
	}

	ch <- metrics.MustNewConstMetric(c.limitRemaining, metrics.GaugeValue, float64(results.Search.Remaining), "search")
	ch <- metrics.MustNewConstMetric(c.limitTotal, metrics.GaugeValue, float64(results.Search.Limit), "search")
	ch <- metrics.MustNewConstMetric(c.limitRemaining, metrics.GaugeValue, float64(results.Core.Remaining), "core")
	ch <- metrics.MustNewConstMetric(c.limitTotal, metrics.GaugeValue, float64(results.Core.Limit), "core")

	return nil
}

func init() {
	registerCollector("ratelimit", newRatelimitCollector)
}
