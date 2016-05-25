package gitstats

import (
	"fmt"
	"time"

	"github.com/google/go-github/github"
	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/ctypes"
	"golang.org/x/oauth2"
)

const (
	// Name of plugin
	Name = "rt-gitstats"
	// Version of plugin
	Version = 1
	// Type of plugin
	Type = plugin.CollectorPluginType
)

// make sure that we actually satisify requierd interface
var _ plugin.CollectorPlugin = (*Gitstats)(nil)

var (
	repoMetricNames = []string{
		"forks",
		"issues",
		"network",
		"stars",
		"subscribers",
		"watches",
		"size",
	}
	userMetricNames = []string{
		"public_repos",
		"public_gists",
		"followers",
		"following",
		"private_repos",
		"private_gists",
		"plan_private_repos",
		"plan_seats",
		"plan_filled_seats",
	}
)

type Gitstats struct {
}

// CollectMetrics collects metrics for testing
func (f *Gitstats) CollectMetrics(mts []plugin.MetricType) ([]plugin.MetricType, error) {
	var err error

	conf := mts[0].Config().Table()
	fmt.Printf("%v", conf)
	accessToken, ok := conf["access_token"]
	if !ok || accessToken.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("access token missing from config, %v", conf)
	}
	owner, ok := conf["owner"]
	if !ok || owner.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("owner missing from config")
	}
	repo, ok := conf["repo"]
	if !ok || repo.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("repo missing from config")
	}

	metrics, err := gitStats(accessToken.(ctypes.ConfigValueStr).Value, owner.(ctypes.ConfigValueStr).Value, repo.(ctypes.ConfigValueStr).Value, mts)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

func gitStats(accessToken, owner, repo string, mts []plugin.MetricType) ([]plugin.MetricType, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)

	repoStats, err := getRepo(client, owner, repo)
	if err != nil {
		return nil, err
	}

	userStats, err := getUser(client, owner)
	if err != nil {
		return nil, err
	}

	metrics := make([]plugin.MetricType, 0)
	for _, m := range mts {
		stat := m.Namespace()[5].Value
		if value, ok := repoStats[stat]; ok {
			mt := plugin.MetricType{
				Data_:      value,
				Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "repo", owner, repo, stat),
				Timestamp_: time.Now(),
				Version_:   m.Version(),
			}
			metrics = append(metrics, mt)
		}
		if value, ok := userStats[stat]; ok {
			mt := plugin.MetricType{
				Data_:      value,
				Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "owner", owner, stat),
				Timestamp_: time.Now(),
				Version_:   m.Version(),
			}
			metrics = append(metrics, mt)
		}
	}

	return metrics, nil
}

func getUser(client *github.Client, owner string) (map[string]int, error) {
	//get contributor stats, then traverse and count. https://api.github.com/repos/openstack/nova/stats/contributors
	user, _, err := client.Users.Get(owner)
	if err != nil {
		return nil, err
	}
	stats := make(map[string]int)
	if user.PublicRepos != nil {
		stats["public_repos"] = *user.PublicRepos
	}
	if user.PublicGists != nil {
		stats["public_gists"] = *user.PublicGists
	}
	if user.Followers != nil {
		stats["followers"] = *user.Followers
	}
	if user.Following != nil {
		stats["following"] = *user.Following
	}

	if *user.Type == "Organization" {
		org, _, err := client.Organizations.Get(owner)
		if err != nil {
			return nil, err
		}
		if org.PrivateGists != nil {
			stats["private_gists"] = *org.PrivateGists
		}
		if org.TotalPrivateRepos != nil {
			stats["private_repos"] = *org.TotalPrivateRepos
		}
		if org.DiskUsage != nil {
			stats["disk_usage"] = *org.DiskUsage
		}
	}

	return stats, nil
}

func getRepo(client *github.Client, owner, repo string) (map[string]int, error) {
	resp, _, err := client.Repositories.Get(owner, repo)
	if err != nil {
		return nil, err
	}
	stats := make(map[string]int)

	if resp.ForksCount != nil {
		stats["forks"] = *resp.ForksCount
	}
	if resp.OpenIssuesCount != nil {
		stats["issues"] = *resp.OpenIssuesCount
	}
	if resp.NetworkCount != nil {
		stats["network"] = *resp.NetworkCount
	}
	if resp.StargazersCount != nil {
		stats["stars"] = *resp.StargazersCount
	}
	if resp.SubscribersCount != nil {
		stats["subcribers"] = *resp.SubscribersCount
	}
	if resp.WatchersCount != nil {
		stats["watchers"] = *resp.WatchersCount
	}
	if resp.Size != nil {
		stats["size"] = *resp.Size
	}
	return stats, nil
}

//GetMetricTypes returns metric types for testing
func (f *Gitstats) GetMetricTypes(cfg plugin.ConfigType) ([]plugin.MetricType, error) {
	mts := []plugin.MetricType{}
	for _, metricName := range repoMetricNames {
		mts = append(mts, plugin.MetricType{
			Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "repo", "*", "*", metricName),
			Config_:    cfg.ConfigDataNode,
		})
	}
	for _, metricName := range userMetricNames {
		mts = append(mts, plugin.MetricType{
			Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "repo", "*", "*", metricName),
			Config_:    cfg.ConfigDataNode,
		})
	}
	return mts, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (f *Gitstats) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("access_token", true)
	rule2, _ := cpolicy.NewStringRule("owner", true)
	rule3, _ := cpolicy.NewStringRule("repo", true)
	p := cpolicy.NewPolicyNode()
	p.Add(rule)
	p.Add(rule2)
	p.Add(rule3)
	c.Add([]string{"raintank", "apps", "gitstats"}, p)
	return c, nil
}

//Meta returns meta data for testing
func Meta() *plugin.PluginMeta {
	return plugin.NewPluginMeta(
		Name,
		Version,
		Type,
		[]string{plugin.SnapGOBContentType},
		[]string{plugin.SnapGOBContentType},
		plugin.Unsecure(true),
		plugin.ConcurrencyCount(1000),
	)
}
