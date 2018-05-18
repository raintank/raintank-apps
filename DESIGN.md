# Overall Architecture

[Loopy.io Diagram](https://bit.ly/2EwOKTS)

![raintank-app](img/raintank-app-animation.gif)

## Task Server

The task server provides a REST API to manage plugins, requests, agents, and handles scheduling tasks to be executed by agents.

### configuration settings

|Key|Value|Description
|---|-----|-----------|
addr | :8082 | port to bind task-server
app-api-key| API_KEY | secret used to communicate between task-agent and task-server
db-type| mysql \| sqlite3 | Database backend to use
db-connect-str | USER:PASSWORD@tcp(localhost:3306)/task_server?charset=utf8| sample MySQL connection string
exchange| *leave this empty* | rabbitmq connection string, not used
log-level| 0..6 | log output level from TRACE (verbose) to INFO

|Section|Key|Value|Description
|-------|---|-----|-----------|
| stats | addr | address:port | graphite address for internal Metrics
|       | enabled | true\|false| send internal metrics
Example configuration file:

```
addr = :8082
app-api-key = API_KEY
db-type = mysql
db-connect-str = USER:PASSWORD@tcp(localhost:3306)/task_server?charset=utf8
exchange =
log-level = 0
[stats]
addr = 192.168.1.99:2003
enabled = true
```

### Requests

Grafana Applications like NS1 connect to the task server to configure metric collection

### Tasks

Requests are turned into tasks that will be run by an agent.  Tasks have typical CRUD operations, and can also be enabled/disabled.

### Agents

Agents connect to the task server and receive tasks to process, sending metric results to a tsdb-gw.


### Dependencies

Databases supported are sqlite3 and MySQL.

## Task Agent

Task agent executes a task on a regular interval specified by the task.
The task agent connects to a task server to receive tasks that need to be executed.
The task agent will send the metric results to the specified TSDB-GW

### Configuration Settings

```
app-api-key = API_KEY
log-level = 0
name = agent1
server-url = ws://task-server:8082/api/v1/
tsdbgw-url = https://not-tsdb-gw.raintank.io/
tsdbgw-admin-key = EASY
[stats]
addr = 192.168.1.99:2003
enabled = true
```
|Key|Value|Description
|---|-----|-----------|
app-api-key| API_KEY | secret used to communicate between task-agent and task-server
log-level| 0..6 | log output level from TRACE (verbose) to INFO
name| agentname<br>or<br>""| name of agent, leave empty to use hostname
server-url| wss://task-server:8082/api/v1/<br>or<br>ws://task-server:8082/api/v1/|websocket address of the task server
tsdbgw-url | https://tsdb-gw.raintank.io/ | url to your TSDB-GW
tsdbgw-admin-key| EASY | API Admin Key for TSDB-GW


|Section|Key|Value|Description
|-------|---|-----|-----------|
| stats | addr | address:port | graphite address for internal Metrics
|       | enabled | true\|false| send internal metrics

### Plugins

The task agent has builtin plugin support to process tasks.

#### NS1

The NS1 plugin leverages the NS1 API to get QPS stats for domains. These metrics are sent to the Grafana.com TSDB Gateway and are stored on a per-user basis using a Grafana API Key.

#### Voxter

Currently under development due to API changes.

# Deployment

The application can be run on an Ubuntu/Debian distribution, under docker, and also inside a Kubernetes cluster.

## Kubernetes

Example Kubernetes deployment are provided.

The task-server and task-agents are run as statefulsets to retain the naming convention for the pod plus allow for buffering of metrics when they cannot be sent.

## Docker

A "docker-compose" example is provided that will stand up a complete instance of the application.

# Internal Metrics

## Task Agent Metrics
The following metrics are sent to metrictank, using the prefix:
```
raintank.app.stats.taskagent.$instance
```

|name|type|description|
|----|----|-----------|
runner.initialized|gauge|1 when runner is enabled
runner.tasks.active|gauge|current count of active tasks in runner
runner.tasks.added|counter|number of tasks added to runner
runner.tasks.removed|counter|number of tasks removed from runner
tasks.added|counter|tasks added to queue
tasks.removed|counter|tasks removed from queue
tasks.updated|counter|tasks updated in queue


## Plugin Metrics

The following metrics are sent to metrictank, using the prefix:
```
raintank.app.stats.taskagent.$instance
```

### NS1
|name|type|description|
|----|----|-----------|
collector.ns1.collect.attempts|counter|
collector.ns1.collect.success|counter|
collector.ns1.collect.failure|counter|
collector.ns1.client.queries|counter|
collector.ns1.client.authfailures|counter|
collector.ns1.collect.duration_ns|gauge|
collector.ns1.collect.success.duration_ns|gauge|
collector.ns1.collect.failure.duration_ns|gauge|

## Task Server metrics

The following metrics are sent to metrictank, using the prefix:
```
raintank.app.stats.taskserver.$instance
```
|name|type|description|
|----|----|-----------|
running|gauge|Set to 1 on startup, 0 on shutdown
tasks.active|gauge|Total tasks that are scheduled
tasks.disabled|gauge|Total tasks are disabled
api.tasks.created|counter|Tasks create via API
api.tasks.deleted|counter|Tasks deleted via API
api.tasks.updated|counter|Tasks updated via API
agent.connections.active|gauge|Total agents connected
agent.connections.failed|counter|Count of Agent connect failures
agent.connections.accepted|counter|Count of Accepted connections
agent.autocreate.success|counter|Agent auto create successes
agent.autocreate.failed|counter|Agent auto create failures
