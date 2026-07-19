const { createElement: h, useCallback, useEffect, useMemo, useRef, useState } = React;

const DEFAULT_CONFIG = {
  id: "",
  name: "",
  repo_url: "",
  harness_repo_url: "",
  source_branch: "main",
  work_directory: ".",
  harness_name: "default",
  config_path: "",
  sandbox_size: "large",
  agent_provider: "opencode",
  agent_model: "opencode/big-pickle",
  github_token: "",
  linear_api_key: "",
  opencode_api_key: "",
  kilo_api_key: "",
  clear_github_token: false,
  clear_linear_api_key: false,
  clear_opencode_api_key: false,
  clear_kilo_api_key: false,
  push_changes: true,
  create_mr: true,
};

const NAV = [
  { id: "configs", label: "Configurations", icon: "settings" },
  { id: "run", label: "Run", icon: "play_arrow" },
  { id: "history", label: "History", icon: "history" },
];

function App() {
  const [view, setView] = useState("configs");
  const [apiOnline, setAPIOnline] = useState(false);
  const [configs, setConfigs] = useState([]);
  const [runs, setRuns] = useState([]);
  const [selectedConfigID, setSelectedConfigID] = useState("");
  const [selectedRunID, setSelectedRunID] = useState("");
  const [selectedRun, setSelectedRun] = useState(null);
  const [logs, setLogs] = useState("");
  const [messages, setMessages] = useState({ config: "", session: "", history: "" });
  const [historyConfigID, setHistoryConfigID] = useState("");

  const loadConfigs = useCallback(async () => {
    const data = await api("/configs");
    const next = data.configurations || [];
    setConfigs(next);
    setSelectedConfigID((current) => current || next[0]?.id || "");
  }, []);

  const loadRuns = useCallback(async (configID = historyConfigID) => {
    const suffix = configID ? `?limit=50&config_id=${encodeURIComponent(configID)}` : "?limit=50";
    const data = await api(`/runs${suffix}`);
    setRuns(data.runs || []);
  }, [historyConfigID]);

  const loadRun = useCallback(async (requestID) => {
    if (!requestID) return;
    const run = await api(`/runs/${encodeURIComponent(requestID)}`);
    setSelectedRun(run);
    setSelectedRunID(run.request_id);
    const logData = await api(`/runs/${encodeURIComponent(requestID)}/logs?tail=400`);
    setLogs(logData.logs || "");
  }, []);

  const checkHealth = useCallback(async () => {
    try {
      await api("/healthz");
      setAPIOnline(true);
    } catch {
      setAPIOnline(false);
    }
  }, []);

  useEffect(() => {
    checkHealth();
    loadConfigs().then(() => loadRuns("")).catch(() => {});
  }, [checkHealth, loadConfigs, loadRuns]);

  useEffect(() => {
    if (view !== "history") return undefined;
    loadRuns().catch(() => {});
    const id = window.setInterval(() => {
      loadRuns().catch(() => {});
      if (selectedRunID) {
        loadRun(selectedRunID).catch(() => {});
      }
    }, 5000);
    return () => window.clearInterval(id);
  }, [loadRun, loadRuns, selectedRunID, view]);

  async function saveConfig(input) {
    setMessages((current) => ({ ...current, config: "Saving configuration..." }));
    try {
      const cfg = await api("/configs", { method: "POST", body: JSON.stringify(input) });
      setMessages((current) => ({ ...current, config: `Saved ${cfg.name}` }));
      await loadConfigs();
      setSelectedConfigID(cfg.id);
    } catch (error) {
      setMessages((current) => ({ ...current, config: error.message }));
    }
  }

  async function startRun(input) {
    if (!input.config_id) {
      setMessages((current) => ({ ...current, session: "Save and select a configuration first." }));
      return;
    }
    setMessages((current) => ({ ...current, session: "Starting session..." }));
    try {
      const run = await api("/runs", { method: "POST", body: JSON.stringify(input) });
      setMessages((current) => ({ ...current, session: `Started ${run.request_id}` }));
      setHistoryConfigID(run.config_id);
      setSelectedRunID(run.request_id);
      setView("history");
      await loadRuns(run.config_id);
      await loadRun(run.request_id);
    } catch (error) {
      setMessages((current) => ({ ...current, session: error.message }));
    }
  }

  async function stopSelectedRun() {
    if (!selectedRunID) return;
    const run = await api(`/runs/${encodeURIComponent(selectedRunID)}/stop`, { method: "POST", body: "{}" });
    setSelectedRun(run);
    await loadRuns();
    await loadRun(run.request_id);
  }

  const activeConfig = configs.find((cfg) => cfg.id === selectedConfigID) || null;

  return h("div", { className: "app-shell" },
    h(Sidebar, { view, setView, apiOnline }),
    h("div", { className: "workspace" },
      h(Topbar, { view, apiOnline, configs, runs }),
      h("main", { className: "content" },
        view === "configs" && h(ConfigurationsView, {
          configs,
          selectedConfigID,
          setSelectedConfigID,
          activeConfig,
          onSave: saveConfig,
          onRefresh: loadConfigs,
          message: messages.config,
        }),
        view === "run" && h(RunView, {
          configs,
          selectedConfigID,
          setSelectedConfigID,
          onRun: startRun,
          message: messages.session,
        }),
        view === "history" && h(HistoryView, {
          configs,
          runs,
          historyConfigID,
          setHistoryConfigID,
          selectedRun,
          selectedRunID,
          logs,
          onRefreshRuns: () => loadRuns(),
          onSelectRun: loadRun,
          onRefreshLogs: () => loadRun(selectedRunID),
          onStop: stopSelectedRun,
        })
      )
    ),
    h(MobileNav, { view, setView })
  );
}

