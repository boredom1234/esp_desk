// ==========================================
// ESP DESK_OS - UI Controls
// ==========================================
// Issue 1: All protected API calls now use authFetch()

function selectStyle(style) {
  textStyle = style;
  document.querySelectorAll(".style-btn").forEach((btn) => {
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

function updateEspRefresh(value) {
  espRefreshDuration = parseInt(value);
  document.getElementById("espRefreshValue").textContent = `${(
    espRefreshDuration / 1000
  ).toFixed(1)}s`;

  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ espRefreshDuration: espRefreshDuration }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("updateEspRefresh error:", err);
    }
  });
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

function updateLedBrightness(value) {
  ledBrightness = parseInt(value);
  document.getElementById(
    "ledBrightnessValue"
  ).textContent = `${ledBrightness}%`;

  // Save to backend API
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ledBrightness: ledBrightness }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("updateLedBrightness error:", err);
    }
  });

  console.log("🛰️ LED Brightness:", ledBrightness + "%");
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
