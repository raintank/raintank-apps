package task_client

import (
	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/raintank-apps/task-server/client"
)

var Client *client.Client

func Init(addr, apiKey string, insecure bool) (err error) {
	log.Info("setting taskServer address to: %s", addr)
	Client, err = client.New(addr, apiKey, insecure)
	return err
}