function Sidebar({ view, setView, apiOnline }) {
  return h("aside", { className: "sidebar" },
    h("div", { className: "brand" },
      h(Icon, { name: "terminal" }),
      h("div", null,
        h("strong", null, "Nala Chores"),
        h("span", null, "Agent runner")
      )
    ),
    h("nav", { className: "nav-list", "aria-label": "Primary" },
      NAV.map((item) => h("button", {
        key: item.id,
        className: `nav-item ${view === item.id ? "active" : ""}`,
        type: "button",
        onClick: () => setView(item.id),
      }, h(Icon, { name: item.icon }), h("span", null, item.label)))
    ),
    h("div", { className: "sidebar-status" },
      h(Badge, { tone: apiOnline ? "success" : "danger" }, apiOnline ? "API ONLINE" : "API OFFLINE"),
      h("span", null, "Minikube control plane")
    )
  );
}

function Topbar({ view, apiOnline, configs, runs }) {
  const title = view === "configs" ? "Configurations" : view === "run" ? "Run Session" : "Session History";
  const subtitle = view === "configs"
    ? "Manage execution environments, credentials, harnesses, and agent settings."
    : view === "run"
      ? "Launch a scoped agent job from a saved project profile."
      : "Audit run state, pull requests, and transcript output.";

  return h("header", { className: "topbar" },
    h("div", null,
      h("p", { className: "eyebrow" }, "Nala Chores"),
      h("h1", null, title),
      h("p", { className: "topbar-subtitle" }, subtitle)
    ),
    h("div", { className: "topbar-metrics" },
      h(Metric, { label: "Configs", value: configs.length }),
      h(Metric, { label: "Runs", value: runs.length }),
      h(Badge, { tone: apiOnline ? "success" : "danger" }, apiOnline ? "ONLINE" : "OFFLINE")
    )
  );
}

