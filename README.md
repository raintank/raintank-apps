# [Raintank Apps](https://raintank.io) [![Circle CI](https://circleci.com/gh/raintank/raintank-apps.svg?style=shield)](https://circleci.com/gh/raintank/raintank-apps)
================
[Website](https://raintank.io) |
[Twitter](https://twitter.com/raintankSaaS) |
[Slack](https://raintank.slack.com) |
[Email](mailto:hello@raintank.io)


Raintank Apps is the backend service for a number of Grafana Apps provided through [grafana.com](https://grafana.com/plugins)

## Release Notes

Version|Date         |Notes
-------|-------------|--------
0.0.1  | 2016-09-01 |initial release
1.0.0  | 2018-04-07 |Removal of SNAP dependencies and fix known bugs<br/>&bull; unit tests added<br/>&bull; now runs within docker<br/>&bull; docker-compose files provided for development

## TODO

### task-server

- [x] implement internal metrics and publisher
- [x] add internal metrics for no-agents connected (state critical) metric is "agent.connections.active"
- [x] add internal metrics for task-agents created automatically
- [ ] add database encryption for all sensitive data

### task-agent
  - [x] update task needs to be completed
  - [x] adding a task should use the specified interval (was hardcoded to 300 seconds)
  - [x] removeTask implementation
  - [x] add code to self-register agent to allow for rolling update/scaling
  - [ ] needs to report metrics for failing jobs
  - [ ] add task needs unit test
  - [ ] remove task needs unit test
  - [ ] update task needs unit test
  - [ ] send internal metrics even when there are no tasks active
  - [ ] align current NS1 Grafana plugin with metrics being sent

### plugins
  - [ ] voxter plugin needs to be converted (API not functioning, stubbed out plugin only)
