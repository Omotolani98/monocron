# Monocron

## Introduction

Monocron is simply a git-driven cron scheduling manager. The idea is to have a to track cron scheduled on a server or multiple servers.
I wrote this project basically as a pursuit of building at least one open source self-hosted project.
To be very open, the idea was given to me by ChatGPT (ikr! ridiculous!) but I did have to make major adjustments to fit how I perceived the problem.

I imagined how so much easier it is to set up a job on fleets of servers to, let's say, backup-up databases or update packages or update ssl-certificates.
Then I just went down a rabbit-hole of thinking what ifs; what if we can run a command and list our scheduled jobs and see who made what changes and when is the next execution of a job? I tried to manage it by just releasing what I have currently and then I can push new updates. This project is not currently versioned.

## Technology

Golang, GRPC, Postgresql and Encore.

All the technologies here are chosen basically for performance and my own learning.

## How to run?
You simply need encore.

```bash
encore run
```
