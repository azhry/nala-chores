#!/usr/bin/env bash

create_merge_request() {
  local branch="${OUTPUT_BRANCH:-agent/${REQUEST_ID}}"
  local token="${GITHUB_TOKEN:-${GIT_TOKEN:-}}"
  if [[ -n "${GITHUB_TOKEN:-${GIT_TOKEN:-}}" && "${REPO_URL}" == *github.com* ]]; then
    local title body draft_args
    title="$(pull_request_title)"
    body="$(pull_request_body)"
    if command -v gh >/dev/null 2>&1; then
      draft_args=()
      if [[ "$(bool "${MR_DRAFT:-false}")" == "true" ]]; then
        draft_args=(--draft)
      fi
      GH_TOKEN="${token}" gh pr create \
        "${draft_args[@]}" \
        --title "${title}" \
        --body "${body}" 2>/dev/null
      return
    fi

    create_github_pull_request "${token}" "${branch}" "${title}" "${body}"
    return
  fi

  log "MR creation is not configured for this git host; branch ${branch} was pushed"
  printf ''
}

create_github_pull_request() {
  local token="$1"
  local branch="$2"
  local title="$3"
  local body="$4"
  local repo_path
  repo_path="$(github_repo_path)"
  if [[ -z "${repo_path}" ]]; then
    log "could not infer GitHub repo path for PR creation"
    printf ''
    return
  fi

  local payload draft
  draft="$(bool "${MR_DRAFT:-false}")"
  payload="$(jq -n \
    --arg title "${title}" \
    --arg head "${branch}" \
    --arg base "${SOURCE_BRANCH}" \
    --arg body "${body}" \
    --argjson draft "${draft}" \
    '{title: $title, head: $head, base: $base, body: $body, draft: $draft}')"

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

pull_request_title() {
  local artifact
  for artifact in \
    "${MR_TITLE_FILE:-}" \
    "/tmp/pr_title.txt" \
    "/workspace/repo/.nala-chores/pr-title.txt"; do
    if [[ -n "${artifact}" && -s "${artifact}" ]]; then
      head -n 1 "${artifact}" | sed 's/^[[:space:]]*//; s/[[:space:]]*$//'
      return
    fi
  done
  if [[ -n "${MR_TITLE:-}" ]]; then
    printf '%s' "${MR_TITLE}"
    return
  fi
  git -C /workspace/repo log -1 --pretty=%s 2>/dev/null || printf 'Agent run %s' "${REQUEST_ID}"
}

pull_request_body() {
  local artifact
  for artifact in \
    "${MR_BODY_FILE:-}" \
    "/tmp/pr_body.txt" \
    "/tmp/pr_body.md" \
    "/workspace/repo/.nala-chores/pr-body.md" \
    "/workspace/repo/.nala-chores/pr-body.txt"; do
    if [[ -n "${artifact}" && -s "${artifact}" ]]; then
      cat "${artifact}"
      return
    fi
  done
  if [[ -n "${MR_BODY:-}" ]]; then
    printf '%s' "${MR_BODY}"
    return
  fi
  fallback_pull_request_body
}

fallback_pull_request_body() {
  local commit subject stats
  commit="$(git -C /workspace/repo rev-parse --short HEAD 2>/dev/null || true)"
  subject="$(git -C /workspace/repo log -1 --pretty=%s 2>/dev/null || true)"
  stats="$(git -C /workspace/repo show --stat --oneline --no-renames --format='' HEAD 2>/dev/null || true)"
  cat <<EOF
## Summary
- ${subject:-Agent changes for ${REQUEST_ID}}

## Verification
- Runner completed successfully.

## Delivery
- Request ID: ${REQUEST_ID}
- Commit: ${commit:-unknown}

## Changed Files
\`\`\`
${stats:-No diff stat available.}
\`\`\`
EOF
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
