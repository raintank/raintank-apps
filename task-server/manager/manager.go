package manager

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"github.com/raintank/raintank-apps/task-server/api"
	"github.com/raintank/raintank-apps/task-server/event"
	"github.com/raintank/raintank-apps/task-server/model"
	"github.com/raintank/raintank-apps/task-server/sqlstore"
	"github.com/raintank/worldping-api/pkg/log"
)

func Init() {
	agentOfflineChan := make(chan event.RawEvent, 100)
	event.Subscribe("agent.offline", agentOfflineChan)
	go HandleAgentOfflineEvents(agentOfflineChan)

	taskCreatedChan := make(chan event.RawEvent, 100)
	event.Subscribe("task.created", taskCreatedChan)
	go HandleTaskCreatedEvent(taskCreatedChan)

	go checkOrphanedAgents()
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

func HandleTaskCreatedEvent(c chan event.RawEvent) {
	for event := range c {
		task := new(model.TaskDTO)
		err := json.Unmarshal(event.Body, task)
		if err != nil {
			log.Error(3, "Unable to unmarshal agentOffline event. %s", err)
			continue
		}

		go api.ActiveSockets.EmitTask(task, "taskAdd")
	}
}

func checkOrphanedAgents() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		// check for agents marked as online but have no sessions
		nowOffline, err := sqlstore.OnlineAgentsWithNoSession()
		if err != nil {
			log.Error(3, "unable to get list of OnlineAgentsWithNoSession, %s", err)
			continue
		}
		if len(nowOffline) > 0 {
			for _, a := range nowOffline {
				a.Online = false
				a.OnlineChange = time.Now()
				log.Info("Agent %s has no sessions. Marking as offline.", a.Name)
				err = sqlstore.UpdateAgent(a)
				if err != nil {
					log.Error(3, "failed to update agent. %s", err)
					continue
				}
				event.Publish(&event.AgentOffline{Ts: time.Now(), Payload: a}, 0)
			}
		}

		// check for agent_sessions with heartbeats that are no longer being updated.
		err = sqlstore.DeleteAgentSessionsWithStaleHeartbeat(time.Minute)
		if err != nil {
			log.Error(3, "failed to prune stale agent_sessions. %s", err)
		}

	}
}
