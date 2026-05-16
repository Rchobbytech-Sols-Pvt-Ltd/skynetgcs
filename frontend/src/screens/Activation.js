const FALLBACK_ERROR = "Activation failed. Please try again.";
const SUCCESS_MESSAGE = "Activation successful. You're in.";
const SUCCESS_DELAY_MS = 1500;

function friendlyMessage(err) {
  const raw = (err && (err.message || err.toString())) || "";
  const cleaned = raw.replace(/^[\w.]+:\s*/, "").trim();
  return cleaned || FALLBACK_ERROR;
}

export function renderActivation(root, { onActivated }) {
  root.innerHTML = `
    <main class="screen activation">
      <div class="card">
        <h1>Activate Skynet GCS Launcher</h1>
        <p class="subtitle">Enter the activation key emailed to you.</p>
        <form id="activation-form" novalidate>
          <input
            id="activation-key"
            name="key"
            type="text"
            placeholder="XXXX-XXXX-XXXX-XXXX"
            autocomplete="off"
            spellcheck="false"
          />
          <div class="activation-row">
            <button type="submit" id="activate-btn">Activate</button>
            <span id="activation-spinner" class="spinner" hidden aria-hidden="true"></span>
          </div>
        </form>
        <p id="activation-error" class="error" hidden></p>
        <p id="activation-success" class="success" hidden></p>
      </div>
    </main>
  `;

  const form = root.querySelector("#activation-form");
  const input = root.querySelector("#activation-key");
  const errorEl = root.querySelector("#activation-error");
  const successEl = root.querySelector("#activation-success");
  const spinner = root.querySelector("#activation-spinner");
  const btn = root.querySelector("#activate-btn");

  function showError(msg) {
    errorEl.textContent = msg;
    errorEl.hidden = false;
    successEl.hidden = true;
  }

  function clearError() {
    errorEl.hidden = true;
    errorEl.textContent = "";
  }

  function showSuccess(msg) {
    successEl.textContent = msg;
    successEl.hidden = false;
    errorEl.hidden = true;
  }

  function setBusy(busy) {
    btn.disabled = busy;
    input.disabled = busy;
    btn.textContent = busy ? "Activating..." : "Activate";
    spinner.hidden = !busy;
  }

  input.addEventListener("input", clearError);

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    clearError();

    const key = input.value.trim();
    if (!key) {
      showError("Please enter an activation key.");
      input.focus();
      return;
    }

    setBusy(true);

    try {
      const ok = await window.go.main.App.Activate(key);
      if (ok) {
        spinner.hidden = true;
        btn.disabled = true;
        input.disabled = true;
        btn.textContent = "Activated";
        showSuccess(SUCCESS_MESSAGE);
        setTimeout(() => onActivated(), SUCCESS_DELAY_MS);
        return;
      }
      showError(FALLBACK_ERROR);
      setBusy(false);
    } catch (err) {
      showError(friendlyMessage(err));
      setBusy(false);
    }
  });
}