function ConfigurationsView({ configs, selectedConfigID, setSelectedConfigID, activeConfig, onSave, onRefresh, message }) {
  const [draft, setDraft] = useState(DEFAULT_CONFIG);

  useEffect(() => {
    if (!activeConfig) {
      setDraft(DEFAULT_CONFIG);
      return;
    }
    setDraft({
      ...DEFAULT_CONFIG,
      id: activeConfig.id || "",
      name: activeConfig.name || "",
      repo_url: activeConfig.repo_url || "",
      harness_repo_url: activeConfig.harness_repo_url || "",
      source_branch: activeConfig.source_branch || "main",
      work_directory: activeConfig.work_directory || ".",
      harness_name: activeConfig.harness_name || "default",
      config_path: activeConfig.config_path || "",
      sandbox_size: activeConfig.sandbox_size || "large",
      agent_provider: activeConfig.agent_provider || "opencode",
      agent_model: activeConfig.agent_model || "opencode/big-pickle",
      push_changes: Boolean(activeConfig.push_changes),
      create_mr: Boolean(activeConfig.create_mr),
    });
  }, [activeConfig]);

  function update(name, value) {
    setDraft((current) => {
      const next = { ...current, [name]: value };
      if (name === "agent_provider" && !current.agent_model) {
        next.agent_model = value === "kilocode" ? "kilo/kilo-auto/free" : "opencode/big-pickle";
      }
      if (name === "create_mr" && value) {
        next.push_changes = true;
      }
      return next;
    });
  }

  function submit(event) {
    event.preventDefault();
    onSave({
      ...draft,
      source_branch: draft.source_branch || "main",
      work_directory: draft.work_directory || ".",
      harness_name: draft.harness_name || "default",
      sandbox_size: draft.sandbox_size || "large",
      agent_provider: draft.agent_provider || "opencode",
      agent_model: draft.agent_model || (draft.agent_provider === "kilocode" ? "kilo/kilo-auto/free" : "opencode/big-pickle"),
    });
  }

  return h("section", { className: "config-grid" },
    h("form", { className: "ops-card config-form", onSubmit: submit },
      h(CardHeader, {
        icon: "settings",
        title: activeConfig ? `Editing ${activeConfig.name}` : "New configuration",
        subtitle: "Credentials are stored once per project profile and never displayed back.",
        action: h("button", { className: "ghost-button", type: "button", onClick: onRefresh }, h(Icon, { name: "refresh" }), "Refresh"),
      }),
      h("div", { className: "form-section" },
        h("h2", null, "Repository & Harness"),
        h(Field, { label: "Name", name: "name", value: draft.name, onChange: update, required: true, placeholder: "Nala Grow" }),
        h(Field, { label: "Repository URL", name: "repo_url", value: draft.repo_url, onChange: update, required: true, mono: true, placeholder: "https://github.com/azhry/nala-grow.git" }),
        h(Field, { label: "Harness repository URL", name: "harness_repo_url", value: draft.harness_repo_url, onChange: update, mono: true, placeholder: "https://github.com/azhry/my-harnesses.git" }),
        h("div", { className: "field-row" },
          h(Field, { label: "Branch", name: "source_branch", value: draft.source_branch, onChange: update, mono: true }),
          h(Field, { label: "Work directory", name: "work_directory", value: draft.work_directory, onChange: update, mono: true })
        ),
        h("div", { className: "field-row" },
          h(Field, { label: "Harness", name: "harness_name", value: draft.harness_name, onChange: update, mono: true }),
          h(Field, { label: "Config path", name: "config_path", value: draft.config_path, onChange: update, mono: true, placeholder: ".opencode-runner.yml" })
        )
      ),
      h("div", { className: "form-section" },
        h("h2", null, "Agent Runtime"),
        h("div", { className: "field-row" },
          h(SelectField, { label: "Agent provider", name: "agent_provider", value: draft.agent_provider, onChange: update, options: [["opencode", "OpenCode"], ["kilocode", "KiloCode"]] }),
          h(Field, { label: "Agent model", name: "agent_model", value: draft.agent_model, onChange: update, mono: true })
        ),
        h("div", { className: "field-row" },
          h(SelectField, { label: "Sandbox size", name: "sandbox_size", value: draft.sandbox_size, onChange: update, options: [["small", "Small"], ["large", "Large"], ["xlarge", "XLarge"], ["2xlarge", "2XLarge"]] }),
          h("div", { className: "switch-stack" },
            h(SwitchField, { label: "Push changes", description: "Push committed work to the remote branch.", checked: draft.push_changes, onChange: (value) => update("push_changes", value) }),
            h(SwitchField, { label: "Create PR/MR", description: "Require a pull request URL for success.", checked: draft.create_mr, onChange: (value) => update("create_mr", value) })
          )
        )
      ),
      h("div", { className: "form-section" },
        h("h2", null, "Secrets"),
        h("div", { className: "secret-grid" },
          h(SecretField, { label: "GitHub token", name: "github_token", clearName: "clear_github_token", saved: activeConfig?.has_github_token, value: draft.github_token, clear: draft.clear_github_token, onChange: update }),
          h(SecretField, { label: "Linear API key", name: "linear_api_key", clearName: "clear_linear_api_key", saved: activeConfig?.has_linear_api_key, value: draft.linear_api_key, clear: draft.clear_linear_api_key, onChange: update }),
          h(SecretField, { label: "OpenCode API key", name: "opencode_api_key", clearName: "clear_opencode_api_key", saved: activeConfig?.has_opencode_api_key, value: draft.opencode_api_key, clear: draft.clear_opencode_api_key, onChange: update }),
          h(SecretField, { label: "Kilo API key", name: "kilo_api_key", clearName: "clear_kilo_api_key", saved: activeConfig?.has_kilo_api_key, value: draft.kilo_api_key, clear: draft.clear_kilo_api_key, onChange: update })
        )
      ),
      h("div", { className: "form-actions" },
        h("button", { className: "primary-button", type: "submit" }, h(Icon, { name: "save" }), activeConfig ? "Save changes" : "Save configuration"),
        h("button", { className: "ghost-button", type: "button", onClick: () => { setSelectedConfigID(""); setDraft(DEFAULT_CONFIG); } }, "New profile")
      ),
      message && h("p", { className: "form-message" }, message)
    ),
    h("aside", { className: "ops-card saved-panel" },
      h(CardHeader, { icon: "bookmarks", title: "Saved Profiles", subtitle: `${configs.length} configured environment${configs.length === 1 ? "" : "s"}` }),
      h("div", { className: "profile-list" },
        configs.length ? configs.map((cfg) => h(ConfigCard, {
          key: cfg.id,
          cfg,
          active: cfg.id === selectedConfigID,
          onClick: () => setSelectedConfigID(cfg.id),
        })) : h(EmptyState, { title: "No configurations", body: "Save a profile to start running agent sessions." })
      )
    )
  );
}

