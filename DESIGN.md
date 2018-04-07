# Overall Architecture

[Diagram](https://bit.ly/2H1Oi5m)

<iframe width="500" height="440" frameborder="0" src="http://ncase.me/loopy/v1.1/?embed=1&data=[[[3,487,309,1,%22Task%2520Server%22,0],[4,481,495,1,%22Task%2520Agent%22,0],[6,729,169,0.83,%22database%22,0],[7,455,753,0.66,%22Voxter%2520Plugin%22,0],[8,610,758,0.66,%22NS1%2520Plugin%22,0],[9,996,629,0.83,%22metrictank%22,0],[10,1008,378,0.83,%22TSDB-GW%2520API%22,0]],[[3,6,13,1,0],[3,4,-9,1,0],[4,7,12,1,0],[4,8,24,1,0],[4,9,17,1,0],[3,9,-24,1,0],[4,10,54,1,0],[3,10,55,1,0],[7,4,72,-1,0],[8,4,16,-1,0],[6,3,-40,-1,0]],[[732,94,%22Task%2520Storage%22],[1008,306,%22Plugin%2520Metrics%22],[999,556,%22App%2520Metrics%22],[801,458,%22...%22]],10%5D"></iframe>

## Task Server

The task server provides a REST API to manage plugins, requests, agents, and handles scheduling tasks to be executed by agents.

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

## Docker

A "docker-compose" example is provided that will stand up a complete instance of the application.
