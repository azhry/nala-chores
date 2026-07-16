#!/usr/bin/env bash

commit_changes() {
  cd /workspace/repo
  if git diff --quiet && git diff --cached --quiet; then
    log "no changes to commit"
    return
  fi
  git add -A
  local subject
  subject="[${ISSUE_KEY:-agent}] [ai-assisted] ${REQUEST_ID}"
  git commit -m "${COMMIT_MESSAGE:-${subject}}"
}

push_changes() {
  cd /workspace/repo
  local branch="${OUTPUT_BRANCH:-agent/${REQUEST_ID}}"
  git branch -M "${branch}"
  git push -u origin "${branch}"
}

