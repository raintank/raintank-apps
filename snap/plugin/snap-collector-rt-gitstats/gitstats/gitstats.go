package gitstats

import (
	"fmt"
	"time"

	"github.com/google/go-github/github"
	. "github.com/intelsdi-x/snap-plugin-utilities/logger"
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
	accessToken, ok := conf["access_token"]
	if !ok || accessToken.(ctypes.ConfigValueStr).Value == "" {
		return nil, fmt.Errorf("access token missing from config, %v", conf)
	}

	metrics, err := gitStats(accessToken.(ctypes.ConfigValueStr).Value, mts)
	if err != nil {
		return nil, err
	}

	return metrics, nil
}

type repoName struct {
	Repo  string
	Owner string
}

func gitStats(accessToken string, mts []plugin.MetricType) ([]plugin.MetricType, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	client := github.NewClient(tc)
	collectionTime := time.Now()
	repos := make(map[string]map[string]map[string]int)
	users := make(map[string]map[string]int)

	userRepos := make(map[string]struct{})

	authUser := ""
	metrics := make([]plugin.MetricType, 0)

	for _, m := range mts {
		ns := m.Namespace().Strings()
		switch ns[3] {
		case "repo":
			user := ns[4]
			repo := ns[5]
			stat := ns[6]

			if user == "*" {
				//need to get user
				if authUser == "" {
					gitUser, _, err := client.Users.Get("")
					if err != nil {
						LogError("failed to get authenticated user.", err)
						return nil, err
					}
					stats, err := userStats(gitUser, client)
					if err != nil {
						LogError("failed to get stats from user object.", err)
						return nil, err
					}
					users[*gitUser.Login] = stats
					authUser = *gitUser.Login
				}
				user = authUser
			}
			if repo == "*" {
				// we only need to list a users repos once.
				if _, ok := userRepos[user]; !ok {
					repoList, _, err := client.Repositories.List(user, nil)
					if err != nil {
						LogError("failed to get repos owned by user.", err)
						return nil, err
					}
					userRepos[user] = struct{}{}
					if _, ok := repos[user]; !ok {
						repos[user] = make(map[string]map[string]int)
					}
					for _, r := range repoList {
						stats, err := repoStats(&r)
						if err != nil {
							LogError("failed to get stats from repo object.", err)
							return nil, err
						}
						repos[user][*r.Name] = stats
					}
				}
				for repo, stats := range repos[user] {
					mt := plugin.MetricType{
						Data_:      stats[stat],
						Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "repo", user, repo, stat),
						Timestamp_: collectionTime,
						Version_:   m.Version(),
					}
					metrics = append(metrics, mt)
				}

			} else {
				if _, ok := repos[user]; !ok {
					repos[user] = make(map[string]map[string]int)
				}
				if _, ok := repos[user][repo]; !ok {
					r, _, err := client.Repositories.Get(user, repo)
					if err != nil {
						LogError("failed to user repos.", err)
						return nil, err
					}
					stats, err := repoStats(r)
					if err != nil {
						LogError("failed to get stats from repo object.", err)
						return nil, err
					}
					repos[user][repo] = stats
				}
				mt := plugin.MetricType{
					Data_:      repos[user][repo][stat],
					Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "repo", user, repo, stat),
					Timestamp_: collectionTime,
					Version_:   m.Version(),
				}
				metrics = append(metrics, mt)
			}

		case "user":
			user := ns[4]
			stat := ns[5]
			if user == "*" {
				//need to get user
				if authUser == "" {
					gitUser, _, err := client.Users.Get(user)
					if err != nil {
						LogError("failed to get authenticated user.", err)
						return nil, err
					}
					authUser = *gitUser.Login
					stats, err := userStats(gitUser, client)
					if err != nil {
						LogError("failed to get stats from user object", err)
						return nil, err
					}
					users[*gitUser.Login] = stats
				}
			} else {
				if _, ok := users[user]; !ok {
					u, _, err := client.Users.Get(user)
					if err != nil {
						LogError("failed to lookup user.", err)
						return nil, err
					}
					stats, err := userStats(u, client)
					if err != nil {
						LogError("failed to get stats from user object.", err)
						return nil, err
					}
					users[user] = stats
				}
			}
			mt := plugin.MetricType{
				Data_:      users[user][stat],
				Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "user", user, stat),
				Timestamp_: collectionTime,
				Version_:   m.Version(),
			}
			metrics = append(metrics, mt)
		}
	}

	return metrics, nil
}

func userStats(user *github.User, client *github.Client) (map[string]int, error) {
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
		org, _, err := client.Organizations.Get(*user.Login)
		if err != nil {
			LogError("failed to lookup org data.", err)
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

func repoStats(resp *github.Repository) (map[string]int, error) {
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
	mts := make([]plugin.MetricType, 0)
	for _, metricName := range repoMetricNames {
		mts = append(mts, plugin.MetricType{
			Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "repo").
				AddDynamicElement("owner", "repository owner").
				AddDynamicElement("repo", "repository name").
				AddStaticElement(metricName),
			Config_: cfg.ConfigDataNode,
		})
	}
	for _, metricName := range userMetricNames {
		mts = append(mts, plugin.MetricType{
			Namespace_: core.NewNamespace("raintank", "apps", "gitstats", "user").
				AddDynamicElement("user", "user or orginisation name").
				AddStaticElement(metricName),
			Config_: cfg.ConfigDataNode,
		})
	}
	return mts, nil
}

//GetConfigPolicy returns a ConfigPolicyTree for testing
func (f *Gitstats) GetConfigPolicy() (*cpolicy.ConfigPolicy, error) {
	c := cpolicy.New()
	rule, _ := cpolicy.NewStringRule("access_token", true)
	p := cpolicy.NewPolicyNode()
	p.Add(rule)
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