function RunView({ configs, selectedConfigID, setSelectedConfigID, onRun, message }) {
  const [prompt, setPrompt] = useState("");
  const [linearIssueKey, setLinearIssueKey] = useState("");
  const selected = configs.find((cfg) => cfg.id === selectedConfigID) || configs[0] || null;

  useEffect(() => {
    if (!selectedConfigID && configs[0]?.id) setSelectedConfigID(configs[0].id);
  }, [configs, selectedConfigID, setSelectedConfigID]);

  function submit(event) {
    event.preventDefault();
    onRun({
      config_id: selected?.id || "",
      prompt,
      linear_issue_key: linearIssueKey,
      issue_key: linearIssueKey,
    });
  }

  return h("section", { className: "run-grid" },
    h("form", { className: "ops-card run-form", onSubmit: submit },
      h(CardHeader, { icon: "play_arrow", title: "Run Session", subtitle: "One profile, one prompt, one task boundary." }),
      h(SelectField, {
        label: "Configuration",
        name: "config_id",
        value: selected?.id || "",
        onChange: (_, value) => setSelectedConfigID(value),
        options: configs.map((cfg) => [cfg.id, cfg.name]),
      }),
      h(Field, { label: "Linear issue key", name: "linear_issue_key", value: linearIssueKey, onChange: (_, value) => setLinearIssueKey(value), mono: true, placeholder: "AZH-282" }),
      h("label", { className: "field" },
        h("span", null, "Prompt"),
        h("textarea", { value: prompt, onChange: (event) => setPrompt(event.target.value), required: true, placeholder: "Implement the Linear task exactly. Do not perform unrelated cleanup or docs-only work." })
      ),
      h("button", { className: "primary-button", type: "submit", disabled: !configs.length }, h(Icon, { name: "rocket_launch" }), "Run session"),
      message && h("p", { className: "form-message" }, message)
    ),
    h("aside", { className: "ops-card run-context" },
      h(CardHeader, { icon: "dns", title: "Selected Profile", subtitle: selected ? selected.repo_url : "No profile selected" }),
      selected ? h("div", { className: "context-list" },
        h(ContextRow, { label: "Provider", value: selected.agent_provider }),
        h(ContextRow, { label: "Model", value: selected.agent_model, mono: true }),
        h(ContextRow, { label: "Harness", value: selected.harness_name, mono: true }),
        h(ContextRow, { label: "Harness repo", value: selected.harness_repo_url || "None", mono: true }),
        h(ContextRow, { label: "Branch", value: selected.source_branch, mono: true }),
        h("div", { className: "badge-row" },
          h(Badge, { tone: selected.has_github_token ? "success" : "muted" }, selected.has_github_token ? "GITHUB" : "NO GITHUB"),
          h(Badge, { tone: selected.has_linear_api_key ? "success" : "muted" }, selected.has_linear_api_key ? "LINEAR" : "NO LINEAR"),
          h(Badge, { tone: selected.create_mr ? "info" : "muted" }, selected.create_mr ? "PR REQUIRED" : "NO PR")
        )
      ) : h(EmptyState, { title: "No profile", body: "Create a configuration before starting a run." })
    )
  );
}

