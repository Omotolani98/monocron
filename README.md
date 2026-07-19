# Monocron

A self-hosted scheduler for tasks on VMs, bare metals, or container environments.

## Components

- **monocron-controller** — central control plane that owns desired state, runners, schedules, audit logs, and the public API.
- **monocron-runner** — host agent that enrolls with the controller, reconciles assignments, and reports status over outbound HTTPS.
- **monocrond** — host-local daemon that schedules and executes commands through a protected Unix socket.
- **monocronctl** — administrative CLI for operators.

## Install

Each component can be installed with a one-liner. Scripts are hosted at `https://raw.githubusercontent.com/Omotolani98/monocron/dev/scripts/install/`.

### Linux / macOS (curl)

```bash
# monocronctl (CLI)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocronctl.sh | bash

# monocron-controller (control plane)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocron-controller.sh | bash

# monocrond (host-local scheduler/executor)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocrond.sh | bash

# monocron-runner (host agent)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocron-runner.sh | bash
```

### Windows (PowerShell)

```powershell
# monocronctl (CLI)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocronctl.ps1 | iex

# monocron-controller (control plane)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocron-controller.ps1 | iex

# monocrond (host-local scheduler/executor)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocrond.ps1 | iex

# monocron-runner (host agent)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocron-runner.ps1 | iex
```

All scripts download the latest GitHub release for the current OS/architecture, extract the binary, and place it in the system or user path.

On Linux, the component install scripts also copy the systemd unit files to `/etc/systemd/system`, create the `monocron` user and required directories, and enable the services where appropriate.

## Quick Start

Monocron has three logical roles. They can all run on one server or be split across machines.

```text
Controller server          Runner servers
├─ monocron-controller     ├─ monocron-runner
└─ PostgreSQL              ├─ monocrond
                           └─ scheduled jobs

Operator machine
└─ monocronctl
```

- The controller stores state and exposes the API.
- Each runner host runs `monocron-runner` and `monocrond`. Jobs execute on the runner host.
- Runner hosts only need outbound access to the controller.

### 1. Controller server

Install the binary and systemd unit:

```bash
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocron-controller.sh | bash
```

Create a PostgreSQL database, then edit the controller environment file:

```bash
sudo nano /etc/monocron/controller.env
```

Set at least:

```ini
MONOCRON_DATABASE_URL=postgres://user:password@localhost/monocron?sslmode=disable
```

Start and verify:

```bash
sudo systemctl enable --now monocron-controller
systemctl status monocron-controller --no-pager -l
curl http://127.0.0.1:8080/api/v1/health
```

Note the controller's reachable URL (`http://CONTROLLER_IP:8080`).

### 2. Operator machine

Install the CLI:

```bash
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocronctl.sh | bash
```

Log in to the controller. Admin authentication is currently a placeholder, so any key works:

```bash
monocronctl login http://CONTROLLER_IP:8080 placeholder
```

Create an enrollment token for the runner host:

```bash
monocronctl runner token zone=home os=linux
```

Copy the printed token.

### 3. Runner host

Install the daemon and runner. The install script registers the systemd units and creates the `monocron` user:

```bash
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocrond.sh | bash
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.4.0/scripts/install/install-monocron-runner.sh | bash
```

Edit the runner environment file:

```bash
sudo nano /etc/monocron/runner.env
```

Set the controller URL:

```ini
MONOCRON_CONTROLLER_URL=http://CONTROLLER_IP:8080
```

Start `monocrond`:

```bash
sudo systemctl enable --now monocrond
```

Enroll the runner as the `monocron` user so the state file is readable by the service:

```bash
sudo -u monocron monocronctl \
  --controller-url http://CONTROLLER_IP:8080 \
  runner \
  --state /var/lib/monocron/runner.state \
  join '<PASTE_TOKEN_HERE>'
```

Start the runner:

```bash
sudo systemctl enable --now monocron-runner
systemctl status monocron-runner --no-pager -l
```

### 4. Create a schedule

From the operator machine:

```bash
monocronctl schedule create backup "0 2 * * *" 10m /usr/local/bin/backup.sh zone=home
monocronctl schedule list
monocronctl execution list
monocronctl execution logs <execution-id>
```

Labels on the schedule (`zone=home os=linux`) must match the labels given to the enrollment token.

### Same-server development

To run everything on one machine, use `http://127.0.0.1:8080` for the controller URL.

```bash
export MONOCRON_DATABASE_URL="postgres://user:pass@localhost/monocron?sslmode=disable"
sudo systemctl enable --now monocron-controller
sudo systemctl enable --now monocrond
monocronctl login http://127.0.0.1:8080 placeholder
TOKEN=$(monocronctl runner token zone=home os=linux | awk '/Enrollment token:/{print $3}')
sudo -u monocron monocronctl --controller-url http://127.0.0.1:8080 runner --state /var/lib/monocron/runner.state join "$TOKEN"
sudo systemctl enable --now monocron-runner
```

## Troubleshooting

Check service status and recent logs:

```bash
systemctl status monocron-controller --no-pager -l
systemctl status monocrond --no-pager -l
systemctl status monocron-runner --no-pager -l

journalctl -u monocron-controller -n 100 --no-pager
journalctl -u monocrond -n 100 --no-pager
journalctl -u monocron-runner -n 100 --no-pager
```

