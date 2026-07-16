const state = {
  configs: [],
  runs: [],
  selectedRunID: "",
};

const apiStatus = document.querySelector("#apiStatus");
const configForm = document.querySelector("#configForm");
const configMessage = document.querySelector("#configMessage");
const configList = document.querySelector("#configList");
const sessionForm = document.querySelector("#sessionForm");
const sessionMessage = document.querySelector("#sessionMessage");
const runConfigSelect = document.querySelector("#runConfigSelect");
const historyConfigSelect = document.querySelector("#historyConfigSelect");
const runsBody = document.querySelector("#runsBody");
const runDetail = document.querySelector("#runDetail");
const logs = document.querySelector("#logs");
const refreshConfigs = document.querySelector("#refreshConfigs");
const refreshHistory = document.querySelector("#refreshHistory");
const refreshLogs = document.querySelector("#refreshLogs");
const stopRun = document.querySelector("#stopRun");

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
  } catch {
    apiStatus.textContent = "API Offline";
    apiStatus.className = "status-pill bad";
  }
}

async function loadConfigs() {
  const data = await api("/configs");
  state.configs = data.configurations || [];
  renderConfigs();
  renderConfigSelects();
}

async function loadRuns() {
  const configID = historyConfigSelect.value;
  const suffix = configID ? `?limit=50&config_id=${encodeURIComponent(configID)}` : "?limit=50";
  const data = await api(`/runs${suffix}`);
  state.runs = data.runs || [];
  renderRuns();
  if (state.selectedRunID) {
    const selected = state.runs.find((run) => run.request_id === state.selectedRunID);
    if (selected) {
      renderDetail(selected);
    }
  }
}

function renderConfigs() {
  if (!state.configs.length) {
    configList.innerHTML = '<div class="empty">No configurations yet.</div>';
    return;
  }
  configList.innerHTML = state.configs.map((cfg) => `
    <article class="config-card" data-config="${escapeHTML(cfg.id)}">
      <div>
        <h3>${escapeHTML(cfg.name)}</h3>
        <p>${escapeHTML(cfg.repo_url)}</p>
      </div>
      <div class="config-meta">
        <span>${escapeHTML(cfg.source_branch)}</span>
        <span>${escapeHTML(cfg.sandbox_size)}</span>
        <span>${cfg.has_github_token ? "GitHub" : "No GitHub key"}</span>
        <span>${cfg.has_linear_api_key ? "Linear" : "No Linear key"}</span>
        <span>${cfg.has_opencode_api_key ? "OpenCode" : "No OpenCode key"}</span>
      </div>
    </article>
  `).join("");
}

function renderConfigSelects() {
  const options = state.configs.map((cfg) => `<option value="${escapeHTML(cfg.id)}">${escapeHTML(cfg.name)}</option>`).join("");
  const allOption = '<option value="">All configurations</option>';
  runConfigSelect.innerHTML = options || '<option value="">Save a configuration first</option>';
  historyConfigSelect.innerHTML = allOption + options;
}

function renderRuns() {
  if (!state.runs.length) {
    runsBody.innerHTML = '<tr><td colspan="5" class="empty">No sessions yet.</td></tr>';
    return;
  }
  runsBody.innerHTML = state.runs.map((run) => `
    <tr data-id="${escapeHTML(run.request_id)}">
      <td><strong>${escapeHTML(run.request_id)}</strong></td>
      <td><span class="phase ${phaseClass(run.phase)}">${escapeHTML(run.phase)}</span></td>
      <td>${escapeHTML(run.config_name || run.config_id || "")}</td>
      <td>${formatTime(run.updated_at)}</td>
      <td><button class="secondary" data-view="${escapeHTML(run.request_id)}" type="button">Open</button></td>
    </tr>
  `).join("");
}

