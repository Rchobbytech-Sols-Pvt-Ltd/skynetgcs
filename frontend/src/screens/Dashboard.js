// Dashboard states:
//   idle       → Launch button visible
//   starting   → spawning children, button disabled with spinner
//   running    → Stop button visible (red), both children up
//   stopping   → terminating children, button disabled with spinner
//   installing → first-run component install in progress, Launch disabled
//   blocked    → components still missing after install attempt, Launch disabled

const POLL_INTERVAL_MS = 3000;
const MIN_LAUNCH_DELAY_MS = 1000;
const MIN_STOP_DELAY_MS = 1000;
const MIN_UPDATE_CHECK_DELAY_MS = 1000;

const wait = (ms) => new Promise((r) => setTimeout(r, ms));

function friendlyChildError(item) {
  switch (item.code) {
    case "not_installed":
      return "Resource not found";
    case "spawn_failed":
      return "Failed to start";
    default:
      return item.error || "Failed to start";
  }
}

function summarizeFailures(failed) {
  if (failed.length === 0) return "";
  const allMissing = failed.every((i) => i.code === "not_installed");
  if (allMissing) {
    return "Resource not found. Please reinstall or download the latest update.";
  }
  const allSpawnFailed = failed.every((i) => i.code === "spawn_failed");
  if (allSpawnFailed) {
    return "Failed to start. Please try again.";
  }
  return failed
    .map((i) => `${i.name}: ${friendlyChildError(i)}`)
    .join(" — ");
}