Common issues:

- **"Controller URL is required"** — run `monocronctl login <url> <key>` or pass `--controller-url`.
- **Runner fails to start** — verify `/etc/monocron/runner.env` has `MONOCRON_CONTROLLER_URL`, the runner was enrolled, and `/var/lib/monocron/runner.state` is owned by `monocron:monocron`.
- **Jobs never run** — confirm schedule labels match runner labels (`monocronctl runner list`).
- **monocrond.sock missing** — ensure `monocrond` started before `monocron-runner`.

## Configuration

`monocronctl` stores its config in the platform-specific user config directory:

- Linux: `~/.config/monocronctl/config.json`
- macOS: `~/Library/Application Support/monocronctl/config.json`
- Windows: `%AppData%\monocronctl\config.json`

The config file is created automatically on the first operational command. Override the path with `--config` or `MONOCRON_CONFIG`.

`monocron-runner` uses a separate config file in the same config directory:

- Linux / macOS: `<config-dir>/monocron/runner.json`
- Windows: `<config-dir>\monocron\runner.json`

It is also created automatically and can be overridden with `MONOCRON_RUNNER_CONFIG`.

### Precedence

Command-line flags > environment variables > config file > built-in defaults.

## Updating

Update all installed Monocron binaries to the latest release:

```bash
monocronctl update
```

Update to a specific version (dry run first):

```bash
monocronctl update --dry-run v0.4.0
monocronctl update v0.4.0
```

The command updates `monocronctl`, `monocron-controller`, `monocron-runner`, and `monocrond` in the directory that contains `monocronctl` (override with `--bin-dir`). Updates are supported on Linux and macOS.

## Repository Layout

```text
cmd/
  monocronctl/        Administrative CLI
  monocron-controller/ Central control plane
  monocron-runner/     Host agent
  monocrond/           Local scheduler and executor
internal/
  contracts/           Shared wire contracts
  controller/          Controller API, store, placement
  runner/              Runner agent clients and reconciliation
  daemon/              Daemon scheduler, executor, store, API
  cli/                 CLI client
  platform/            Config, logging, version
migrations/controller/ PostgreSQL migrations
migrations/daemon/     SQLite migrations
deploy/                Systemd units and Dockerfiles
```

## Building

```bash
go build ./cmd/...
```

## Tests

```bash
go test ./...
```

## Environment Variables

### monocron-controller

Set these in `/etc/monocron/controller.env`:

- `MONOCRON_DATABASE_URL` — required PostgreSQL DSN
- `MONOCRON_CONTROLLER_LISTEN` — default `:8080`
- `MONOCRON_METRICS_LISTEN` — default `:9090`
- `MONOCRON_LOG_LEVEL` — debug, info, warn, error

### monocron-runner

Set these in `/etc/monocron/runner.env`:

- `MONOCRON_CONTROLLER_URL` — required controller URL (overrides config file)
- `MONOCRON_RUNNER_STATE_PATH` — default platform config dir (`~/.config/monocron/runner.state` on Linux)
- `MONOCRON_RUNNER_TYPE` — default `bare_metal`
- `MONOCRON_RUNNER_LABELS` — comma-separated `key=value` labels
- `MONOCRON_POLL_INTERVAL` or `MONOCRON_RUNNER_POLL_INTERVAL` — default `10s`
- `MONOCRON_HEARTBEAT_INTERVAL` or `MONOCRON_RUNNER_HEARTBEAT_INTERVAL` — default `10s`

The runner also reads `MONOCRON_RUNNER_CONFIG` for the config file path and `MONOCRON_DAEMON_SOCKET` for the local daemon socket.

### monocrond

- `MONOCRON_DAEMON_SOCKET` — default `/run/monocron/monocrond.sock`
- `MONOCRON_DAEMON_STATE_PATH` — default `/var/lib/monocron/monocrond.db`
- `MONOCRON_MAX_LOG_BYTES` — default `262144`
- `MONOCRON_DEFAULT_TIMEOUT` — default `30s`

## Sequence

```mermaid
sequenceDiagram
    actor User
    participant CLI as monocronctl
    participant Controller
    participant Runner as monocron-runner
    participant Daemon as monocrond
    participant Process as OS Process

    User->>CLI: Create schedule
    CLI->>Controller: POST /api/v1/schedules
    Controller->>Controller: Select matching runners
    Controller->>Controller: Persist assignment
    Controller-->>CLI: Schedule pending

    loop Long poll
        Runner->>Controller: GET /api/v1/runners/assignments
        Controller-->>Runner: Return desired assignments
    end

    Runner->>Daemon: PUT /v1/assignments/{id}
    Daemon->>Daemon: Validate, persist, schedule
    Daemon-->>Runner: Applied
    Runner->>Controller: Submit observation

    Note over Daemon: Execution time arrives
    Daemon->>Process: Spawn command
    Daemon->>Daemon: Record execution result
    Runner->>Daemon: Poll executions
    Runner->>Controller: Submit execution events

    CLI->>Controller: GET execution status/logs
    Controller-->>CLI: Result
```