function HistoryView({ configs, runs, historyConfigID, setHistoryConfigID, selectedRun, selectedRunID, logs, onRefreshRuns, onSelectRun, onRefreshLogs, onStop }) {
  const entries = useMemo(() => parseLogEntries(logs), [logs]);
  const logRef = useRef(null);
  const live = Boolean(selectedRun && !isTerminal(selectedRun.phase));
  const logState = selectedRun ? (live ? "Streaming" : "Finished") : "Idle";

  useEffect(() => {
    if (selectedRunID || !runs.length) return;
    onSelectRun(runs[0].request_id);
  }, [onSelectRun, runs, selectedRunID]);

  useEffect(() => {
    const node = logRef.current;
    if (!node) return;
    node.scrollTop = node.scrollHeight;
  }, [entries.length, logs, selectedRunID]);

  return h("section", { className: "history-grid" },
    h("div", { className: "ops-card runs-panel" },
      h(CardHeader, {
        icon: "history",
        title: "Runs",
        subtitle: `${runs.length} recent session${runs.length === 1 ? "" : "s"}`,
        action: h("button", { className: "ghost-button", type: "button", onClick: onRefreshRuns }, h(Icon, { name: "refresh" }), "Refresh"),
      }),
      h("div", { className: "toolbar-row" },
        h(SelectField, {
          label: "Configuration filter",
          name: "history_config_id",
          value: historyConfigID,
          onChange: (_, value) => setHistoryConfigID(value),
          options: [["", "All configurations"], ...configs.map((cfg) => [cfg.id, cfg.name])],
        })
      ),
      h("div", { className: "run-list" },
        runs.length ? runs.map((run) => h(RunRow, {
          key: run.request_id,
          run,
          active: run.request_id === selectedRunID,
          onClick: () => onSelectRun(run.request_id),
        })) : h(EmptyState, { title: "No sessions", body: "Run history will appear here after a session starts." })
      )
    ),
    h("div", { className: "history-inspector" },
      h("div", { className: "ops-card transcript-panel" },
        h(CardHeader, {
          icon: "terminal",
          title: "Session Transcript",
          subtitle: selectedRun ? `${selectedRun.request_id} · ${logState}${selectedRun.updated_at ? ` · updated ${formatTime(selectedRun.updated_at)}` : ""}` : "Select a run to read the transcript.",
          action: h("div", { className: "header-actions" },
            h(StatusIndicator, { live, phase: selectedRun?.phase }),
            h("button", { className: "ghost-button", type: "button", disabled: !selectedRunID, onClick: onRefreshLogs }, h(Icon, { name: "sync" }), "Refresh"),
            h("button", { className: "danger-button", type: "button", disabled: !selectedRun || isTerminal(selectedRun.phase), onClick: onStop }, "Stop")
          ),
        }),
        h("div", { className: "chat-log", ref: logRef },
          entries.length ? entries.map((entry, index) => h(ChatEntry, { key: `${entry.title}-${index}`, entry })) : h(EmptyState, { title: "No logs", body: selectedRun ? "Logs are not available yet." : "Select a run to read the transcript." })
        )
      ),
      h("div", { className: "ops-card detail-panel" },
        h(CardHeader, {
          icon: "fact_check",
          title: "Session Detail",
          subtitle: selectedRun ? "Repository, job, PR, and prompt metadata." : "No session selected.",
        }),
        selectedRun ? h(RunDetail, { run: selectedRun }) : h(EmptyState, { title: "No session selected", body: "Open a run to view repository, job, PR, and prompt metadata." })
      )
    )
  );
}

