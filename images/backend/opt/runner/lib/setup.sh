#!/usr/bin/env bash

CONFIG_PATH="${CONFIG_PATH:-.opencode-runner.yml}"
AGENT_PROVIDER="${AGENT_PROVIDER:-opencode}"
MODEL="${AGENT_MODEL:-opencode/big-pickle}"
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

looks_like_base64_secret() {
  local value="$1"
  [[ "${value}" =~ ^[A-Za-z0-9+/]+={0,2}$ ]] && (( ${#value} % 4 == 0 ))
}

decode_secret_if_needed() {
  local value="$1"
  local decoded
  if ! looks_like_base64_secret "${value}"; then
    printf '%s' "${value}"
    return
  fi
  decoded="$(printf '%s' "${value}" | base64 -d 2>/dev/null || true)"
  case "${decoded}" in
    github_pat_*|ghp_*|gho_*|ghu_*|ghs_*|ghr_*|sk-*|lin_api_*|eyJ*)
      printf '%s' "${decoded}"
      ;;
    *)
      printf '%s' "${value}"
      ;;
  esac
}

normalize_secrets() {
  if [[ -n "${GIT_TOKEN:-}" ]]; then
    GIT_TOKEN="$(decode_secret_if_needed "${GIT_TOKEN}")"
    export GIT_TOKEN
  fi
  if [[ -n "${GITHUB_TOKEN:-}" ]]; then
    GITHUB_TOKEN="$(decode_secret_if_needed "${GITHUB_TOKEN}")"
    export GITHUB_TOKEN
  fi
  if [[ -n "${OPENCODE_API_KEY:-}" ]]; then
    OPENCODE_API_KEY="$(decode_secret_if_needed "${OPENCODE_API_KEY}")"
    export OPENCODE_API_KEY
  fi
  if [[ -n "${KILO_API_KEY:-}" ]]; then
    KILO_API_KEY="$(decode_secret_if_needed "${KILO_API_KEY}")"
    export KILO_API_KEY
  fi
  if [[ -n "${LINEAR_API_KEY:-}" ]]; then
    LINEAR_API_KEY="$(decode_secret_if_needed "${LINEAR_API_KEY}")"
    export LINEAR_API_KEY
  fi
}

configure_git_credentials() {
  git config --global user.name "${GIT_AUTHOR_NAME:-OpenCode Runner}"
  git config --global user.email "${GIT_AUTHOR_EMAIL:-opencode-runner@example.invalid}"
  git config --global --add safe.directory /workspace/repo

  if [[ -n "${GIT_TOKEN:-}" ]]; then
    git config --global url."https://x-access-token:${GIT_TOKEN}@github.com/".insteadOf "https://github.com/"
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
  AGENT_PROVIDER="$(yaml_value ".agent.provider" "${AGENT_PROVIDER}")"
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

clone_harness_repo() {
  if [[ -z "${HARNESS_REPO_URL:-}" ]]; then
    return
  fi

  log "cloning harness repository"
  git clone --depth 1 "${HARNESS_REPO_URL}" /workspace/my-harnesses
}

prepare_kilo_config() {
  case "${AGENT_PROVIDER}" in
    kilo|kilocode) ;;
    *) return ;;
  esac

  mkdir -p "${HOME}/.config/kilo"
  jq -n \
    --arg model "${MODEL}" \
    '{
      "$schema": "https://app.kilo.ai/config.json",
      model: $model,
      permission: "allow",
      formatter: false,
      lsp: false
    }' > "${HOME}/.config/kilo/kilo.json"
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

  if [[ -z "${ISSUE_CONTEXT}" ]]; then
    log "Linear issue context not found for ${LINEAR_ISSUE_KEY}; continuing without ticket details"
  else
    log "loaded Linear issue context for ${LINEAR_ISSUE_KEY}"
  fi
}
