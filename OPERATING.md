# Overview

task-server is "fronted" by nginx

API calls are made and tasks created/edited/removed within a SQL Database

task-agent
  Connects with a websocket to a task-server and is handed tasks to execute

## task-server

### configuration

```
log-level = 1
addr = :8082
db-type = mysql
db-connect-str = DB_USERNAME:DB_PASSWORD@tcp(NGINX_REVERSE_PROXY:3306)/task_server?charset=utf8
stats-enabled = true
statsd-addr = localhost:8125
statsd-type = standard
admin-key = YOUR_TASK_SERVER_API_KEY
exchange =
#rabbitmq-url = amqp://RMQ_USER:RMQ_PASSWORD@NGINX_REVERSE_PROXY:5672/

```
## task-agent

### configuration

```
log-level = 1
server-url = wss://task-server.raintank.io/api/v1
tsdb-url = https://tsdb-gw.raintank.io/
snap-url = http://localhost:8181/
stats-enabled = false
statsd-addr = localhost:8125
statsd-type = standard
api-key = YOUR_TASK_SERVER_API_KEY
name = task-agent-1
```

# Questions


#### Registering an Agent
For a task agent to work, it has to be pre-registered using the task-server API:

http://localhost:4000/api/v1/agents

CURL
```
curl -X POST \
  http://localhost:4000/api/v1/agents \
  -H 'Authorization: Bearer EASY' \
  -H 'Cache-Control: no-cache' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -H 'Postman-Token: 4ba0a028-5b71-f875-718b-63d71eb866ae' \
  -H 'content-type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW' \
  -F name=agent2
```

HTTP/RAW
```
POST /api/v1/agents HTTP/1.1
Host: localhost:4000
Content-Type: multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW
Authorization: Bearer EASY
Cache-Control: no-cache
Postman-Token: aa131313-1086-75a1-7749-958479bd554f

------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="name"

agent2
------WebKitFormBoundary7MA4YWxkTrZu0gW--
```

GO Client:
```
package main

import (
	"fmt"
	"strings"
	"net/http"
	"io/ioutil"
)

func main() {

	url := "http://localhost:4000/api/v1/agents"

	payload := strings.NewReader("------WebKitFormBoundary7MA4YWxkTrZu0gW\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nagent2\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW--")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("content-type", "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer EASY")
	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Postman-Token", "a32f687c-5915-44f4-8bc6-82e915ae0cad")

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	fmt.Println(res)
	fmt.Println(string(body))

}
```

One the POST has been submitted, the following output is received
```
{
    "meta": {
        "code": 200,
        "message": "success",
        "type": "agent"
    },
    "body": {
        "id": 1,
        "name": "agent1",
        "enabled": false,
        "enabledChange": "0001-01-01T00:00:00Z",
        "public": false,
        "tags": null,
        "online": false,
        "onlineChange": "0001-01-01T00:00:00Z",
        "created": "2018-03-17T04:58:56.087896316Z",
        "updated": "2018-03-17T04:58:56.087896376Z"
    }
}
```

#### Running with docker-compose

The provided docker-compose.yml file will stand up both a task-server and a task-agent.  Once the agent is created with the POST above, the task-agent will be able to communicate with the task-server.

```
$ docker-compose up
Starting raintankapps_task-server_1 ... done
Starting raintankapps_task-agent_1  ... done
Attaching to raintankapps_task-server_1, raintankapps_task-agent_1
task-server_1  | 2018/03/17 05:09:14 [I] Database: sqlite3
task-server_1  | 2018/03/17 05:09:14 [I] Migrator: Starting DB migration
task-server_1  | 2018/03/17 05:09:14 [I] using internal event channels
task-server_1  | 2018/03/17 05:09:14 [I] starting up
task-agent_1   | 2018/03/17 05:09:15 [I] connecting to ws://task-server:8082/api/v1/socket/agent1/1
task-server_1  | [Macaron] Started GET /api/v1/socket/agent1/1 for 172.18.0.3
task-server_1  | 2018/03/17 05:09:15 [D] socket: agent name agent1
task-server_1  | 2018/03/17 05:09:15 [D] socket: agent ver %!s(int64=1)
task-server_1  | 2018/03/17 05:09:15 [D] socket: agent orgid %!s(int64=1)
task-server_1  | 2018/03/17 05:09:15 [D] socket: agent agent1 connected.
task-server_1  | 2018/03/17 05:09:15 [D] Agent 1 is connected to this server.
task-agent_1   | 2018/03/17 05:09:15 [I] running SnapClient task supervisor.
task-agent_1   | 2018/03/17 05:09:15 [I] running SnapClient supervisor.
task-server_1  | 2018/03/17 05:09:15 [D] setting handler for disconnect event.
task-server_1  | 2018/03/17 05:09:15 [I] starting session 567edc2f-29a1-11e8-9a05-0242ac120002
task-server_1  | 2018/03/17 05:09:15 [D] sending TaskUpdate to 567edc2f-29a1-11e8-9a05-0242ac120002
task-server_1  | 2018/03/17 05:09:15 [D] socket 567edc2f-29a1-11e8-9a05-0242ac120002 sending message
task-agent_1   | 2018/03/17 05:09:15 [D] TaskList. null
task-agent_1   | 2018/03/17 05:09:16 [D] Snap server is unreachable. URL target is not available. Get http://localhost:8181/v1/plugins////config: dial tcp 127.0.0.1:8181: connect: connection refused
task-agent_1   | 2018/03/17 05:09:17 [D] Snap server is unreachable. URL target is not available. Get http://localhost:8181/v1/plugins////config: dial tcp 127.0.0.1:8181: connect: connection refused
task-server_1  | 2018/03/17 05:09:17 [D] socket 567edc2f-29a1-11e8-9a05-0242ac120002 sending message
task-agent_1   | 2018/03/17 05:09:17 [D] recieved heartbeat event. 2018-03-17 05:09:17.25672063 +0000 UTC m=+2.532669118
task-agent_1   | 2018/03/17 05:09:18 [D] Snap server is unreachable. URL target is not available. Get http://localhost:8181/v1/plugins////config: dial tcp 127.0.0.1:8181: connect: connection refused
task-agent_1   | 2018/03/17 05:09:19 [D] Snap server is unreachable. URL target is not available. Get http://localhost:8181/v1/plugins////config: dial tcp 127.0.0.1:8181: connect: connection refused
task-server_1  | 2018/03/17 05:09:19 [D] socket 567edc2f-29a1-11e8-9a05-0242ac120002 sending message
task-agent_1   | 2018/03/17 05:09:19 [D] recieved heartbeat event. 2018-03-17 05:09:19.256805662 +0000 UTC m=+4.532754141
task-agent_1   | 2018/03/17 05:09:20 [D] Snap server is unreachable. URL target is not available. Get http://localhost:8181/v1/plugins////config: dial tcp 127.0.0.1:8181: connect: connection refused
task-agent_1   | 2018/03/17 05:09:21 [D] Snap server is unreachable. URL target is not available. Get http://localhost:8181/v1/plugins////config: dial tcp 127.0.0.1:8181: connect: connection refused
task-server_1  | 2018/03/17 05:09:21 [D] socket 567edc2f-29a1-11e8-9a05-0242ac120002 sending message
task-agent_1   | 2018/03/17 05:09:21 [D] recieved heartbeat event. 2018-03-17 05:09:21.256605886 +0000 UTC m=+6.532554401
```

