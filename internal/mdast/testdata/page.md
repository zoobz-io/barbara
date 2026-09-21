---
title: Deploying the API
description: A realistic page that mixes every construct.
---

# Deploying the API

This page walks through a deploy. It mixes **strong**, *emphasis*,
`inline code`, and a [link to the runbook](https://example.com/runbook "Runbook").

## Prerequisites

You need the following before you start:

1. A built binary.
2. Access to the cluster.
3. The `deploy` role.

> **Note:** never deploy on a Friday.
>
> > And never on a Friday afternoon.

## Steps

- [x] Build the binary
- [ ] Push the image
- [ ] Roll out

Run the deploy command:

```sh title="deploy.sh"
./deploy --env prod --wait
```

The rollout status table:

| stage   | status  | owner |
| :------ | :-----: | ----: |
| build   | done    | ci    |
| push    | pending | ci    |
| rollout | blocked | you   |

![deploy diagram](../assets/deploy.png "Deploy flow")

---

See the <https://example.com/docs> for more, or the &copy; notice below.
