# Nala Chores

Nala Chores is my personal remote coding-agent runner. It lets me save project profiles, then delegate coding tasks from a phone or laptop into sandboxed Minikube jobs that can clone a repo, load Linear context, run OpenCode or KiloCode, push changes, and create a pull request.

The main use case is mobile-friendly delegation: configure a repository once, open the web UI later, choose the configuration, write the task prompt, and let the runner do the work in an isolated Kubernetes workspace while I am away from my laptop.

## Components

- `cmd/runner-cli`: submit tasks and poll status.
- `cmd/runner-manager`: HTTP API and mobile-friendly web UI for saved configurations, run launch, and session history.
- `images/backend`: sandbox image that clones the target repo, loads optional harness code, runs the selected agent, and pushes results.
- `deploy/minikube`: namespace, RBAC, and manager deployment manifests.
- `examples`: sample `.opencode-runner.yml` and OpenCode agents/commands for target repositories.

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
  --mr
```

Check status:

```bash
go run ./cmd/runner-cli status --last
```

## Minikube Bootstrap

This is designed to run locally in Minikube, with a web UI that works well on mobile-sized screens. Use `http://chores.nala.local` for a stable local browser URL, or expose the manager through your own tunnel, VPN, or reverse proxy when you want to start and inspect runs from a smartphone.

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

The web UI stores credentials per configuration. Each saved project profile syncs one stable Kubernetes Secret, and all runs for that profile reuse it instead of generating a new secret for every session.

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

Then open the web UI from the machine running Minikube:

```bash
minikube service -n agent-runner runner-manager
```

To use a stable local hostname through the Minikube ingress controller:

```bash
kubectl apply -f deploy/minikube/ingress.yaml
./scripts/chores-local-url.sh
```

The helper maps `chores.nala.local` to `127.0.0.1` in `/etc/hosts` and runs a local port-80 forward to the Minikube ingress controller. Leave it running, then open:

```text
http://chores.nala.local
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

1. Open **Configurations** and save a project profile with repo URL, optional harness repository URL, agent provider/model, branch, sandbox size, GitHub API key, Linear API key, and OpenCode or Kilo API key.
2. Open **Run**, select the saved profile, enter the coding-task prompt, optionally add a Linear issue key, and start the sandboxed session.
3. Open **History** to reopen previous sessions, read the chat-style transcript, inspect job metadata, and jump to the created pull request.
