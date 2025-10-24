# Monocron

A simple cli tool to manage cron jobs on multiple machines.

Currently, this only supports one machine, in further updates yamls will be updated to support
multiple machines.

## monocrond

Daemon that handles system level executions of crons.
Installation guide is [Monocrond Doc](docs/monocron-daemon.md)

## monocron-runner

An API layer that expose API to your machine. It powers
webhook from Github and (in the future) monocronctl.

See [Monocron Runner Doc](docs/runner.md)
