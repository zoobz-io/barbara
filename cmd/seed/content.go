package main

// The pages the site script writes: documentation for Lumen, a fictional
// uptime and status-page product. Each constant is one version's full
// markdown; a V2 is the page after an edit, so the studio has real diffs to
// show between versions and between releases.

const indexV1 = `# Lumen

Lumen watches your endpoints from six regions and tells your users what is
going on before they ask. A status page, an incident timeline, and alerts that
respect on-call hours.

## Start here

- [Getting started](guides/getting-started.md) — a monitor and a status page in ten minutes.
- [Installation](guides/installation.md) — the CLI, the agent, and the Terraform provider.
- [API reference](reference/api.md) — every endpoint, with examples.

## Why Lumen

Most uptime tools check from one place and page everyone. Lumen checks from
many, agrees before it alerts, and writes the status page for you.
`

const indexV2 = `# Lumen

Lumen watches your endpoints from six regions and tells your users what is
going on before they ask. A status page, an incident timeline, and alerts that
respect on-call hours.

> **New in 2.3:** scheduled maintenance windows silence alerts and post to the
> status page automatically. See the [release notes](blog/release-notes.md).

## Start here

- [Getting started](guides/getting-started.md) — a monitor and a status page in ten minutes.
- [Installation](guides/installation.md) — the CLI, the agent, and the Terraform provider.
- [API reference](reference/api.md) — every endpoint, with examples.
- [Troubleshooting](guides/advanced/troubleshooting.md) — when a check disagrees with you.

## Why Lumen

Most uptime tools check from one place and page everyone. Lumen checks from
many, agrees before it alerts, and writes the status page for you.
`

const aboutV1 = `# About

Lumen is built by a small team that spent too many nights paged for a check
that failed from one region. Every feature starts from that night.

## Principles

1. **Agree before alerting.** A single failing probe is weather, not an outage.
2. **The status page is the source of truth.** If it is not on the page, it did not happen.
3. **Quiet by default.** Alerts go to the person on call, at hours they chose.

## Contact

Support: support@lumen.example
Status: https://status.lumen.example
`

const gettingStartedV1 = `# Getting started

This guide takes you from an empty account to a public status page with one
monitored endpoint.

## 1. Create a monitor

Add the URL you want watched. Lumen probes it every minute from six regions.

` + "```" + `sh
lumen monitors create --url https://api.example.com/health --name "API"
` + "```" + `

## 2. Watch the first checks land

` + "```" + `sh
lumen checks tail API
` + "```" + `

Each line is one probe: region, latency, status code.

## 3. Publish a status page

` + "```" + `sh
lumen pages create --name "Example status" --monitors API
` + "```" + `

The page is live at the URL the command prints. That is it.
`

const gettingStartedV2 = `# Getting started

This guide takes you from an empty account to a public status page with one
monitored endpoint. It takes about ten minutes.

## Before you begin

Install the CLI and sign in — see [Installation](installation.md).

## 1. Create a monitor

Add the URL you want watched. Lumen probes it every minute from six regions.

` + "```" + `sh
lumen monitors create --url https://api.example.com/health --name "API"
` + "```" + `

A probe passes when the endpoint answers 2xx within the timeout (default 10s).
Change either with ` + "`--expect`" + ` and ` + "`--timeout`" + `.

## 2. Watch the first checks land

` + "```" + `sh
lumen checks tail API
` + "```" + `

Each line is one probe: region, latency, status code. The monitor turns green
once four of six regions agree.

## 3. Publish a status page

` + "```" + `sh
lumen pages create --name "Example status" --monitors API
` + "```" + `

The page is live at the URL the command prints.

## Next

- Add more monitors and group them into components on the page.
- Set up [alert routing](../reference/api.md#alerts) so the right person is paged.
`

const installationV1 = `# Installation

Lumen has three installable pieces. Most teams only need the CLI.

## CLI

` + "```" + `sh
curl -fsSL https://get.lumen.example | sh
lumen login
` + "```" + `

Homebrew and Scoop packages are also available under the name ` + "`lumen`" + `.

## Agent

The agent runs inside your network and probes endpoints the public regions
cannot reach. It is a single static binary.

` + "```" + `sh
lumen agent install --token $LUMEN_AGENT_TOKEN
` + "```" + `

## Terraform provider

` + "```" + `hcl
terraform {
  required_providers {
    lumen = { source = "lumen/lumen", version = "~> 1.4" }
  }
}
` + "```" + `

Monitors, pages, and alert routes are all resources.
`