function ConfigCard({ cfg, active, onClick }) {
  return h("button", { className: `profile-card ${active ? "active" : ""}`, type: "button", onClick },
    h("div", null,
      h("strong", null, cfg.name),
      h("span", { className: "mono truncate" }, cfg.repo_url),
      cfg.harness_repo_url && h("span", { className: "mono truncate" }, cfg.harness_repo_url)
    ),
    h("div", { className: "badge-row" },
      h(Badge, { tone: "muted" }, cfg.source_branch || "MAIN"),
      h(Badge, { tone: "muted" }, cfg.sandbox_size || "LARGE"),
      h(Badge, { tone: "info" }, cfg.agent_provider || "OPENCODE"),
      h(Badge, { tone: cfg.has_github_token ? "success" : "muted" }, cfg.has_github_token ? "GITHUB" : "NO GITHUB"),
      h(Badge, { tone: cfg.has_linear_api_key ? "success" : "muted" }, cfg.has_linear_api_key ? "LINEAR" : "NO LINEAR")
    )
  );
}

function RunRow({ run, active, onClick }) {
  return h("button", { className: `run-row ${active ? "active" : ""}`, type: "button", onClick },
    h("div", null,
      h("strong", { className: "mono" }, run.request_id),
      h("span", null, run.config_name || run.config_id || "Unknown configuration")
    ),
    h("div", null,
      h(Badge, { tone: phaseTone(run.phase) }, run.phase || "UNKNOWN"),
      h("span", { className: "run-time" }, formatTime(run.updated_at))
    )
  );
}

function RunDetail({ run }) {
  return h("div", { className: "detail-grid" },
    h(ContextRow, { label: "Phase", value: run.phase }),
    h(ContextRow, { label: "Configuration", value: run.config_name || run.config_id }),
    h(ContextRow, { label: "Repository", value: run.repo_url, mono: true, wide: true }),
    h(ContextRow, { label: "Harness repository", value: run.harness_repo_url || "None", mono: true, wide: true }),
    h(ContextRow, { label: "Provider", value: run.agent_provider }),
    h(ContextRow, { label: "Model", value: run.agent_model, mono: true }),
    h(ContextRow, { label: "Linear issue", value: run.linear_issue_key || "None", mono: true }),
    h(ContextRow, { label: "Branch", value: run.source_branch, mono: true }),
    h(ContextRow, { label: "Pull request", value: run.mr_url || "None", mono: true, wide: true }),
    h(ContextRow, { label: "Job", value: run.job_name || "Pending", mono: true }),
    h(ContextRow, { label: "Message", value: run.message || "No message", wide: true }),
    h(ContextRow, { label: "Prompt", value: run.prompt || "", wide: true })
  );
}

function CardHeader({ icon, title, subtitle, action }) {
  return h("div", { className: "card-header" },
    h("div", { className: "card-title" },
      icon && h(Icon, { name: icon }),
      h("div", null, h("h2", null, title), subtitle && h("p", null, subtitle))
    ),
    action
  );
}

function Field({ label, name, value, onChange, required = false, placeholder = "", mono = false }) {
  return h("label", { className: `field ${mono ? "mono-field" : ""}` },
    h("span", null, label),
    h("input", {
      name,
      value,
      required,
      placeholder,
      autoComplete: "off",
      onChange: (event) => onChange(name, event.target.value),
    })
  );
}

