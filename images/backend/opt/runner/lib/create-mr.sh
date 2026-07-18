#!/usr/bin/env bash

create_merge_request() {
  local branch="${OUTPUT_BRANCH:-agent/${REQUEST_ID}}"
  local token="${GITHUB_TOKEN:-${GIT_TOKEN:-}}"
  if [[ -n "${GITHUB_TOKEN:-${GIT_TOKEN:-}}" && "${REPO_URL}" == *github.com* ]]; then
    if command -v gh >/dev/null 2>&1; then
      GH_TOKEN="${token}" gh pr create \
        --draft \
        --title "${MR_TITLE:-Agent run ${REQUEST_ID}}" \
        --body "Created by Nala Chores for ${REQUEST_ID}." 2>/dev/null
      return
    fi

    create_github_pull_request "${token}" "${branch}"
    return
  fi

  log "MR creation is not configured for this git host; branch ${branch} was pushed"
  printf ''
}

create_github_pull_request() {
  local token="$1"
  local branch="$2"
  local repo_path
  repo_path="$(github_repo_path)"
  if [[ -z "${repo_path}" ]]; then
    log "could not infer GitHub repo path for PR creation"
    printf ''
    return
  fi

  local title="${MR_TITLE:-Agent run ${REQUEST_ID}}"
  local body="${MR_BODY:-Created by Nala Chores for ${REQUEST_ID}.}"
  local payload
  payload="$(jq -n \
    --arg title "${title}" \
    --arg head "${branch}" \
    --arg base "${SOURCE_BRANCH}" \
    --arg body "${body}" \
    '{title: $title, head: $head, base: $base, body: $body, draft: true}')"

  local response status url message
  response="$(curl -sS -w '\n%{http_code}' \
    -X POST "https://api.github.com/repos/${repo_path}/pulls" \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer ${token}" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    --data "${payload}")"
  status="$(printf '%s' "${response}" | tail -n 1)"
  response="$(printf '%s' "${response}" | sed '$d')"
  url="$(printf '%s' "${response}" | jq -r '.html_url // empty' 2>/dev/null || true)"
  if [[ "${status}" =~ ^2 && -n "${url}" ]]; then
    printf '%s' "${url}"
    return
  fi

  message="$(printf '%s' "${response}" | jq -r '.message // empty' 2>/dev/null || true)"
  log "GitHub PR creation failed with HTTP ${status}${message:+: ${message}}"
  printf ''
}

github_repo_path() {
  local url="${REPO_URL}"
  url="${url#https://github.com/}"
  url="${url#http://github.com/}"
  url="${url#git@github.com:}"
  url="${url%.git}"
  if [[ "${url}" == */* && "${url}" != http* && "${url}" != git@* ]]; then
    printf '%s' "${url}"
  fi
}
