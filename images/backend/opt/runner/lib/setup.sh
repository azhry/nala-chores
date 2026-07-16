#!/usr/bin/env bash

CONFIG_PATH="${CONFIG_PATH:-.opencode-runner.yml}"
MODEL="opencode/big-pickle"
AGENT_MODE="run"
PLAN_AGENT="plan-agent"
EDIT_AGENT="edit-agent"
VERIFY_AGENT="verify-agent"
PLAN_COMMAND=""
VERIFY_COMMAND=""
MAX_FIX_ATTEMPTS=3
SESSION_TIMEOUT="2h"
INIT_SCRIPT=""
ISSUE_CONTEXT=""

configure_git_credentials() {
  git config --global user.name "${GIT_AUTHOR_NAME:-OpenCode Runner}"
  git config --global user.email "${GIT_AUTHOR_EMAIL:-opencode-runner@example.invalid}"
  git config --global --add safe.directory /workspace/repo

  if [[ -n "${GIT_TOKEN:-}" ]]; then
    git config --global url."https://oauth2:${GIT_TOKEN}@github.com/".insteadOf "https://github.com/"
    git config --global url."https://oauth2:${GIT_TOKEN}@gitlab.com/".insteadOf "https://gitlab.com/"
  fi
}

yaml_value() {
  local expression="$1"
  local fallback="${2:-}"
  if [[ ! -f "${CONFIG_PATH}" ]]; then
    printf '%s' "${fallback}"
    return
  fi
  python3 - "$CONFIG_PATH" "$expression" "$fallback" <<'PY'
import re
import sys

path, expression, fallback = sys.argv[1:]
keys = expression.strip(".").split(".")
try:
    import yaml
except Exception:
    print(fallback)
    raise SystemExit

try:
    with open(path, "r", encoding="utf-8") as f:
        data = yaml.safe_load(f) or {}
    for key in keys:
        if isinstance(data, dict):
            data = data.get(key)
        else:
            data = None
        if data is None:
            print(fallback)
            raise SystemExit
    if isinstance(data, bool):
        print(str(data).lower())
    elif isinstance(data, (dict, list)):
        print(fallback)
    else:
        print(data)
except Exception:
    print(fallback)
PY
}

load_harness_config() {
  if [[ ! -f "${CONFIG_PATH}" ]]; then
    log "harness config ${CONFIG_PATH} not found; using defaults"
    return
  fi

  MODEL="$(yaml_value ".agent.model" "${MODEL}")"
  AGENT_MODE="$(yaml_value ".agent.mode" "${AGENT_MODE}")"
  PLAN_AGENT="$(yaml_value ".agent.agents.plan" "${PLAN_AGENT}")"
  EDIT_AGENT="$(yaml_value ".agent.agents.edit" "${EDIT_AGENT}")"
  VERIFY_AGENT="$(yaml_value ".agent.agents.verify" "${VERIFY_AGENT}")"
  INIT_SCRIPT="$(yaml_value ".sandbox.initScript" "${INIT_SCRIPT}")"
  PLAN_COMMAND="$(yaml_value ".harnesses.${HARNESS_NAME}.planCommand" "${PLAN_COMMAND}")"
  VERIFY_COMMAND="$(yaml_value ".harnesses.${HARNESS_NAME}.verifyCommand" "${VERIFY_COMMAND}")"
  MAX_FIX_ATTEMPTS="$(yaml_value ".harnesses.${HARNESS_NAME}.maxFixAttempts" "${MAX_FIX_ATTEMPTS}")"
  SESSION_TIMEOUT="$(yaml_value ".harnesses.${HARNESS_NAME}.sessionTimeout" "${SESSION_TIMEOUT}")"

  log "loaded harness ${HARNESS_NAME}: model=${MODEL} mode=${AGENT_MODE} maxFix=${MAX_FIX_ATTEMPTS}"
}

run_init_script() {
  if [[ -z "${INIT_SCRIPT}" ]]; then
    return
  fi
  if [[ ! -f "${INIT_SCRIPT}" ]]; then
    log "initScript ${INIT_SCRIPT} not found; skipping"
    return
  fi
  log "running initScript ${INIT_SCRIPT}"
  bash "${INIT_SCRIPT}"
}

prepare_opencode_config() {
  if [[ ! -f opencode.json ]]; then
    log "opencode.json not found; OpenCode will use defaults"
  fi
}

load_linear_issue_context() {
  if [[ -z "${LINEAR_API_KEY:-}" || -z "${LINEAR_ISSUE_KEY:-}" ]]; then
    return
  fi

  log "loading Linear issue context for ${LINEAR_ISSUE_KEY}"
  local payload
  payload="$(jq -n --arg key "${LINEAR_ISSUE_KEY}" '{
    query: "query Issue($key: String!) { issue(id: $key) { identifier title description url state { name } assignee { name } labels { nodes { name } } } }",
    variables: { key: $key }
  }')"

  ISSUE_CONTEXT="$(curl -fsS https://api.linear.app/graphql \
    -H "Authorization: ${LINEAR_API_KEY}" \
    -H "Content-Type: application/json" \
    --data "${payload}" | jq -r '
      .data.issue as $i |
      if $i == null then empty else
        "Linear: " + $i.identifier + "\n" +
        "Title: " + $i.title + "\n" +
        "State: " + ($i.state.name // "") + "\n" +
        "Assignee: " + ($i.assignee.name // "Unassigned") + "\n" +
        "URL: " + $i.url + "\n\n" +
        ($i.description // "")
      end
    ' || true)"
}