function SecretField({ label, name, clearName, saved, value, clear, onChange }) {
  return h("div", { className: "secret-field" },
    h("div", { className: "secret-head" },
      h("span", null, label),
      h(Badge, { tone: saved ? "success" : "muted" }, saved ? "SAVED" : "EMPTY")
    ),
    h("input", {
      type: "password",
      value,
      placeholder: saved ? "Leave blank to keep saved value" : "Paste key",
      autoComplete: "new-password",
      onChange: (event) => onChange(name, event.target.value),
    }),
    h("label", { className: "mini-check" },
      h("input", { type: "checkbox", checked: Boolean(clear), onChange: (event) => onChange(clearName, event.target.checked) }),
      h("span", null, `Clear saved ${label}`)
    )
  );
}

function SelectField({ label, name, value, onChange, options }) {
  return h("label", { className: "field select-field" },
    h("span", null, label),
    h("select", { name, value, onChange: (event) => onChange(name, event.target.value) },
      options.map(([optionValue, optionLabel]) => h("option", { key: optionValue, value: optionValue }, optionLabel))
    )
  );
}

function SwitchField({ label, description, checked, onChange }) {
  return h("label", { className: "switch-field" },
    h("span", null, h("strong", null, label), h("small", null, description)),
    h("input", { type: "checkbox", checked: Boolean(checked), onChange: (event) => onChange(event.target.checked) }),
    h("i", { "aria-hidden": "true" })
  );
}

function ContextRow({ label, value, mono = false, wide = false }) {
  return h("div", { className: `context-row ${wide ? "wide" : ""}` },
    h("span", null, label),
    h("strong", { className: mono ? "mono" : "" }, value || "")
  );
}

function Metric({ label, value }) {
  return h("div", { className: "metric" }, h("span", null, label), h("strong", null, value));
}

function Badge({ tone = "muted", children }) {
  return h("span", { className: `badge ${tone}` }, children);
}

function StatusIndicator({ live, phase }) {
  if (!phase) return h(Badge, { tone: "muted" }, "NO RUN");
  return h("span", { className: `stream-indicator ${live ? "live" : "done"}` },
    h("span", { "aria-hidden": "true" }),
    live ? "LIVE" : String(phase).toUpperCase()
  );
}

function Icon({ name }) {
  return h("span", { className: "material-symbols-outlined", "aria-hidden": "true" }, name);
}

function EmptyState({ title, body }) {
  return h("div", { className: "empty-state" }, h("strong", null, title), h("span", null, body));
}

function MobileNav({ view, setView }) {
  return h("nav", { className: "mobile-nav", "aria-label": "Mobile primary" },
    NAV.map((item) => h("button", {
      key: item.id,
      className: `mobile-nav-item ${view === item.id ? "active" : ""}`,
      type: "button",
      onClick: () => setView(item.id),
    }, h(Icon, { name: item.icon }), h("span", null, item.label)))
  );
}

