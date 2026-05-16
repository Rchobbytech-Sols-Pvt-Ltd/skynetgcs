# Skynet GCS

**Secure, next-generation drone Ground Control Software** by RCHobbytech Solutions Pvt. Ltd.

Skynet GCS is an encrypted ground control platform for VTOL and multirotor UAVs. It delivers encrypted telemetry, mission planning, signed firmware management, and tamper-proof logging — built for both operational drone deployments and research environments where security and auditability matter.

---

## Features

### Flight & Operations
- **MAVLink v2 telemetry** over encrypted TCP / UDP / UART
- **Mission planner** with graphical waypoint design and live execution
- **Flight modes:** GUIDED, AUTO, RTL
- **Live RTSP video** with on-screen telemetry overlay
- **Flight replay** with encrypted mission logs (`.tlog`, `.bin`, `.gpx`)

### Security
- End-to-end **AES-256-GCM** encryption with ECC + RSA-4096 handshake
- **Certificate-based drone authentication** — only pre-registered drones connect
- **Signed firmware verification** before flashing
- **Role-based access control** (User / Developer / Admin)
- **Secure boot** — binary hash verification before launch
- OS hardening: USB autorun disabled, restricted ports, audit logging
- SHA-256 integrity hashing on all logs

### Interfaces
- **Map engines:** offline TileServer, Bing, Google, OSM, DSM
- **Input devices:** joystick, gamepad, touchscreen, mouse
- **Video:** RTSP / H.264 + optional AI object-detection overlay

### Compliance & Safety
- GPS, battery, and temperature warning thresholds
- 2-year encrypted data retention
- Secure handshake required before any flight operation

---

## System Requirements

| | Minimum | Recommended |
|--|--|--|
| **OS** | Windows 10 (1909+) | Windows 11 |
| **Architecture** | 64-bit (x64) | 64-bit (x64) |
| **RAM** | 8 GB | 16 GB |
| **Free disk space** | 500 MB | 1 GB |
| **Network** | Required for activation & updates | Stable broadband |

You'll need administrator rights for the **installation only**, not for everyday use.

---

## Installation

### 1. Download the installer

Get the latest `skynetgcs-launcher-setup.exe` from the [official releases page](https://github.com/jhakrishan20/skynetgcs/releases/latest).

> Always download from the official link above. Do not install Skynet GCS from third-party mirrors.

### 2. Run setup

Double-click the installer. Windows SmartScreen may warn you the first time — click **More info → Run anyway** to continue.

The installer will:
- Install the launcher to `C:\Program Files\Skynet GCS Launcher\`
- Create Start Menu shortcuts (and a Desktop shortcut if you opt in)
- Offer to launch the app when finished

### 3. Activate your license (first launch only)

The first time you open Skynet GCS Launcher, you'll see the **Activation** screen.

1. Enter the activation key from your purchase email (format: `XXXX-XXXX-XXXX-XXXX`).
2. Click **Activate**.
3. Wait for the green confirmation — you'll be routed to the dashboard automatically.

> **One license = one machine.** Activation keys are bound to your computer's hardware fingerprint. To transfer a license to another machine, contact support.

---

## Using Skynet GCS

The dashboard shows the **Skynet GCS** application with a status indicator and a **Launch** button.

### Status indicator

A small colored dot next to the app name shows the current state:

| Color | Meaning |
|--|--|
| 🔴 **Red** | Stopped — not running |
| 🟠 **Orange** (pulsing) | Starting up or shutting down |
| 🟢 **Green** | All components are running |

### To start the application

Click **Launch**. Skynet GCS will:
1. Start the `airunit` and `hci` components in parallel
2. Show "Launching..." while they come up
3. Switch the button to red **Stop** when everything is running

### To stop

Click **Stop**. Both components are terminated cleanly and the launcher returns to idle.

> Closing the launcher window also stops every running component automatically.

---

## Updates

Skynet GCS Launcher checks our official release server for updates. When a new version is available, it can download and install components in place — no manual reinstall needed.

The update process:
1. Stops running components
2. Downloads the latest `airunit` and `hci` modules
3. Verifies and extracts them
4. Restarts everything

You can keep using older versions, but for security patches we recommend staying current.

---

## Troubleshooting

### "Invalid activation key"
- Check the key for typos — it's case-sensitive and there are no spaces.
- The key may have already been activated on a different machine. Contact support to transfer it.

### "No internet connection"
- The launcher must reach our license and update servers. Check your network.
- Corporate firewalls sometimes block outbound HTTPS. Ask IT to allowlist Skynet servers.

### "This key is already activated on another machine"
- One license is bound to one machine. Contact support if you need to migrate.

### "Resource not found" when clicking Launch
- The drone components (`airunit`, `hci`) aren't installed yet, or are corrupted.
- Use the in-app update flow to fetch them, or reinstall Skynet GCS Launcher.

### App won't open after installation
- Confirm your Windows version meets the requirements above.
- Try running once as Administrator (right-click the shortcut → **Run as administrator**).
- Check `%APPDATA%\SkynetGCS\` for log files when reporting issues.

### Where is my data stored?
- **Activation file:** `%APPDATA%\SkynetGCS\activation.json`
- **Application binaries:** `C:\Program Files\Skynet GCS Launcher\` (or your custom install path)

---

## Uninstalling

**Settings → Apps → Installed apps → Skynet GCS Launcher → Uninstall**, or run the uninstaller from the Start Menu folder.

To remove user data as well (activation, cached files):

```powershell
Remove-Item $env:APPDATA\SkynetGCS -Recurse -Force
```

---

## Support

- **Email:** skynetintel@dronestechlab.com
- **Issues / bug reports:** https://github.com/jhakrishan20/skynetgcs/issues
- **Documentation:** https://github.com/jhakrishan20/skynetgcs

When reporting a problem, please include:
- Your Skynet GCS Launcher version (visible in the launcher window title or About dialog)
- Windows version (`winver`)
- A description of what you were doing when the issue occurred

---

## License

© 2026 RCHobbytech Solutions Pvt. Ltd. All rights reserved.

See [LICENSE](LICENSE) for the full license terms.
