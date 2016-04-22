package manager

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/raintank-apps/task-server/event"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
)

func Init() {
	c := make(chan event.RawEvent, 100)
	event.Subscribe("agent.offline", c)
	go HandleAgentOfflineEvents(c)
}

func HandleAgentOfflineEvents(c chan event.RawEvent) {
	for event := range c {
		agent := new(model.AgentDTO)
		err := json.Unmarshal(event.Body, agent)
		if err != nil {
			log.Error(3, "Unable to unmarshal agentOffline event. %s", err)
			continue
		}
		log.Debug("Processing agentOffline event for %s", agent.Name)
		go handleAgentOffline(event.Source, agent)
	}
}

func handleAgentOffline(source string, a *model.AgentDTO) {
	// sleep 1 second before checking if agent is still offline.
	hostname, _ := os.Hostname()
	delay := time.Second
	if source != hostname {
		// add a random delay between 0 and 2.147 seconds (maxInt32 nanoseconds)
		delay = delay + time.Duration(rand.Int31())
	}
	time.Sleep(delay)
	//check if agent is still offline.
	currentState, err := sqlstore.GetAgentById(a.Id, 0)
	if err != nil {
		log.Error(3, "Failed to get current agent state from DB. %s", err)
		return
	}
	if currentState.Online {
		// agent is online again. Nothing further to do.
		return
	}

	// need to move any routeByAny tasks that are running on this agent to another one.
	err = sqlstore.RelocateRouteAnyTasks(a)
	if err != nil {
		log.Error(3, "Failed to relocated agents Tasks. %s", err)
	}

}
