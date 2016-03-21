package task_client

import (
	"github.com/raintank/raintank-apps/task-server/client"
)

var Client *client.Client

func Init(addr, apiKey string, insecure bool) (err error) {
	Client, err = client.New(addr, apiKey, insecure)
	return err
}
