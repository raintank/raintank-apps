package gitstats

import (
	"os"
	"testing"

	"github.com/intelsdi-x/snap/control/plugin"
	"github.com/intelsdi-x/snap/control/plugin/cpolicy"
	"github.com/intelsdi-x/snap/core"
	"github.com/intelsdi-x/snap/core/cdata"
	"github.com/intelsdi-x/snap/core/ctypes"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGitstatsPlugin(t *testing.T) {
	Convey("Meta should return metadata for the plugin", t, func() {
		meta := Meta()
		So(meta.Name, ShouldResemble, Name)
		So(meta.Version, ShouldResemble, Version)
		So(meta.Type, ShouldResemble, plugin.CollectorPluginType)
	})

	Convey("Create Gitstats Collector", t, func() {
		collector := &Gitstats{}
		Convey("So Gitstats collector should not be nil", func() {
			So(collector, ShouldNotBeNil)
		})
		Convey("So Gitstats collector should be of Gitstats type", func() {
			So(collector, ShouldHaveSameTypeAs, &Gitstats{})
		})
		Convey("collector.GetConfigPolicy() should return a config policy", func() {
			configPolicy, _ := collector.GetConfigPolicy()
			Convey("So config policy should not be nil", func() {
				So(configPolicy, ShouldNotBeNil)
			})
			Convey("So config policy should be a cpolicy.ConfigPolicy", func() {
				So(configPolicy, ShouldHaveSameTypeAs, &cpolicy.ConfigPolicy{})
			})
			Convey("So config policy namespace should be /raintank/Gitstats", func() {
				conf := configPolicy.Get([]string{"raintank", "apps", "gitstats"})
				So(conf, ShouldNotBeNil)
				So(conf.HasRules(), ShouldBeTrue)
				tables := conf.RulesAsTable()
				So(len(tables), ShouldEqual, 3)
				for _, rule := range tables {
					So(rule.Name, ShouldBeIn, "access_token", "user", "repo")
					switch rule.Name {
					case "access_token":
						So(rule.Required, ShouldBeTrue)
						So(rule.Type, ShouldEqual, "string")
					case "user":
						So(rule.Required, ShouldBeFalse)
						So(rule.Type, ShouldEqual, "string")
					case "repo":
						So(rule.Required, ShouldBeFalse)
						So(rule.Type, ShouldEqual, "string")
					}
				}
			})
		})
	})
}

func TestGitstatsCollectMetrics(t *testing.T) {
	cfg := setupCfg("woodsaj", "")

	Convey("Ping collector", t, func() {
		p := &Gitstats{}
		mt, err := p.GetMetricTypes(cfg)
		if err != nil {
			t.Fatal("failed to get metricTypes", err)
		}
		So(len(mt), ShouldBeGreaterThan, 0)
		for _, m := range mt {
			t.Log(m.Namespace().String())
		}
		Convey("collect metrics", func() {
			mts := []plugin.MetricType{
				plugin.MetricType{
					Namespace_: core.NewNamespace(
						"raintank", "apps", "gitstats", "user", "*", "followers"),
					Config_: cfg.ConfigDataNode,
				},
			}
			metrics, err := p.CollectMetrics(mts)
			So(err, ShouldBeNil)
			So(metrics, ShouldNotBeNil)
			So(len(metrics), ShouldEqual, 1)
			So(metrics[0].Namespace()[0].Value, ShouldEqual, "raintank")
			So(metrics[0].Namespace()[1].Value, ShouldEqual, "apps")
			So(metrics[0].Namespace()[2].Value, ShouldEqual, "gitstats")
			for _, m := range metrics {
				So(m.Namespace()[3].Value, ShouldEqual, "user")
				So(m.Namespace()[4].Value, ShouldEqual, "woodsaj")
				t.Log(m.Namespace().String(), m.Data())
			}
		})
	})
}

func setupCfg(user, repo string) plugin.ConfigType {
	node := cdata.NewNode()
	node.AddItem("access_token", ctypes.ConfigValueStr{Value: os.Getenv("GITSTATS_ACCESS_TOKEN")})
	node.AddItem("user", ctypes.ConfigValueStr{Value: user})
	node.AddItem("repo", ctypes.ConfigValueStr{Value: repo})
	return plugin.ConfigType{ConfigDataNode: node}
}
