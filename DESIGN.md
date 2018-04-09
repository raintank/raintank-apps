# Overall Architecture

[Diagram](https://bit.ly/2H1Oi5m)

![raintank-app](img/raintank-app-animation.gif)

## Task Server

The task server provides a REST API to manage plugins, requests, agents, and handles scheduling tasks to be executed by agents.

### configuration settings

### Requests

Grafana Applications like NS1 connect to the task server to configure metric collection

### Tasks

Requests are turned into tasks that will be run by an agent.  Tasks have typical CRUD operations, and can also be enabled/disabled.

### Agents

Agents connect to the task server and receive tasks to process.


### Dependencies

Databases supported are sqlite3 and MySQL.

## Task Agent

Task agent executes a task on a regular interval specified by the task.
The task agent connects to a task server to receive tasks that need to be executed.

### Configuration Settings

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
runner.tasks.active.count|gauge|current count of active tasks in runner
runner.tasks.added.count|counter|number of tasks added to runner
runner.tasks.removed.count|counter|number of tasks removed from runner
tasks.added.count|counter|tasks added to queue
tasks.removed.count|counter|tasks removed from queue
tasks.updated.count|counter|tasks updated in queue


## Plugin Metrics

The following metrics are sent to metrictank, using the prefix:
```
raintank.app.stats.taskagent.$instance
```

### NS1
|name|type|description|
|----|----|-----------|
collector.ns1.collect.attempts.count|counter|
collector.ns1.collect.success.count|counter|
collector.ns1.collect.failure.count|counter|
collector.ns1.client.queries.count|counter|
collector.ns1.client.authfailures.count|counter|
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