#### Creating a task
To create a task, you can do an HTTP POST to task-server:/api/v1/tasks

##### Headers
```
Authentication: Bearer EASY
Content-Type: application/json
```
##### JSON BODY
```
{
  "name": "example-task-1",
  "metrics": {
    "/example-task/*":0
  },
  "config": {
    "/example-task": {
      "param1": "value1",
      "param2": "value2"
    }
  },
  "interval": 300,
  "route": {
    "type": "any"
  },
  "enabled": true
}
```

##### Response

```
{
    "meta": {
        "code": 200,
        "message": "success",
        "type": "task"
    },
    "body": {
        "id": 1,
        "name": "example-task-1",
        "taskType": "",
        "orgId": 1,
        "config": {
            "/example-task": {
                "param1": "value1",
                "param2": "value2"
            }
        },
        "interval": 300,
        "route": {
            "type": "any",
            "config": {}
        },
        "enabled": true,
        "created": "2018-03-22T03:03:11.161804242Z",
        "updated": "2018-03-22T03:03:11.161804504Z"
    }
}
```

##### Server logs

```
task-server_1    | [Macaron] Started POST /api/v1/tasks for 172.18.0.1
task-server_1    | [Macaron] Completed /api/v1/tasks 200 OK in 43.661691ms
task-server_1    | 2018/03/22 03:03:11 [D] processing event of type task.created
task-server_1    | 2018/03/22 03:03:11 [D] sending taskAdd task event to connected agents.
task-server_1    | 2018/03/22 03:03:11 [D] Task has 1 agents. [1]
task-server_1    | 2018/03/22 03:03:11 [D] sending taskAdd event to agent 1
task-server_1    | 2018/03/22 03:03:11 [D] socket 2f8a6967-2d7a-11e8-82be-0242ac120002 sending message
task-agent-ng_1  | 2018/03/22 03:03:11 [D] Adding Task. {"id":1,"name":"example-task-1","taskType":"","orgId":1,"config":{"/example-task":{"param1":"value1","param2":"value2"}},"interval":300,"route":{"type":"any","config":{}},"enabled":true,"created":"2018-03-22T03:03:11.161804242Z","updated":"2018-03-22T03:03:11.161804504Z"}
```

```
task-server_1    | [Macaron] Started POST /api/v1/tasks for 172.18.0.1
task-server_1    | [Macaron] Completed /api/v1/tasks 200 OK in 28.858261ms
task-agent-ng_1  | 2018/03/22 03:40:23 [D] Adding Task. {"id":1,"name":"example-task-1","taskType":"","orgId":1,"config":{"/example-task":{"param1":"value1","param2":"value2"}},"interval":300,"route":{"type":"any","config":{}},"enabled":true,"created":"2018-03-22T03:40:23.620900394Z","updated":"2018-03-22T03:40:23.620900591Z"}
task-server_1    | 2018/03/22 03:40:23 [D] processing event of type task.created
task-server_1    | 2018/03/22 03:40:23 [D] sending taskAdd task event to connected agents.
task-server_1    | 2018/03/22 03:40:23 [D] Task has 1 agents. [1]
task-server_1    | 2018/03/22 03:40:23 [D] sending taskAdd event to agent 1
```
