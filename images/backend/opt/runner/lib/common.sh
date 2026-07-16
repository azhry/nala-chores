#!/usr/bin/env bash

log() {
  printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*" >&2
}

start_heartbeat() {
  local path="$1"
  (
    while true; do
      date -u +%s > "${path}"
      sleep 10
    done
  ) &
}

json_escape() {
  jq -Rs .
}

write_result() {
  local status="$1"
  local message="$2"
  local session_id="${3:-}"
  local mr_url="${4:-}"
  jq -n \
    --arg request_id "${REQUEST_ID}" \
    --arg status "${status}" \
    --arg message "${message}" \
    --arg edit_session_id "${session_id}" \
    --arg mr_url "${mr_url}" \
    --arg completed_at "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    '{
      request_id: $request_id,
      status: $status,
      message: $message,
      edit_session_id: $edit_session_id,
      mr_url: $mr_url,
      completed_at: $completed_at
    }' | tee "${RESULT_FILE}"
}

bool() {
  case "${1:-}" in
    true|TRUE|1|yes|YES) printf 'true' ;;
    *) printf 'false' ;;
  esac
}

