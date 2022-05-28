# Nezabx

## Overview

Nezabx (njɛ-za-bɪks) - simple periodic job scheduler with notifications.

Supported notification channels:
- email

Supported jobs: 
- arbitrary shell commands

## Build

```
$ make
go build -ldflags "-X main.Version=v0.1.0" -o ./bin/nezabx
```

## Config

See configuration example with comments in [example.yaml](/example.yaml)

## Run

```
$ ./bin/nezabx -c config.yaml 
2022-05-28T22:47:10.979+0300    info    application started
2022-05-28T22:48:00.049+0300    info    script run ok   {"cmd": "./healthcheck.sh arg", "next iteration at": "2022-05-28T22:49:00.000+0300"}
^C2022-05-28T22:48:05.354+0300  info    application received termination signal
```

## License

Source code is available under the [MIT License](/LICENSE).