export function renderDashboard(root) {
  root.innerHTML = `
    <main class="screen dashboard">
      <section class="apps">
        <h2 class="section-title">Apps</h2>
        <ul class="app-list">
          <li class="app-row">
            <div class="app-info">
              <div class="app-title-row">
                <span class="app-title">Skynet GCS</span>
                <span id="status-dot" class="status-dot idle" aria-hidden="true"></span>
              </div>
              <span id="app-status" class="app-subtitle">Stopped</span>
            </div>
            <div class="app-actions">
              <button id="settings-btn" class="icon-btn" type="button" title="Settings" aria-label="Settings">
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <circle cx="12" cy="12" r="3"></circle>
                  <path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1.1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1.1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"></path>
                </svg>
              </button>
              <button id="update-btn" class="icon-btn" type="button" title="Check for updates" aria-label="Check for updates">
                <svg viewBox="0 0 24 24" aria-hidden="true">
                  <path d="M12 3v12"></path>
                  <path d="m7 10 5 5 5-5"></path>
                  <path d="M5 21h14"></path>
                </svg>
              </button>
              <button id="primary-btn" class="action-btn">Launch</button>
            </div>
          </li>
        </ul>
        <p id="control-error" class="error" hidden></p>
      </section>
      <div id="settings-backdrop" class="modal-backdrop" hidden>
        <section class="update-popup" role="dialog" aria-modal="true" aria-labelledby="settings-title">
          <button id="settings-close-btn" class="modal-close" type="button" aria-label="Close settings">&times;</button>
          <h3 id="settings-title">Settings</h3>
          <label class="setting-row">
            <input id="setting-show-consoles" type="checkbox" />
            <span class="setting-text">
              <span class="setting-label">Debug mode</span>
              <span class="setting-hint">Show logs in a terminal window.</span>
            </span>
          </label>
        </section>
      </div>
      <div id="update-backdrop" class="modal-backdrop" hidden>
        <section class="update-popup" role="dialog" aria-modal="true" aria-labelledby="update-title">
          <span id="download-indicator" class="download-indicator" title="Downloading" aria-label="Downloading" hidden>
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M12 3v12"></path>
              <path d="m7 10 5 5 5-5"></path>
              <path d="M5 21h14"></path>
            </svg>
            <span id="download-percent" class="download-percent">0%</span>
          </span>
          <button id="update-close-btn" class="modal-close" type="button" aria-label="Close update dialog">&times;</button>
          <h3 id="update-title">Updates</h3>
          <p id="update-message" class="update-message">Checking for updates...</p>
          <div id="update-progress" class="update-progress">
            <span class="spinner"></span>
          </div>
          <button id="download-update-btn" class="action-btn download" type="button" hidden>Download latest</button>
        </section>
      </div>
    </main>
  `;

  const btn = root.querySelector("#primary-btn");
  const updateBtn = root.querySelector("#update-btn");
  const settingsBtn = root.querySelector("#settings-btn");
  const settingsBackdrop = root.querySelector("#settings-backdrop");
  const settingsCloseBtn = root.querySelector("#settings-close-btn");
  const showConsolesCheckbox = root.querySelector("#setting-show-consoles");
  const updateBackdrop = root.querySelector("#update-backdrop");
  const updateCloseBtn = root.querySelector("#update-close-btn");
  const updateMessageEl = root.querySelector("#update-message");
  const updateProgressEl = root.querySelector("#update-progress");
  const downloadIndicatorEl = root.querySelector("#download-indicator");
  const downloadPercentEl = root.querySelector("#download-percent");
  const downloadUpdateBtn = root.querySelector("#download-update-btn");
  const statusEl = root.querySelector("#app-status");
  const dotEl = root.querySelector("#status-dot");
  const errorEl = root.querySelector("#control-error");

  let state = "idle";
  let latestRelease = null;
  let updateChecking = false;
  let downloadInProgress = false;
  let downloadRunId = 0;
  let downloadProgressTimer = null;
  let pollTimer = null;

  function render() {
    btn.classList.remove("danger", "pending");
    btn.disabled = false;
    updateBtn.disabled = updateChecking || state === "running" || state === "installing";

    const dotClass = {
      idle: "idle",
      starting: "pending",
      running: "running",
      stopping: "pending",
      installing: "pending",
      blocked: "idle",
    }[state];
    dotEl.className = `status-dot ${dotClass}`;

    switch (state) {
      case "idle":
        btn.textContent = "Launch";
        statusEl.textContent = "Stopped";
        statusEl.className = "app-subtitle";
        break;

      case "starting":
        btn.classList.add("pending");
        btn.innerHTML = `<span class="btn-spinner"></span><span>Launching...</span>`;
        btn.disabled = true;
        statusEl.textContent = "Launching...";
        statusEl.className = "app-subtitle pending";
        errorEl.hidden = true;
        break;

      case "running":
        btn.textContent = "Stop";
        btn.classList.add("danger");
        statusEl.textContent = "Running";
        statusEl.className = "app-subtitle running";
        break;

      case "stopping":
        btn.classList.add("pending");
        btn.innerHTML = `<span class="btn-spinner"></span><span>Stopping...</span>`;
        btn.disabled = true;
        statusEl.textContent = "Stopping...";
        statusEl.className = "app-subtitle pending";
        errorEl.hidden = true;
        break;

      case "installing":
        btn.classList.add("pending");
        btn.innerHTML = `<span class="btn-spinner"></span><span>Setting up...</span>`;
        btn.disabled = true;
        statusEl.textContent = "Installing components...";
        statusEl.className = "app-subtitle pending";
        errorEl.hidden = true;
        break;

      case "blocked":
        btn.textContent = "Setup required";
        btn.disabled = true;
        statusEl.textContent = "Components not installed";
        statusEl.className = "app-subtitle";
        break;
    }
  }

  function showError(msg) {
    errorEl.textContent = msg;
    errorEl.hidden = false;
  }

  function clearError() {
    errorEl.hidden = true;
    errorEl.textContent = "";
  }

  function setUpdatePopup({
    message,
    checking = false,
    release = null,
    tone = "neutral",
    downloading = false,
  }) {
    latestRelease = release;
    downloadInProgress = downloading;
    updateMessageEl.textContent = message;
    updateMessageEl.className = `update-message ${tone}`;
    updateProgressEl.hidden = !checking;
    downloadUpdateBtn.hidden = !release;
    downloadIndicatorEl.hidden = !downloading;
    if (downloading) downloadPercentEl.textContent = "0%";
    downloadUpdateBtn.classList.toggle("danger", downloading);
    downloadUpdateBtn.classList.toggle("download", !downloading);
    downloadUpdateBtn.textContent = downloading ? "Cancel download" : "Download latest";
  }

  function stopDownloadProgressPolling() {
    if (downloadProgressTimer) {
      clearInterval(downloadProgressTimer);
      downloadProgressTimer = null;
    }
  }

  async function refreshDownloadProgress() {
    try {
      const status = await window.go.main.App.UpdateStatus();
      const percent = Math.max(0, Math.min(100, Number(status?.percent || 0)));
      downloadPercentEl.textContent = `${percent}%`;
    } catch (err) {
      console.error("[Dashboard] Update status error:", err);
    }
  }

  function startDownloadProgressPolling() {
    stopDownloadProgressPolling();
    refreshDownloadProgress();
    downloadProgressTimer = setInterval(refreshDownloadProgress, 250);
  }

  async function checkForUpdates() {
    updateBackdrop.hidden = false;
    updateChecking = true;
    render();
    setUpdatePopup({ message: "Checking for updates...", checking: true, tone: "pending" });
    const minDelay = wait(MIN_UPDATE_CHECK_DELAY_MS);

    try {
      const result = await window.go.main.App.CheckForUpdates();
      await minDelay;
      if (result?.update_available && result.release) {
        const version = result.latest_version || result.release.tag_name || "latest";
        setUpdatePopup({
          message: `Update ${version} is available.`,
          release: result.release,
          tone: "pending",
        });
        return;
      }

      const version = result?.current_version || "current version";
      setUpdatePopup({
        message: `No new update available. You are on ${version}.`,
        tone: "success",
      });
    } catch (err) {
      await minDelay;
      setUpdatePopup({ message: err.message || String(err), tone: "error" });
    } finally {
      updateChecking = false;
      render();
    }
  }

  async function start() {
    clearError();
    state = "starting";
    console.log("[Dashboard] Starting apps...");
    render();

    const minDelay = wait(MIN_LAUNCH_DELAY_MS);

    try {
      const items = await window.go.main.App.LaunchApps();
      await minDelay;
      console.log("[Dashboard] Launch results:", items);
      const allUp = items.length > 0 && items.every((i) => i.running);

      if (allUp) {
        console.log("[Dashboard] All components running successfully");
        state = "running";
        render();
        return;
      }

      const failed = items.filter((i) => !i.running);
      await window.go.main.App.StopApps();
      showError(summarizeFailures(failed));
      state = "idle";
      render();
    } catch (err) {
      await minDelay;
      console.error("[Dashboard] Launch catch error:", err);
      await window.go.main.App.StopApps().catch(() => {});
      showError(err.message || String(err));
      state = "idle";
      render();
    }
  }

  async function stop() {
    clearError();
    state = "stopping";
    console.log("[Dashboard] Stopping apps...");
    render();
    const minDelay = wait(MIN_STOP_DELAY_MS);

    try {
      await window.go.main.App.StopApps();
      await minDelay;
      const items = await window.go.main.App.AppsStatus();
      const anyRunning = items.some((i) => i.running);
      state = anyRunning ? "running" : "idle";
    } catch (err) {
      await minDelay;
      showError(err.message || String(err));
      const items = await window.go.main.App.AppsStatus().catch(() => []);
      state = items.some((i) => i.running) ? "running" : "idle";
    }

    render();
  }

  async function syncFromBackend() {
    try {
      const items = await window.go.main.App.AppsStatus();
      if (state !== "running" && state !== "idle") return;
      state = items.some((i) => i.running) ? "running" : "idle";
      render();
    } catch {
      // ignore transient binding errors
    }
  }

  async function ensureComponentsInstalled() {
    let missing;
    try {
      missing = await window.go.main.App.MissingComponents();
    } catch (err) {
      console.error("[Dashboard] MissingComponents check failed:", err);
      return;
    }
    if (!missing || missing.length === 0) return;

    console.log("[Dashboard] Missing components:", missing);
    state = "installing";
    render();

    const ok = await runFirstRunInstall(missing);
    let stillMissing = [];
    try {
      stillMissing = (await window.go.main.App.MissingComponents()) || [];
    } catch (err) {
      console.error("[Dashboard] MissingComponents recheck failed:", err);
    }

    if (ok && stillMissing.length === 0) {
      state = "idle";
      // Auto-dismiss the success popup after a moment.
      setTimeout(() => {
        if (!downloadInProgress) updateBackdrop.hidden = true;
      }, 1200);
    } else {
      state = "blocked";
      showError(
        "Components couldn't be installed. Click the update button to retry.",
      );
    }
    render();
  }

  async function runFirstRunInstall(missing) {
    updateBackdrop.hidden = false;
    setUpdatePopup({
      message: `Setting up components (${missing.join(", ")})...`,
      checking: true,
      tone: "pending",
    });

    let release;
    try {
      const result = await window.go.main.App.CheckForUpdates();
      release = result && result.release;
      if (!release) {
        setUpdatePopup({
          message: "No release available. Check your connection and try again.",
          tone: "error",
        });
        return false;
      }
    } catch (err) {
      setUpdatePopup({
        message: `Couldn't reach update server: ${err.message || String(err)}`,
        tone: "error",
      });
      return false;
    }

    const runId = downloadRunId + 1;
    downloadRunId = runId;
    setUpdatePopup({
      message: "Downloading components...",
      release,
      tone: "pending",
      downloading: true,
    });
    startDownloadProgressPolling();

    try {
      await window.go.main.App.DownloadUpdate(release);
      if (downloadRunId !== runId) return false;
      stopDownloadProgressPolling();
      downloadPercentEl.textContent = "100%";
      setUpdatePopup({
        message: "Components installed.",
        tone: "success",
      });
      return true;
    } catch (err) {
      if (downloadRunId !== runId) return false;
      stopDownloadProgressPolling();
      const message = err.message || String(err);
      const cancelled =
        message.includes("canceled") || message.includes("cancelled");
      setUpdatePopup({
        message: cancelled ? "Install cancelled." : message,
        tone: cancelled ? "pending" : "error",
      });
      return false;
    }
  }

  async function cancelDownloadAndClose() {
    const wasDownloading = downloadInProgress;
    downloadRunId += 1;
    downloadInProgress = false;
    stopDownloadProgressPolling();
    downloadIndicatorEl.hidden = true;
    updateBackdrop.hidden = true;

    if (wasDownloading) {
      await window.go.main.App.CancelUpdate().catch((err) => {
        console.error("[Dashboard] Cancel update error:", err);
      });
    }
  }

  async function startDownload() {
    if (!latestRelease || downloadInProgress) return;

    const runId = downloadRunId + 1;
    downloadRunId = runId;
    setUpdatePopup({
      message: `Downloading ${latestRelease.tag_name || latestRelease.name || "the latest update"}...`,
      release: latestRelease,
      tone: "pending",
      downloading: true,
    });
    startDownloadProgressPolling();

    try {
      await window.go.main.App.DownloadUpdate(latestRelease);
      if (downloadRunId !== runId) return;
      stopDownloadProgressPolling();
      downloadPercentEl.textContent = "100%";
      setUpdatePopup({
        message: "Update downloaded successfully.",
        tone: "success",
      });
      if (state === "blocked") {
        try {
          const stillMissing =
            (await window.go.main.App.MissingComponents()) || [];
          if (stillMissing.length === 0) {
            clearError();
            state = "idle";
            render();
          }
        } catch (err) {
          console.error("[Dashboard] post-update recheck failed:", err);
        }
      }
    } catch (err) {
      if (downloadRunId !== runId) return;
      stopDownloadProgressPolling();
      const message = err.message || String(err);
      setUpdatePopup({
        message: message.includes("canceled") || message.includes("cancelled")
          ? "Download cancelled."
          : message,
        release: latestRelease,
        tone: message.includes("canceled") || message.includes("cancelled") ? "pending" : "error",
      });
    }
  }

  async function openSettings() {
    try {
      const s = await window.go.main.App.GetSettings();
      showConsolesCheckbox.checked = !!(s && s.show_component_consoles);
    } catch (err) {
      console.error("[Dashboard] GetSettings failed:", err);
      showConsolesCheckbox.checked = false;
    }
    settingsBackdrop.hidden = false;
  }

  async function persistSettings() {
    try {
      await window.go.main.App.SetSettings({
        show_component_consoles: showConsolesCheckbox.checked,
      });
    } catch (err) {
      console.error("[Dashboard] SetSettings failed:", err);
    }
  }

  btn.addEventListener("click", () => {
    if (state === "idle") start();
    else if (state === "running") stop();
  });

  updateBtn.addEventListener("click", checkForUpdates);

  settingsBtn.addEventListener("click", openSettings);
  settingsCloseBtn.addEventListener("click", () => {
    settingsBackdrop.hidden = true;
  });
  settingsBackdrop.addEventListener("click", (event) => {
    if (event.target === settingsBackdrop) settingsBackdrop.hidden = true;
  });
  showConsolesCheckbox.addEventListener("change", persistSettings);

  updateCloseBtn.addEventListener("click", () => {
    cancelDownloadAndClose();
  });

  updateBackdrop.addEventListener("click", (event) => {
    if (event.target === updateBackdrop) {
      cancelDownloadAndClose();
    }
  });

  downloadUpdateBtn.addEventListener("click", () => {
    if (!latestRelease) return;
    if (downloadInProgress) {
      cancelDownloadAndClose();
      return;
    }

    startDownload();
  });

  syncFromBackend();
  ensureComponentsInstalled();
  pollTimer = setInterval(syncFromBackend, POLL_INTERVAL_MS);

  return () => {
    clearInterval(pollTimer);
    stopDownloadProgressPolling();
  };
}
