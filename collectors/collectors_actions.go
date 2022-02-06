package collectors

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/google/go-github/v42/github"
	metrics "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	actionsOrganizations = kingpin.Flag("collector.actions.organizations", "Github organizations to scrape (Actions service)").Default("").Envar("COLLECTOR_ACTIONS_ORGANIZATIONS").String()
	actionsRepositories  = kingpin.Flag("collector.actions.repositories", "Github repositories to scrape (Actions service)").Default("").Envar("COLLECTOR_ACTIONS_REPOSITORIES").String()
)

type actionsCollector struct {
	runnersStatus      *metrics.Desc
	runnersIdleCount   *metrics.Desc
	runnersBusyCount   *metrics.Desc
	workflowStatus     *metrics.Desc
	workflowRunsStatus *metrics.Desc

	logger *zap.Logger
	client *github.Client

	organizations string
	repositories  string

	wg *sync.WaitGroup
}

func newActionsCollector(client *github.Client, logger *zap.Logger) (Collector, error) {
	runnersStatus := metrics.NewDesc(
		metrics.BuildFQName(
			defaultNamespace, "actions", "runners_status",
		),
		"Status of Github Actions runners",
		[]string{"name", "status", "busy", "os", "org", "repo"}, nil,
	)
	runnersIdleCount := metrics.NewDesc(
		metrics.BuildFQName(
			defaultNamespace, "actions", "runners_idle_count",
		),
		"Total idle Github Action runners",
		[]string{"org", "repo"}, nil,
	)
	runnersBusyCount := metrics.NewDesc(
		metrics.BuildFQName(
			defaultNamespace, "actions", "runners_busy_count",
		),
		"Total busy Github Action runners",
		[]string{"org", "repo"}, nil,
	)
	workflowStatus := metrics.NewDesc(
		metrics.BuildFQName(
			defaultNamespace, "actions", "workflows_status",
		),
		"Status of Github Action workflow",
		[]string{"org", "repo", "state", "name", "url"}, nil,
	)
	workflowRunsStatus := metrics.NewDesc(
		metrics.BuildFQName(
			defaultNamespace, "actions", "workflows_runs_status",
		),
		"Status of Github Action workflow runs",
		[]string{"org", "repo", "status"}, nil,
	)

	c := &actionsCollector{
		workflowRunsStatus: workflowRunsStatus,
		workflowStatus:     workflowStatus,
		runnersBusyCount:   runnersBusyCount,
		runnersIdleCount:   runnersIdleCount,
		runnersStatus:      runnersStatus,
		organizations:      *actionsOrganizations,
		repositories:       *actionsRepositories,
		logger:             logger,
		client:             client,
		wg:                 &sync.WaitGroup{},
	}

	c.logger.Info(
		"registered and started collector",
		zap.String("organizations", c.organizations),
		zap.String("repositories", c.repositories),
		zap.String("collector", "actions"),
	)

	return c, nil
}

func (c *actionsCollector) Update(ch chan<- metrics.Metric) error {
	ctx := context.Background()

	errCh := make(chan error, 1)
	for _, org := range strings.Split(c.organizations, ",") {
		repos := make([]*github.Repository, 0)
		opts := &github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{
				PerPage: 100,
			},
		}
		for {
			results, resp, err := c.client.Repositories.ListByOrg(ctx, org, opts)
			if err != nil {
				errCh <- err
				break
			}
			repos = append(repos, results...)
			if resp.NextPage == 0 {
				break
			}
			opts.Page = resp.NextPage
		}

		c.wg.Add(1)
		go func(org string) {
			c.scrapeOrganizationRunners(ctx, ch, errCh, org)
			c.wg.Done()
		}(org)

		for _, repo := range repos {
			c.wg.Add(1)
			go func(repo string) {
				c.scrapeWorkflows(ctx, ch, errCh, org, repo)
				c.scrapeRepositoryWorkflowRunsByStatus(ctx, ch, errCh, org, repo, "queued")
				c.scrapeRepositoryWorkflowRunsByStatus(ctx, ch, errCh, org, repo, "in_progress")
				c.scrapeRepositoryWorkflowRunsByStatus(ctx, ch, errCh, org, repo, "completed")
				c.wg.Done()
			}(*repo.Name)
		}
	}

	c.wg.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (c *actionsCollector) scrapeOrganizationRunners(ctx context.Context, ch chan<- metrics.Metric, errCh chan<- error, org string) {
	runners := make([]*github.Runner, 0)
	opts := &github.ListOptions{
		PerPage: 100,
	}
	for {
		results, resp, err := c.client.Actions.ListOrganizationRunners(ctx, org, opts)
		if err != nil {
			errCh <- err
			return
		}
		runners = append(runners, results.Runners...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	busy := 0
	idle := 0
	for _, runner := range runners {
		status := 1.0
		if runner.GetStatus() == "offline" {
			status = 0.0
		}
		ch <- metrics.MustNewConstMetric(
			c.runnersStatus, metrics.GaugeValue, status,
			*runner.Name, *runner.Status, strconv.FormatBool(*runner.Busy), *runner.OS, org, "",
		)
		if *runner.Busy {
			busy++
		} else {
			idle++
		}
	}

	ch <- metrics.MustNewConstMetric(
		c.runnersBusyCount, metrics.GaugeValue, float64(busy),
		org, "",
	)
	ch <- metrics.MustNewConstMetric(
		c.runnersIdleCount, metrics.GaugeValue, float64(idle),
		org, "",
	)
}

func (c *actionsCollector) scrapeRepositoryWorkflowRunsByStatus(ctx context.Context, ch chan<- metrics.Metric, errCh chan<- error, org string, repo string, status string) {
	results, _, err := c.client.Actions.ListRepositoryWorkflowRuns(
		ctx, org, repo,
		&github.ListWorkflowRunsOptions{
			Status: status,
		},
	)
	if err != nil {
		errCh <- err
		return
	}
	ch <- metrics.MustNewConstMetric(
		c.workflowRunsStatus, metrics.GaugeValue, float64(results.GetTotalCount()),
		org, repo, status,
	)
}

func (c *actionsCollector) scrapeWorkflows(ctx context.Context, ch chan<- metrics.Metric, errCh chan<- error, org string, repo string) {
	workflows := make([]*github.Workflow, 0)
	opts := &github.ListOptions{
		PerPage: 100,
	}
	for {
		results, resp, err := c.client.Actions.ListWorkflows(ctx, org, repo, opts)
		if err != nil {
			errCh <- err
			return
		}
		workflows = append(workflows, results.Workflows...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	if workflows == nil {
		fmt.Println("NIL")
	}

	state := 0.0
	for _, workflow := range workflows {
		if workflow.GetState() == "active" {
			state = 1.0
		}
		ch <- metrics.MustNewConstMetric(
			c.workflowStatus, metrics.GaugeValue, state,
			org, repo, workflow.GetState(), workflow.GetName(), workflow.GetURL(),
		)
	}
}

func init() {
	registerCollector("actions", newActionsCollector)
}
