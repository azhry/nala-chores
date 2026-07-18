#!/usr/bin/env bash

CHANGES_COMMITTED="false"
CHANGES_PUSHED="false"

commit_changes() {
  cd /workspace/repo
  if git diff --quiet && git diff --cached --quiet; then
    local ahead
    ahead="$(git rev-list --count "origin/${SOURCE_BRANCH}..HEAD" 2>/dev/null || printf '0')"
    if [[ "${ahead}" != "0" ]]; then
      log "using ${ahead} existing local commit(s)"
      CHANGES_COMMITTED="true"
      return
    fi
    log "no changes to commit"
    return
  fi
  git add -A
  local subject
  subject="[${ISSUE_KEY:-agent}] [ai-assisted] ${REQUEST_ID}"
  git commit -m "${COMMIT_MESSAGE:-${subject}}"
  CHANGES_COMMITTED="true"
}

push_changes() {
  cd /workspace/repo
  local branch="${OUTPUT_BRANCH:-agent/${REQUEST_ID}}"
  git branch -M "${branch}"
  git push -u origin "${branch}"
  CHANGES_PUSHED="true"
}
