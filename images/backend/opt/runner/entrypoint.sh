#!/usr/bin/env bash
set -Eeuo pipefail

source /opt/runner/lib/common.sh
source /opt/runner/lib/setup.sh
source /opt/runner/lib/session.sh
source /opt/runner/lib/git-push.sh
source /opt/runner/lib/create-mr.sh

REQUEST_ID="${REQUEST_ID:?REQUEST_ID is required}"
REPO_URL="${REPO_URL:?REPO_URL is required}"
SOURCE_BRANCH="${SOURCE_BRANCH:?SOURCE_BRANCH is required}"
PROMPT="${PROMPT:?PROMPT is required}"
WORK_DIR="${WORK_DIR:-.}"
HARNESS_NAME="${HARNESS_NAME:-default}"
RESULT_FILE="${RESULT_FILE:-/tmp/run-result.json}"
CREATE_MR="${CREATE_MR:-false}"
PUSH_CHANGES="${PUSH_CHANGES:-false}"

mkdir -p /workspace /tmp/opencode-runner
start_heartbeat /tmp/opencode-runner/heartbeat
trap 'write_result failed "unexpected failure on line ${LINENO}" ""' ERR

log "setup: configuring git and cloning ${SOURCE_BRANCH}"
configure_git_credentials
git clone --depth 1 --branch "${SOURCE_BRANCH}" "${REPO_URL}" /workspace/repo
cd /workspace/repo

load_harness_config
run_init_script
prepare_opencode_config
load_linear_issue_context

cd "/workspace/repo/${WORK_DIR}"

if [[ "${AGENT_MODE}" == "serve" ]]; then
  export OPENCODE_SERVER_PASSWORD="${RUNNER_INTERNAL_TOKEN:-${REQUEST_ID}}"
  opencode serve --hostname 127.0.0.1 --port 4096 > /tmp/opencode-serve.log 2>&1 &
  OPENCODE_ATTACH="--attach http://127.0.0.1:4096"
  wait_for_opencode_server
fi

PLAN_TEXT=""
if [[ -n "${PLAN_COMMAND}" ]]; then
  log "planning: /${PLAN_COMMAND}"
  run_session plan "${PLAN_AGENT}" "BLOCKING: run /${PLAN_COMMAND} first. Task: ${PROMPT}${ISSUE_CONTEXT:+

Issue context:
${ISSUE_CONTEXT}}"
  PLAN_TEXT="$(parse_final_text /tmp/plan.jsonl)"
fi

EDIT_PROMPT="${PROMPT}"
if [[ -n "${ISSUE_CONTEXT}" ]]; then
  EDIT_PROMPT="${EDIT_PROMPT}

## Linear Issue
${ISSUE_CONTEXT}"
fi
if [[ -n "${PLAN_TEXT}" ]]; then
  EDIT_PROMPT="${EDIT_PROMPT}

## Plan
${PLAN_TEXT}"
fi

log "editing with agent ${EDIT_AGENT}"
run_session edit "${EDIT_AGENT}" "${EDIT_PROMPT}"
EDIT_SESSION_ID="$(parse_session_id /tmp/edit.jsonl)"

verdict="PASS"
attempt=0
if [[ -n "${VERIFY_COMMAND}" ]]; then
  verdict="FAIL"
  while (( attempt <= MAX_FIX_ATTEMPTS )); do
    log "validating: attempt ${attempt} via /${VERIFY_COMMAND}"
    run_session verify "${VERIFY_AGENT}" "BLOCKING: run /${VERIFY_COMMAND}. End your final response with PASS or FAIL.

Task:
${EDIT_PROMPT}"
    verdict="$(parse_verdict /tmp/verify.jsonl)"
    if [[ "${verdict}" == "PASS" ]]; then
      break
    fi

    attempt=$((attempt + 1))
    if (( attempt > MAX_FIX_ATTEMPTS )); then
      break
    fi

    log "fixing validation failures with original edit session"
    run_session fix "${EDIT_AGENT}" "Fix the validation failures below. Preserve the original task intent.

$(tail -50 /tmp/verify.jsonl)" "${EDIT_SESSION_ID}"
  done
fi

if [[ "${verdict}" != "PASS" ]]; then
  write_result failed "validation failed after ${MAX_FIX_ATTEMPTS} fix attempts" "${EDIT_SESSION_ID}"
  exit 1
fi

MR_URL=""
if [[ "${PUSH_CHANGES}" == "true" || "${CREATE_MR}" == "true" ]]; then
  log "committing changes"
  commit_changes
  if [[ "${CHANGES_COMMITTED}" == "true" ]]; then
    push_changes
  else
    log "skipping push because no changes were committed"
  fi
fi

if [[ "${CREATE_MR}" == "true" && "${CHANGES_PUSHED}" == "true" ]]; then
  log "creating merge request"
  MR_URL="$(create_merge_request || true)"
fi

write_result succeeded "run completed" "${EDIT_SESSION_ID}" "${MR_URL}"