function ChatEntry({ entry }) {
  return h("article", { className: `chat-message ${entry.kind || "log"}` },
    h("div", { className: "chat-meta" }, [entry.title, entry.time].filter(Boolean).join(" · ")),
    h("pre", { className: entry.mono ? "mono" : "" }, redactSecrets(entry.body || ""))
  );
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

function parseLogEntries(raw) {
  const entries = [];
  const lines = String(raw || "").split(/\r?\n/);
  let jsonBlock = [];

  for (const line of lines) {
    if (!line.trim()) continue;

    if (jsonBlock.length || (line.trim() === "{" && !tryParseJSON(line))) {
      jsonBlock.push(line);
      const parsed = tryParseJSON(jsonBlock.join("\n"));
      if (parsed) {
        entries.push(entryFromFinalJSON(parsed));
        jsonBlock = [];
      }
      continue;
    }

    const parsed = tryParseJSON(line);
    if (parsed) {
      const entry = entryFromJSON(parsed);
      if (entry) entries.push(entry);
      continue;
    }

    const entry = entryFromLine(line);
    if (entry) entries.push(entry);
  }

  if (jsonBlock.length) {
    entries.push({ kind: "log", title: "Raw output", body: jsonBlock.join("\n"), mono: true });
  }

  return entries;
}

function entryFromJSON(event) {
  const part = event.part || {};
  if (event.type === "text" && part.text) {
    return { kind: "agent", title: "Agent", body: part.text.trim(), time: formatEventTime(event.timestamp) };
  }
  if (event.type === "tool_use") {
    const state = part.state || {};
    const input = state.input || {};
    const output = state.output || state.metadata?.output || "";
    const command = input.command || state.title || part.tool || "tool";
    return {
      kind: state.status === "error" ? "error" : "tool",
      title: `${part.tool || "Tool"} · ${state.status || "used"}`,
      body: [command, output].filter(Boolean).join("\n\n"),
      time: formatEventTime(event.timestamp),
      mono: true,
    };
  }
  if (event.type === "error") {
    const error = event.error || {};
    const data = error.data || {};
    return {
      kind: "error",
      title: error.name || "Error",
      body: data.message || error.message || data.responseBody || JSON.stringify(error, null, 2),
      time: formatEventTime(event.timestamp),
    };
  }
  return null;
}

function entryFromFinalJSON(result) {
  return {
    kind: result.status === "failed" ? "error" : "event",
    title: `Run ${result.status || "result"}`,
    body: [result.message, result.mr_url ? `PR: ${result.mr_url}` : "", result.completed_at ? `Completed: ${result.completed_at}` : ""].filter(Boolean).join("\n"),
  };
}

function entryFromLine(line) {
  const clean = stripANSI(line).trim();
  if (!clean || isInfrastructureLog(clean)) return null;
  const match = line.match(/^\[([^\]]+)\]\s*(.*)$/);
  if (match) return { kind: "system", title: match[1], body: stripANSI(match[2]), mono: true };
  const isFailure = /fatal:|error|failed|invalid username or token/i.test(clean);
  return { kind: isFailure ? "error" : "log", title: isFailure ? "Error output" : "Log", body: clean, mono: !isFailure };
}

function isInfrastructureLog(line) {
  return [
    /^INFO\s+\d{4}-\d{2}-\d{2}T.*\bservice=/,
    /^ERROR\s+\d{4}-\d{2}-\d{2}T.*\bservice=/,
    /^WARN\s+\d{4}-\d{2}-\d{2}T.*\bservice=/,
    /^sqlite-migration:/,
    /^Database migration complete\.$/,
    /^Performing one time database migration/,
    /^Warning: truncated output/,
    /^Total output lines:/,
    /^! agent ".+" not found\. Falling back to default agent$/,
  ].some((pattern) => pattern.test(line));
}

function phaseTone(phase) {
  const value = String(phase || "").toUpperCase();
  if (value === "SUCCEEDED") return "success";
  if (value === "FAILED" || value === "CANCELLED") return "danger";
  if (value) return "warning";
  return "muted";
}

function isTerminal(phase) {
  return ["SUCCEEDED", "FAILED", "CANCELLED"].includes(String(phase || "").toUpperCase());
}

function redactSecrets(value) {
  return String(value || "")
    .replace(/github_pat_[A-Za-z0-9_]+/g, "[redacted]")
    .replace(/gh[opsur]_[A-Za-z0-9_]+/g, "[redacted]")
    .replace(/lin_api_[A-Za-z0-9]+/g, "[redacted]")
    .replace(/sk-[A-Za-z0-9_-]+/g, "[redacted]")
    .replace(/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g, "[redacted]")
    .replace(/(https:\/\/(?:x-access-token|oauth2):)[^@\s]+(@github\.com\/)/g, "$1[redacted]$2")
    .replace(/(https:\/\/oauth2:)[^@\s]+(@gitlab\.com\/)/g, "$1[redacted]$2");
}

function tryParseJSON(value) {
  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

function stripANSI(value) {
  return String(value || "").replace(/\u001b\[[0-9;]*m/g, "");
}

function formatEventTime(value) {
  if (!value) return "";
  return formatTime(typeof value === "number" ? value : Number(value));
}

function formatTime(value) {
  if (!value) return "";
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    month: "short",
    day: "numeric",
  }).format(new Date(value));
}

ReactDOM.createRoot(document.getElementById("root")).render(h(App));
