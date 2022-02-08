package collectors

import (
	"context"

	"github.com/google/go-github/v42/github"
	metrics "github.com/prometheus/client_golang/prometheus"
)

type ratelimitCollector struct {
	limitRemaining *metrics.Desc
	limitTotal     *metrics.Desc

	client *github.Client
}

func newRatelimitCollector(client *github.Client) (Collector, error) {
	c := &ratelimitCollector{
		limitRemaining: metrics.NewDesc(
			metrics.BuildFQName(defaultNamespace, "ratelimit", "limit_remaining"),
			"Total amount of requests remaining", []string{"resource"}, nil,
		),
		limitTotal: metrics.NewDesc(
			metrics.BuildFQName(defaultNamespace, "ratelimit", "limit_total"),
			"Total amount of requests", []string{"resource"}, nil,
		),
		client: client,
	}

	return c, nil
}

func (c *ratelimitCollector) Update(ctx context.Context, ch chan<- metrics.Metric) error {
	results, _, err := c.client.RateLimits(ctx)
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
	registerCollector("ratelimit", true, newRatelimitCollector)
}
