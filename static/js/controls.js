// ==========================================
// ESP DESK_OS - UI Controls
// ==========================================
// Issue 1: All protected API calls now use authFetch()

function selectStyle(style) {
  textStyle = style;
  document.querySelectorAll(".style-card").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.style === style);
  });
}

function setDirection(dir) {
  marqueeDirection = dir;
  document.getElementById("dirLeft").classList.toggle("active", dir === "left");
  document
    .getElementById("dirRight")
    .classList.toggle("active", dir === "right");
}

function setMarqueeSize(size) {
  marqueeSize = size;
  document.querySelectorAll("[data-size]").forEach((btn) => {
    btn.classList.toggle("active", parseInt(btn.dataset.size) === size);
  });
}

function toggleHeaders() {
  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/settings/toggle-headers", { method: "POST" })
    .then((res) => res.json())
    .then((data) => {
      updateHeadersToggle(data.headersVisible);
      loadCurrent();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("toggleHeaders error:", err);
      }
    });
}

// Debounced API call for ESP refresh
const saveEspRefreshDebounced = debounce((duration) => {
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ espRefreshDuration: duration }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("updateEspRefresh error:", err);
    }
  });
}, 300);

function updateEspRefresh(value) {
  espRefreshDuration = parseInt(value);
  document.getElementById("espRefreshValue").textContent = `${(
    espRefreshDuration / 1000
  ).toFixed(1)}s`;

  // Debounced API call
  saveEspRefreshDebounced(espRefreshDuration);
}

function updateHeadersToggle(isOn) {
  const toggle = document.getElementById("headersToggle");
  toggle.classList.toggle("active", isOn);
}

// Display rotation toggle (0 = normal, 2 = 180 degrees)
let displayRotation = 0;

function toggleRotation() {
  // Toggle between 0 and 2
  displayRotation = displayRotation === 0 ? 2 : 0;
  updateRotationToggle(displayRotation === 2);

  // Save to server
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ displayRotation: displayRotation }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleRotation error:", err);
    }
  });
}

function updateRotationToggle(isOn) {
  const toggle = document.getElementById("rotationToggle");
  if (toggle) {
    toggle.classList.toggle("active", isOn);
  }
}

// ===== LED Beacon Controls =====
// Settings synced with backend and sent to ESP32 on every frame poll

let ledBeaconEnabled = true;
let ledBrightness = 50;

function toggleBeacon() {
  ledBeaconEnabled = !ledBeaconEnabled;
  const toggle = document.getElementById("beaconToggle");
  if (toggle) {
    toggle.classList.toggle("active", ledBeaconEnabled);
  }

  // Update slider visibility when beacon is disabled
  const slider = document.getElementById("ledBrightnessSlider");
  if (slider) {
    slider.disabled = !ledBeaconEnabled;
    slider.style.opacity = ledBeaconEnabled ? "1" : "0.4";
  }

  // Save to backend API
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ledBeaconEnabled: ledBeaconEnabled }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleBeacon error:", err);
    }
  });

  console.log("🛰️ LED Beacon:", ledBeaconEnabled ? "ON" : "OFF");
}

// Debounced API call for LED brightness
const saveLedBrightnessDebounced = debounce((brightness) => {
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ledBrightness: brightness }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("updateLedBrightness error:", err);
    }
  });
  console.log("🛰️ LED Brightness:", brightness + "%");
}, 300);

function updateLedBrightness(value) {
  ledBrightness = parseInt(value);
  document.getElementById(
    "ledBrightnessValue"
  ).textContent = `${ledBrightness}%`;

  // Debounced API call
  saveLedBrightnessDebounced(ledBrightness);
}

// Update beacon UI from settings (called by loadSettings in api.js)
function updateBeaconUI(brightness, enabled) {
  ledBrightness = brightness;
  ledBeaconEnabled = enabled;

  const toggle = document.getElementById("beaconToggle");
  const slider = document.getElementById("ledBrightnessSlider");
  const valueDisplay = document.getElementById("ledBrightnessValue");

  if (toggle) toggle.classList.toggle("active", ledBeaconEnabled);
  if (slider) {
    slider.value = ledBrightness;
    slider.disabled = !ledBeaconEnabled;
    slider.style.opacity = ledBeaconEnabled ? "1" : "0.4";
  }
  if (valueDisplay) valueDisplay.textContent = `${ledBrightness}%`;
}

// ===== Display Scale Controls =====
let currentDisplayScale = "normal";

function setDisplayScale(scale) {
  if (!["compact", "normal", "large"].includes(scale)) {
    return;
  }
  currentDisplayScale = scale;
  updateDisplayScaleUI(scale);

  // Save to backend API
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ displayScale: scale }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("setDisplayScale error:", err);
    }
  });

  console.log("📐 Display Scale:", scale);
}

function updateDisplayScaleUI(scale) {
  currentDisplayScale = scale;

  // Update button states
  document
    .querySelectorAll("#displayScaleButtons .scale-btn")
    .forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.scale === scale);
    });

  // Update label
  const label = document.getElementById("displayScaleValue");
  if (label) {
    const labels = { compact: "Compact", normal: "Normal", large: "Large" };
    label.textContent = labels[scale] || "Normal";
  }
}

// ===== BCD Clock Controls =====
let bcd24HourMode = true;
let bcdShowSeconds = true;

function setBCDFormat(is24Hour) {
  bcd24HourMode = is24Hour;
  updateBCDFormatUI(is24Hour);

  // Save to backend API
  authFetch("/api/settings/bcd", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ bcd24HourMode: is24Hour }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("setBCDFormat error:", err);
    }
  });

  console.log("🔢 BCD Format:", is24Hour ? "24hr" : "12hr");
}

function toggleBCDSeconds() {
  bcdShowSeconds = !bcdShowSeconds;
  updateBCDSecondsUI(bcdShowSeconds);

  // Save to backend API
  authFetch("/api/settings/bcd", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ bcdShowSeconds: bcdShowSeconds }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleBCDSeconds error:", err);
    }
  });

  console.log("🔢 BCD Seconds:", bcdShowSeconds ? "visible" : "hidden");
}

function updateBCDFormatUI(is24Hour) {
  bcd24HourMode = is24Hour;
  const btn12hr = document.getElementById("bcd12hr");
  const btn24hr = document.getElementById("bcd24hr");
  if (btn12hr) btn12hr.classList.toggle("active", !is24Hour);
  if (btn24hr) btn24hr.classList.toggle("active", is24Hour);
}

function updateBCDSecondsUI(showSeconds) {
  bcdShowSeconds = showSeconds;
  const toggle = document.getElementById("bcdSecondsToggle");
  if (toggle) toggle.classList.toggle("active", showSeconds);
}

function updateBCDSettingsUI(is24Hour, showSeconds) {
  updateBCDFormatUI(is24Hour);
  updateBCDSecondsUI(showSeconds);
}

// Load BCD settings from server (called on page load)
function loadBCDSettings() {
  authFetch("/api/settings/bcd")
    .then((res) => res.json())
    .then((data) => {
      updateBCDSettingsUI(data.bcd24HourMode, data.bcdShowSeconds);
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("loadBCDSettings error:", err);
      }
    });
}
