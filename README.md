# Description
Simple pdagent implementation with go.
https://github.com/PagerDuty/pdagent

# Installation
Building packages you need.

## pdagent daemon
```
# go build -o pdagent cmd/pdagent/*
```

## pd-queue
```
# go build -o pd-queue cmd/pd-queue/*
```

## pd-zabbix
```
# go build -o pd-zabbix integrations/pd-zabbix/*
```

# Features
Much the same as the pdagent(Python).
https://www.pagerduty.com/docs/guides/agent-install-guide/

# Next

* Implement pd-send
* Implement stats interface
