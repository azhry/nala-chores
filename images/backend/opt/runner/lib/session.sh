#!/usr/bin/env bash

OPENCODE_ATTACH=""

wait_for_opencode_server() {
  local attempts=0
  while (( attempts < 30 )); do
    if curl -fsS http://127.0.0.1:4096 >/dev/null 2>&1; then
      return
    fi
    attempts=$((attempts + 1))
    sleep 1
  done
  log "opencode serve did not expose an HTTP health response; continuing after startup delay"
}

run_session() {
  local phase="$1"
  local agent="$2"
  local prompt="$3"
  local session_id="${4:-}"
  local session_args=()
  local attach_args=()

  if [[ -n "${session_id}" ]]; then
    session_args=(--session "${session_id}")
  fi
  if [[ -n "${OPENCODE_ATTACH:-}" ]]; then
    attach_args=(${OPENCODE_ATTACH})
  fi

  timeout "${SESSION_TIMEOUT}" \
    opencode run --auto --format json \
      "${attach_args[@]}" \
      --model "${MODEL}" \
      --agent "${agent}" \
      --dir "/workspace/repo/${WORK_DIR}" \
      --title "${phase}-${REQUEST_ID}" \
      "${session_args[@]}" \
      "${prompt}" | tee "/tmp/${phase}.jsonl"
}

parse_session_id() {
  local file="$1"
  jq -r '
    select(type == "object") |
    (.sessionID // .session_id // .session.id // .sessionID? // empty)
  ' "${file}" 2>/dev/null | awk 'NF { value=$0 } END { print value }'
}

parse_final_text() {
  local file="$1"
  jq -r '
    select(type == "object") |
    (.text // .message // .content // .data.text // empty)
  ' "${file}" 2>/dev/null | awk 'NF { print }'
}

parse_verdict() {
  local file="$1"
  local final
  final="$(parse_final_text "${file}" | tail -40)"
  if printf '%s\n' "${final}" | grep -Eq '(^|[^A-Z])PASS([^A-Z]|$)'; then
    printf 'PASS'
    return
  fi
  printf 'FAIL'
}