function renderDetail(run) {
  state.selectedRunID = run.request_id;
  stopRun.disabled = isTerminal(run.phase);
  refreshLogs.disabled = false;
  runDetail.innerHTML = `
    ${detail("Request ID", run.request_id)}
    ${detail("Phase", run.phase)}
    ${detail("Configuration", run.config_name || run.config_id)}
    ${detail("Repository", run.repo_url, true)}
    ${detail("Branch", run.source_branch)}
    ${detail("Linear issue", run.linear_issue_key || "None")}
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
  if (!state.selectedRunID) {
    return;
  }
  const data = await api(`/runs/${encodeURIComponent(state.selectedRunID)}/logs?tail=400`);
  logs.textContent = data.logs || "No logs yet.";
}

configForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(configForm).entries());
  data.create_mr = configForm.create_mr.checked;
  data.push_changes = configForm.push_changes.checked;
  data.clear_opencode_api_key = configForm.clear_opencode_api_key.checked;
  data.source_branch = data.source_branch || "main";
  data.work_directory = data.work_directory || ".";
  data.harness_name = data.harness_name || "default";
  data.sandbox_size = data.sandbox_size || "large";

  configMessage.textContent = "Saving configuration...";
  try {
    const cfg = await api("/configs", { method: "POST", body: JSON.stringify(data) });
    configMessage.textContent = `Saved ${cfg.name}`;
    configForm.reset();
    configForm.source_branch.value = "main";
    configForm.work_directory.value = ".";
    configForm.harness_name.value = "default";
    configForm.sandbox_size.value = "large";
    await loadConfigs();
  } catch (error) {
    configMessage.textContent = error.message;
  }
});

sessionForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const data = Object.fromEntries(new FormData(sessionForm).entries());
  if (!data.config_id) {
    sessionMessage.textContent = "Save and select a configuration first.";
    return;
  }
  sessionMessage.textContent = "Starting session...";
  try {
    const run = await api("/runs", { method: "POST", body: JSON.stringify(data) });
    sessionMessage.textContent = `Started ${run.request_id}`;
    state.selectedRunID = run.request_id;
    switchView("history");
    historyConfigSelect.value = run.config_id;
    await loadRuns();
  } catch (error) {
    sessionMessage.textContent = error.message;
  }
});

runsBody.addEventListener("click", async (event) => {
  const row = event.target.closest("[data-id]");
  const id = row?.dataset.id || event.target.dataset.view;
  if (!id) {
    return;
  }
  const run = await api(`/runs/${encodeURIComponent(id)}`);
  renderDetail(run);
});

document.querySelectorAll(".menu-item").forEach((button) => {
  button.addEventListener("click", () => switchView(button.dataset.view));
});

refreshConfigs.addEventListener("click", () => loadConfigs().catch(showConfigError));
refreshHistory.addEventListener("click", () => loadRuns().catch(showSessionError));
historyConfigSelect.addEventListener("change", () => loadRuns().catch(showSessionError));
refreshLogs.addEventListener("click", () => loadLogs().catch((error) => { logs.textContent = error.message; }));

stopRun.addEventListener("click", async () => {
  if (!state.selectedRunID) {
    return;
  }
  const run = await api(`/runs/${encodeURIComponent(state.selectedRunID)}/stop`, { method: "POST", body: "{}" });
  renderDetail(run);
  await loadRuns();
});

function switchView(view) {
  document.querySelectorAll(".menu-item").forEach((button) => {
    button.classList.toggle("active", button.dataset.view === view);
  });
  document.querySelectorAll(".view").forEach((section) => {
    section.classList.toggle("active", section.id === `${view}View`);
  });
  if (view === "history") {
    loadRuns().catch(showSessionError);
  }
}

function showConfigError(error) {
  configMessage.textContent = error.message;
}

function showSessionError(error) {
  sessionMessage.textContent = error.message;
}

function phaseClass(phase) {
  return String(phase || "").toLowerCase();
}

function isTerminal(phase) {
  return ["SUCCEEDED", "FAILED", "CANCELLED"].includes(phase);
}

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
loadConfigs().then(loadRuns).catch(() => {});
setInterval(() => {
  if (document.querySelector("#historyView").classList.contains("active")) {
    loadRuns().catch(() => {});
  }
}, 5000);