const cachingV1 = `# Caching

Lumen caches probe results for a short window so the status page and the API
agree with each other and stay fast under load.

## What is cached

| Data | TTL | Invalidated by |
|------|-----|----------------|
| Monitor state | 15s | any probe result |
| Status page render | 30s | monitor state change, incident update |
| Incident timeline | 60s | incident update |

## Bypassing the cache

Send ` + "`Cache-Control: no-cache`" + ` on API reads. The status page has no
bypass; it is meant to be the calm view.

## Edge caching

Public status pages sit behind a CDN with the same 30-second TTL. Custom
domains inherit it. You cannot lengthen the TTL — a status page that lies for
five minutes is worse than no page at all.
`

const apiV1 = `# API reference

Base URL: ` + "`https://api.lumen.example/v1`" + `. Every request carries a bearer
token. Responses are JSON.

## Monitors

### List monitors

` + "```" + `
GET /monitors
` + "```" + `

### Create a monitor

` + "```" + `
POST /monitors
{ "name": "API", "url": "https://api.example.com/health" }
` + "```" + `

### Delete a monitor

` + "```" + `
DELETE /monitors/{id}
` + "```" + `

## Status pages

### Create a page

` + "```" + `
POST /pages
{ "name": "Example status", "monitors": ["mon_123"] }
` + "```" + `

## Errors

Errors carry a ` + "`code`" + ` and a ` + "`message`" + `. Rate limits return 429 with
` + "`Retry-After`" + `.
`

const apiV2 = `# API reference

Base URL: ` + "`https://api.lumen.example/v1`" + `. Every request carries a bearer
token. Responses are JSON.

## Monitors

### List monitors

` + "```" + `
GET /monitors?limit=50&offset=0
` + "```" + `

### Create a monitor

` + "```" + `
POST /monitors
{ "name": "API", "url": "https://api.example.com/health", "expect": "2xx", "timeout_ms": 10000 }
` + "```" + `

### Pause and resume

` + "```" + `
POST /monitors/{id}/pause
POST /monitors/{id}/resume
` + "```" + `

### Delete a monitor

` + "```" + `
DELETE /monitors/{id}
` + "```" + `

## Status pages

### Create a page

` + "```" + `
POST /pages
{ "name": "Example status", "monitors": ["mon_123"] }
` + "```" + `

## Alerts

### Create a route

` + "```" + `
POST /alerts/routes
{ "monitor": "mon_123", "channel": "pagerduty", "hours": "09:00-18:00 Europe/Berlin" }
` + "```" + `

Outside the route's hours, alerts queue and are delivered at the next opening.

## Errors

Errors carry a ` + "`code`" + ` and a ` + "`message`" + `. Rate limits return 429 with
` + "`Retry-After`" + `. Validation failures return 422 with a ` + "`fields`" + ` map.
`

const cliV1 = `# CLI reference

` + "`lumen`" + ` is the command-line client. Every subcommand accepts ` + "`--json`" + `
for machine-readable output.

## Global flags

| Flag | Meaning |
|------|---------|
| ` + "`--profile`" + ` | Named credentials to use |
| ` + "`--json`" + ` | Print JSON instead of a table |
| ` + "`--quiet`" + ` | Suppress progress output |

## Commands

- ` + "`lumen login`" + ` — authenticate in the browser.
- ` + "`lumen monitors list|create|delete`" + `
- ` + "`lumen checks tail <monitor>`" + ` — stream probe results.
- ` + "`lumen pages list|create|publish`" + `
- ` + "`lumen incidents open|update|resolve`" + `
- ` + "`lumen agent install|status`" + `
`

const troubleshootingV1 = `# Troubleshooting

## A monitor is red but the endpoint works for me

You are one region. Run ` + "`lumen checks tail <monitor>`" + ` and look at which
regions fail. A single failing region is usually a routing issue between that
region and your host, not your host.

## Alerts arrive late

Check the route's hours. Alerts raised outside them queue until the window
opens. ` + "`lumen alerts routes list`" + ` shows the schedule.

## The status page shows stale data

The page caches for 30 seconds. If it is older than that, check the incident
timeline: a resolved incident that was never marked resolved keeps the page
degraded.
`

const helloWorldV1 = `# Hello, world

Today Lumen leaves private beta.

We started with one question: why does every uptime tool page the whole team
because one probe in one region timed out? Eighteen months later the answer is
a product that checks from six regions, waits for a quorum, and writes the
status page for you.

Thanks to the two hundred teams who ran the beta and sent us their worst
pages. Every one of them is a test case now.

Pricing and a free tier are on the [site](https://lumen.example/pricing).
`

const releaseNotesV1 = `# Release notes

## 2.3 — maintenance windows

- Scheduled maintenance windows silence alerts for the monitors they cover and
  post a notice to the status page for their duration.
- ` + "`lumen monitors pause`" + ` and ` + "`resume`" + ` in the CLI and the API.
- Validation errors now name the failing fields.

## 2.2 — alert routing

- Routes: send a monitor's alerts to a channel during a schedule.
- PagerDuty and Opsgenie channels.

## 2.1

- Agent: probe private endpoints from inside your network.
- Terraform provider 1.4 with page components.
`
