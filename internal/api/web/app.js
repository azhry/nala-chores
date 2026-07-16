const state = {
  selectedID: "",
  runs: [],
};

const apiStatus = document.querySelector("#apiStatus");
const form = document.querySelector("#runForm");
const formMessage = document.querySelector("#formMessage");
const runsBody = document.querySelector("#runsBody");
const runDetail = document.querySelector("#runDetail");
const logs = document.querySelector("#logs");
const refreshRuns = document.querySelector("#refreshRuns");
const refreshLogs = document.querySelector("#refreshLogs");
const stopRun = document.querySelector("#stopRun");

function requestID(repo, branch) {
  const repoName = repo.split("/").pop().replace(/\.git$/, "") || "repo";
  const suffix = Math.floor(Date.now() / 1000);
  return `ui-${repoName}-${branch}-${suffix}`.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
}

function phaseClass(phase) {
  return String(phase || "").toLowerCase();
}

function isTerminal(phase) {
  return ["SUCCEEDED", "FAILED", "CANCELLED"].includes(phase);
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || response.statusText);
  }
  return data;
}

async function checkHealth() {
  try {
    await api("/healthz");
    apiStatus.textContent = "API Online";
    apiStatus.className = "status-pill ok";
  } catch (error) {
    apiStatus.textContent = "API Offline";
    apiStatus.className = "status-pill bad";
  }
}

async function loadRuns() {
  const data = await api("/runs?limit=50");
  state.runs = data.runs || [];
  renderRuns();
  if (state.selectedID) {
    const selected = state.runs.find((run) => run.request_id === state.selectedID);
    if (selected) {
      renderDetail(selected);
    }
  }
}

function renderRuns() {
  if (!state.runs.length) {
    runsBody.innerHTML = '<tr><td colspan="5" class="empty">No runs yet.</td></tr>';
    return;
  }
  runsBody.innerHTML = state.runs.map((run) => `
    <tr data-id="${escapeHTML(run.request_id)}">
      <td><strong>${escapeHTML(run.request_id)}</strong></td>
      <td><span class="phase ${phaseClass(run.phase)}">${escapeHTML(run.phase)}</span></td>
      <td>${escapeHTML(run.source_branch || "")}</td>
      <td>${formatTime(run.updated_at)}</td>
      <td><button class="secondary" data-view="${escapeHTML(run.request_id)}" type="button">Open</button></td>
    </tr>
  `).join("");
}

function renderDetail(run) {
  state.selectedID = run.request_id;
  stopRun.disabled = isTerminal(run.phase);
  refreshLogs.disabled = false;
  runDetail.innerHTML = `
    ${detail("Request ID", run.request_id)}
    ${detail("Phase", run.phase)}
    ${detail("Repository", run.repo_url, true)}
    ${detail("Branch", run.source_branch)}
    ${detail("Work directory", run.work_directory)}
    ${detail("Harness", run.harness_name)}
    ${detail("Sandbox size", run.sandbox_size)}
    ${detail("Job", run.job_name || "Pending")}
    ${detail("Logs command", run.logs_command || "Pending", true)}
    ${detail("Message", run.message || "No message", true)}
    ${detail("Prompt", run.prompt || "", true)}
  `;
  loadLogs().catch((error) => {
    logs.textContent = error.message;
  });
}

function detail(label, value, wide = false) {
  return `<div class="detail-item ${wide ? "wide" : ""}"><strong>${escapeHTML(label)}</strong><span>${escapeHTML(value || "")}</span></div>`;
}

async function loadLogs() {
  if (!state.selectedID) {
    return;
  }
  const data = await api(`/runs/${encodeURIComponent(state.selectedID)}/logs?tail=240`);
  logs.textContent = data.logs || "No logs yet.";
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(form).entries());
  data.create_mr = form.create_mr.checked;
  data.push_changes = form.push_changes.checked;
  data.work_directory = data.work_directory || ".";
  data.harness_name = data.harness_name || "default";
  data.sandbox_size = data.sandbox_size || "large";
  data.request_id = data.request_id || requestID(data.repo_url, data.source_branch);

  formMessage.textContent = "Submitting run...";
  try {
    const run = await api("/runs", {
      method: "POST",
      body: JSON.stringify(data),
    });
    formMessage.textContent = `Submitted ${run.request_id}`;
    state.selectedID = run.request_id;
    await loadRuns();
  } catch (error) {
    formMessage.textContent = error.message;
  }
});

runsBody.addEventListener("click", async (event) => {
  const id = event.target.closest("[data-id]")?.dataset.id || event.target.dataset.view;
  if (!id) {
    return;
  }
  const run = await api(`/runs/${encodeURIComponent(id)}`);
  renderDetail(run);
});

refreshRuns.addEventListener("click", () => {
  loadRuns().catch((error) => {
    formMessage.textContent = error.message;
  });
});

refreshLogs.addEventListener("click", () => {
  loadLogs().catch((error) => {
    logs.textContent = error.message;
  });
});

stopRun.addEventListener("click", async () => {
  if (!state.selectedID) {
    return;
  }
  const run = await api(`/runs/${encodeURIComponent(state.selectedID)}/stop`, {
    method: "POST",
    body: "{}",
  });
  renderDetail(run);
  await loadRuns();
});

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  })[char]);
}

function formatTime(value) {
  if (!value) {
    return "";
  }
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    month: "short",
    day: "numeric",
  }).format(new Date(value));
}

checkHealth();
loadRuns().catch(() => {});
setInterval(loadRuns, 5000);

