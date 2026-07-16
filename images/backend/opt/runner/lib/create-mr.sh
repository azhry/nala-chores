#!/usr/bin/env bash

create_merge_request() {
  local branch="${OUTPUT_BRANCH:-agent/${REQUEST_ID}}"
  if [[ -n "${GITHUB_TOKEN:-${GIT_TOKEN:-}}" && "${REPO_URL}" == *github.com* ]]; then
    if command -v gh >/dev/null 2>&1; then
      GH_TOKEN="${GITHUB_TOKEN:-${GIT_TOKEN}}" gh pr create \
        --draft \
        --title "${MR_TITLE:-Agent run ${REQUEST_ID}}" \
        --body "Created by Nala Chores for ${REQUEST_ID}." 2>/dev/null
      return
    fi
  fi

  log "MR creation is not configured for this git host; branch ${branch} was pushed"
  printf ''
}
