// Dashboard states:
//   idle      → Launch button visible
//   starting  → spawning children, button disabled with spinner
//   running   → Stop button visible (red), both children up
//   stopping  → terminating children, button disabled with spinner

const POLL_INTERVAL_MS = 3000;
const MIN_LAUNCH_DELAY_MS = 1000;

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
            <button id="primary-btn" class="action-btn">Launch</button>
          </li>
        </ul>
        <p id="control-error" class="error" hidden></p>
      </section>
    </main>
  `;

  const btn = root.querySelector("#primary-btn");
  const statusEl = root.querySelector("#app-status");
  const dotEl = root.querySelector("#status-dot");
  const errorEl = root.querySelector("#control-error");

  let state = "idle";
  let pollTimer = null;

  function render() {
    btn.classList.remove("danger");
    btn.disabled = false;

    const dotClass = {
      idle: "idle",
      starting: "pending",
      running: "running",
      stopping: "pending",
    }[state];
    dotEl.className = `status-dot ${dotClass}`;

    switch (state) {
      case "idle":
        btn.textContent = "Launch";
        statusEl.textContent = "Stopped";
        statusEl.className = "app-subtitle";
        break;

      case "starting":
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
        btn.innerHTML = `<span class="btn-spinner"></span><span>Stopping...</span>`;
        btn.disabled = true;
        statusEl.textContent = "Stopping...";
        statusEl.className = "app-subtitle pending";
        errorEl.hidden = true;
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

  async function start() {
    clearError();
    state = "starting";
    render();

    const minDelay = wait(MIN_LAUNCH_DELAY_MS);

    try {
      const items = await window.go.main.App.LaunchApps();
      await minDelay;
      const allUp = items.length > 0 && items.every((i) => i.running);

      if (allUp) {
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
      await window.go.main.App.StopApps().catch(() => {});
      showError(err.message || String(err));
      state = "idle";
      render();
    }
  }

  async function stop() {
    clearError();
    state = "stopping";
    render();

    try {
      await window.go.main.App.StopApps();
      const items = await window.go.main.App.AppsStatus();
      const anyRunning = items.some((i) => i.running);
      state = anyRunning ? "running" : "idle";
    } catch (err) {
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

  btn.addEventListener("click", () => {
    if (state === "idle") start();
    else if (state === "running") stop();
  });

  syncFromBackend();
  pollTimer = setInterval(syncFromBackend, POLL_INTERVAL_MS);

  return () => {
    clearInterval(pollTimer);
  };
}
