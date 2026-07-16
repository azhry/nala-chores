# Handoff Source

The implementation in this repository was scaffolded from `/Users/azharyarliansyah/Downloads/HANDOFF-minikube-opencode.md`.

Keep that source handoff nearby while evolving the MVP; this repo implements the local Minikube path:

- CLI submit/status/list/stop
- manager API with idempotent `request_id`
- Kubernetes Job rendering and polling
- sandbox worker image with OpenCode plan/edit/validate/fix lifecycle
- example OpenCode harness files

