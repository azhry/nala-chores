# Nala Chores

Local Minikube MVP for running OpenCode headless in an ephemeral Kubernetes Job, with saved GitHub, Linear, and OpenCode configurations.

## Components

- `cmd/runner-cli`: submit tasks and poll status.
- `cmd/runner-manager`: HTTP API and web UI that stores configurations, creates Jobs, and keeps run history/logs.
- `images/backend`: sandbox image that clones the repo and runs OpenCode phases.
- `deploy/minikube`: namespace, RBAC, and manager deployment manifests.
- `examples`: sample `.opencode-runner.yml` and OpenCode agents/commands.

## Run Manager Locally

```bash
go run ./cmd/runner-manager
```

Set `RUNNER_APPLY_JOBS=false` to render Jobs without applying them.

```bash
RUNNER_APPLY_JOBS=false go run ./cmd/runner-manager
```

Submit from a git checkout:

```bash
go run ./cmd/runner-cli submit \
  --prompt "Implement the requested change and update tests." \
  --harness-repo https://github.com/your-org/my-harnesses.git \
  --agent-provider kilocode \
  --agent-model kilo/kilo-auto/free \
  --workdir . \
  --no-mr
```

Check status:

```bash
go run ./cmd/runner-cli status --last
```

## Minikube Bootstrap

```bash
minikube start --cpus 6 --memory 12288 --driver docker
minikube addons enable metrics-server
kubectl apply -f deploy/minikube/namespace.yaml
kubectl apply -f deploy/minikube/rbac.yaml
```

Create fallback secrets:

```bash
kubectl -n agent-runner create secret generic runner-secrets \
  --from-literal=git_token="$GIT_ACCESS_TOKEN" \
  --from-literal=opencode_api_key="$OPENCODE_API_KEY" \
  --from-literal=anthropic_api_key="$ANTHROPIC_API_KEY" \
  --from-literal=openai_api_key="$OPENAI_API_KEY"
```

The web UI can also store per-configuration GitHub, Linear, and OpenCode keys. Each configuration syncs one stable Kubernetes Secret, and all runs for that configuration reuse it.

Build and load images:

```bash
docker build -t opencode-runner-backend:kilocode images/backend
docker build -t opencode-runner-manager:kilocode -f Dockerfile.manager .
minikube image load opencode-runner-backend:kilocode
minikube image load opencode-runner-manager:kilocode
```

Deploy manager:

```bash
kubectl apply -f deploy/minikube/runner-manager.yaml
```

Then open the web UI:

```bash
minikube service -n agent-runner runner-manager
```

You can also port-forward manually and use `runner-cli` with `RUNNER_API_URL=http://127.0.0.1:8080`.

```bash
kubectl -n agent-runner port-forward svc/runner-manager 8080:8080
```

## Target Repo Harness

Copy `examples/.opencode-runner.yml` and `examples/sample-repo-harness/.opencode` into the target repository, then customize commands and agents for that repo.

The sample harness defaults to OpenCode's free `opencode/big-pickle` model.

Configurations can also point at an external harness repository. When `harness_repo_url` is set, the worker clones that repository to `/workspace/my-harnesses` before running OpenCode, so target repo instructions can reference harness paths such as `../my-harnesses/agent-spec-ops`.

Set `agent_provider` to `opencode` or `kilocode`. OpenCode defaults to `opencode/big-pickle`; KiloCode defaults to `kilo/kilo-auto/free` and reads its credential from the saved Kilo API key.

## Web UI Flow

1. Open **Configurations** and save a configuration with repo URL, optional harness repository URL, agent provider/model, branch, GitHub API key, Linear API key, and OpenCode or Kilo API key.
2. Open **Run Session**, select a configuration, enter a prompt, optionally add a Linear issue key, and run it.
3. Open **History** to inspect sessions for each configuration and view stored logs.
