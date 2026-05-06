import { renderActivation } from "./screens/Activation.js";
import { renderDashboard } from "./screens/Dashboard.js";

const root = document.getElementById("app");

async function route() {
  const activated =
    window.go && window.go.main && window.go.main.App
      ? await window.go.main.App.IsActivated()
      : false;

  if (activated) {
    renderDashboard(root, { onSignOut: route });
  } else {
    renderActivation(root, { onActivated: route });
  }
}

route();
