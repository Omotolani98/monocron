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
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocronctl.sh | bash

# monocron-controller (control plane)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocron-controller.sh | bash

# monocrond (host-local scheduler/executor)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocrond.sh | bash

# monocron-runner (host agent)
curl -fsSL https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocron-runner.sh | bash
```

### Windows (PowerShell)

```powershell
# monocronctl (CLI)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocronctl.ps1 | iex

# monocron-controller (control plane)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocron-controller.ps1 | iex

# monocrond (host-local scheduler/executor)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocrond.ps1 | iex

# monocron-runner (host agent)
irm https://raw.githubusercontent.com/Omotolani98/monocron/v0.3.0/scripts/install/install-monocron-runner.ps1 | iex
```

All scripts download the latest GitHub release for the current OS/architecture, extract the binary, and place it in the system or user path.

## Quick Start

```bash
# 1. Start PostgreSQL and run the controller
export MONOCRON_DATABASE_URL="postgres://user:pass@localhost/monocron?sslmode=disable"
monocron-controller

# 2. Install monocrond on a target host and start it
sudo monocrond

# 3. Log in from your workstation
monocronctl login http://localhost:8080 admin-key

# 4. Create an enrollment token for the host
monocronctl runner token zone=home os=linux

# 5. On the host, join the runner
monocronctl runner join <token>
sudo systemctl start monocron-runner

# 6. Create a schedule
monocronctl schedule create backup "0 2 * * *" 10m /usr/local/bin/backup.sh zone=home

# 7. Watch executions
monocronctl execution list
monocronctl execution logs <execution-id>
```

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
monocronctl update --dry-run v0.3.0
monocronctl update v0.3.0
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

- `MONOCRON_DATABASE_URL` — required PostgreSQL DSN
- `MONOCRON_CONTROLLER_LISTEN` — default `:8080`
- `MONOCRON_METRICS_LISTEN` — default `:9090`
- `MONOCRON_LOG_LEVEL` — debug, info, warn, error

### monocron-runner

- `MONOCRON_RUNNER_CONFIG` — config file path, default platform config dir (`~/.config/monocron/runner.json` on Linux)
- `MONOCRON_CONTROLLER_URL` — required controller URL (overrides config file)
- `MONOCRON_DAEMON_SOCKET` — default `/run/monocron/monocrond.sock`
- `MONOCRON_RUNNER_STATE_PATH` — default platform config dir (`~/.config/monocron/runner.state` on Linux)
- `MONOCRON_RUNNER_TYPE` — default `bare_metal`
- `MONOCRON_RUNNER_LABELS` — comma-separated `key=value` labels

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
