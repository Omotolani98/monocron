# Monocrond

The monocron daemon that handles all unix sockets calls received from monocron runner to start and execute cron jobs on the machine level.

## Installation

Start Daemon

```shell
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/dev/monocrond-install.sh | bash
```

Check installation status

```shell
sudo systemctl status monocrond
```

## Uninstall

```shell
{
  sudo systemctl stop monocrond
  sudo systemctl disable monocrond
  sudo rm /etc/systemd/system/monocrond.service
  sudo rm /usr/local/bin/monocrond
  sudo rm /var/log/monocron/monocron.log
  sudo systemctl daemon-reload
}
```

## Socket APIs

Schedule a job

```shell
curl --unix-socket /run/monocron/monocron.sock \
  -H "Content-Type: application/json" \
  -X POST http://unix/schedule \
  -d '{"name":"backup","schedule":"*/5 * * * * *","timezone":"Africa/Lagos","timeout":10,"argv":["mkdir", "HelloFolder"]}'
```

List Jobs

```shell
curl --unix-socket /run/monocron/monocron.sock http://unix/list | jq
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
