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

  saveEspRefreshDebounced(espRefreshDuration);
}

function updateHeadersToggle(isOn) {
  const toggle = document.getElementById("headersToggle");
  toggle.classList.toggle("active", isOn);
}

let displayRotation = 0;

function toggleRotation() {
  displayRotation = displayRotation === 0 ? 2 : 0;
  updateRotationToggle(displayRotation === 2);

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

let ledBeaconEnabled = true;
let ledBrightness = 50;

function toggleBeacon() {
  ledBeaconEnabled = !ledBeaconEnabled;
  const toggle = document.getElementById("beaconToggle");
  if (toggle) {
    toggle.classList.toggle("active", ledBeaconEnabled);
  }

  const slider = document.getElementById("ledBrightnessSlider");
  if (slider) {
    slider.disabled = !ledBeaconEnabled;
    slider.style.opacity = ledBeaconEnabled ? "1" : "0.4";
  }

  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ledBeaconEnabled: ledBeaconEnabled }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleBeacon error:", err);
    }
  });
}

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
}, 300);

function updateLedBrightness(value) {
  ledBrightness = parseInt(value);
  document.getElementById(
    "ledBrightnessValue"
  ).textContent = `${ledBrightness}%`;

  saveLedBrightnessDebounced(ledBrightness);
}

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

let currentDisplayScale = "normal";

function setDisplayScale(scale) {
  if (!["compact", "normal", "large"].includes(scale)) {
    return;
  }
  currentDisplayScale = scale;
  updateDisplayScaleUI(scale);

  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ displayScale: scale }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("setDisplayScale error:", err);
    }
  });
}

function updateDisplayScaleUI(scale) {
  currentDisplayScale = scale;

  document
    .querySelectorAll("#displayScaleButtons .scale-btn")
    .forEach((btn) => {
      btn.classList.toggle("active", btn.dataset.scale === scale);
    });

  const label = document.getElementById("displayScaleValue");
  if (label) {
    const labels = { compact: "Compact", normal: "Normal", large: "Large" };
    label.textContent = labels[scale] || "Normal";
  }
}

let bcd24HourMode = true;
let bcdShowSeconds = true;

function setBCDFormat(is24Hour) {
  bcd24HourMode = is24Hour;
  updateBCDFormatUI(is24Hour);

  authFetch("/api/settings/bcd", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ bcd24HourMode: is24Hour }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("setBCDFormat error:", err);
    }
  });
}

function toggleBCDSeconds() {
  bcdShowSeconds = !bcdShowSeconds;
  updateBCDSecondsUI(bcdShowSeconds);

  authFetch("/api/settings/bcd", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ bcdShowSeconds: bcdShowSeconds }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleBCDSeconds error:", err);
    }
  });
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

let analogShowSeconds = false;
let analogShowRoman = false;

function toggleAnalogSeconds() {
  analogShowSeconds = !analogShowSeconds;
  updateAnalogSecondsUI(analogShowSeconds);

  authFetch("/api/settings/analog", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ analogShowSeconds: analogShowSeconds }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleAnalogSeconds error:", err);
    }
  });
}

function toggleAnalogRoman() {
  analogShowRoman = !analogShowRoman;
  updateAnalogRomanUI(analogShowRoman);

  authFetch("/api/settings/analog", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ analogShowRoman: analogShowRoman }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleAnalogRoman error:", err);
    }
  });
}

function updateAnalogSecondsUI(showSeconds) {
  analogShowSeconds = showSeconds;
  const toggle = document.getElementById("analogSecondsToggle");
  if (toggle) toggle.classList.toggle("active", showSeconds);
}

function updateAnalogRomanUI(showRoman) {
  analogShowRoman = showRoman;
  const toggle = document.getElementById("analogRomanToggle");
  if (toggle) toggle.classList.toggle("active", showRoman);
}

function updateAnalogSettingsUI(showSeconds, showRoman) {
  updateAnalogSecondsUI(showSeconds);
  updateAnalogRomanUI(showRoman);
}

function loadAnalogSettings() {
  authFetch("/api/settings/analog")
    .then((res) => res.json())
    .then((data) => {
      updateAnalogSettingsUI(data.analogShowSeconds, data.analogShowRoman);
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("loadAnalogSettings error:", err);
      }
    });
}

// Time Clock Settings Management

let timeShowSeconds = true;

function toggleTimeSeconds() {
  timeShowSeconds = !timeShowSeconds;
  updateTimeSecondsUI(timeShowSeconds);

  // Send update to backend
  authFetch("/api/settings/time", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ timeShowSeconds: timeShowSeconds }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("toggleTimeSeconds error:", err);
    }
  });
}

function updateTimeSecondsUI(showSeconds) {
  timeShowSeconds = showSeconds;
  const toggle = document.getElementById("timeSecondsToggle");
  if (toggle) toggle.classList.toggle("active", showSeconds);
}

function updateTimeSettingsUI(showSeconds) {
  updateTimeSecondsUI(showSeconds);
}

// Load Time settings on page load
function loadTimeSettings() {
  authFetch("/api/settings/time")
    .then((res) => res.json())
    .then((data) => {
      updateTimeSettingsUI(data.timeShowSeconds);
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("loadTimeSettings error:", err);
      }
    });
}
