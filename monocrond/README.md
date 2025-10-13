# Monocrond

The monocron daemon that handles all unix sockets calls received from monocronctl to start and execute cron jobs on the machine level.

## Usage

Start Daemon

```shell
go run main.go &> monocron.logs &
```
## Socket APIs

Schedule a job

```shell
curl --unix-socket /tmp/monocron.sock \
  -H "Content-Type: application/json" \
  -X POST http://unix/schedule \
  -d '{"name":"backup","schedule":"*/5 * * * * *","timezone":"Africa/Lagos","timeout":10,"argv":["mkdir", "HelloFolder"]}'
```

List Jobs

```shell
curl --unix-socket /tmp/monocron.sock http://unix/list | jq
```

Get One Job

```shell
curl --unix-socket /tmp/monocron.sock http://unix/jobs/2 | jq
```

Remove Job

```shell
curl --unix-socket /tmp/monocron.sock http://unix/delete/1 | jq
```

Shutdown and Prune Crons

```shell
curl --unix-socket /tmp/monocron.sock http://unix/shutdown
```
