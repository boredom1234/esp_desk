// ==========================================
// ESP DESK_OS - Main Application Controller
// ==========================================
// This file coordinates all modules and handles initialization.
// Individual functionality is split into separate files:
// - utils.js     : XSS sanitization utilities
// - render.js    : Canvas rendering
// - api.js       : API calls
// - upload.js    : File upload, drag/drop, clipboard
// - autoplay.js  : Auto-play controls
// - controls.js  : UI controls
// - weather.js   : Weather functions
// - cycle.js     : Display cycle management
// - auth.js      : Authentication
// ==========================================

// ==========================================
// CANVAS & DOM SETUP
// ==========================================
const canvas = document.getElementById("preview");
const ctx = canvas.getContext("2d");
const scale = 4;

ctx.imageSmoothingEnabled = false;
ctx.scale(scale, scale);

// ==========================================
// GLOBAL STATE
// ==========================================
let autoPlayEnabled = true;
let autoPlayInterval = null;
let frameSpeed = 200;
let espRefreshDuration = 3000;
let gifFps = 0; // 0 = original timing, 5-30 = custom FPS
let settings = {};
let marqueeDirection = "left";
let marqueeSize = 2;
let textStyle = "normal";
let lastFrameHash = null; // Track frame changes for smart refresh
let lastUploadedImage = null; // { bitmap, width, height } for saving to cycle
// Note: authToken and authRequired are defined in auth.js

// ==========================================
// INITIALIZATION
// ==========================================
// Note: API calls are deferred until auth is verified via initAfterAuth()
initDragAndDrop();
initClipboardPaste();
initCharCounters();

// Called by auth.js after authentication is verified
function initAfterAuth() {
  loadSettings();
  loadCurrent();
  loadWeather();
  loadTimezone(); // Issue 13: Load timezone setting
  loadBCDSettings(); // Load BCD clock settings
  loadAnalogSettings(); // Load Analog clock settings
  loadSpotifyStatus(); // Load Spotify status
  initPomodoro(); // Initialize Pomodoro timer

  // Start polling only after auth verified
  startPolling();
}

// ==========================================
// POLLING & AUTO-REFRESH
// ==========================================

let pollingInterval = null;
let settingsPollingInterval = null;
let weatherInterval = null;

function startPolling() {
  // Issue 8: Separate frame polling from settings polling
  // Frame polling: 1.5s (needs to be responsive)
  if (pollingInterval) clearInterval(pollingInterval);
  pollingInterval = setInterval(() => {
    loadCurrentWithChangeDetection();
  }, 1500);

  // Issue 8: Settings polling reduced to 10s (settings change infrequently)
  if (settingsPollingInterval) clearInterval(settingsPollingInterval);
  settingsPollingInterval = setInterval(() => {
    loadSettings();
  }, 10000);

  // Update weather every minute
  if (weatherInterval) clearInterval(weatherInterval);
  weatherInterval = setInterval(loadWeather, 60000);
}

// Smart frame loading that detects if content has changed
function loadCurrentWithChangeDetection() {
  fetch("/frame/current")
    .then((res) => {
      if (!res.ok) {
        if (res.status === 503) {
          // Issue 12: Graceful handling
          return null;
        }
        throw new Error(`HTTP ${res.status}`);
      }
      return res.json();
    })
    .then((frame) => {
      if (!frame) return;
      // Create a simple hash of the frame to detect changes
      const frameHash = JSON.stringify(frame);
      if (frameHash !== lastFrameHash) {
        lastFrameHash = frameHash;
        drawFrame(frame);
        showRefreshIndicator();
      }
    })
    .catch(() => {});
}

// Visual indicator when new data is received
function showRefreshIndicator() {
  const badge = document.getElementById("mode-badge");
  if (badge) {
    badge.classList.add("refresh-pulse");
    setTimeout(() => badge.classList.remove("refresh-pulse"), 300);
  }
}

// ==========================================
// EVENT LISTENERS
// ==========================================

// Enter key handlers
document.getElementById("customText").addEventListener("keypress", (e) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    sendCustomText();
  }
});

document.getElementById("marqueeText").addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    sendMarquee();
  }
});

// ==========================================
// GIF FPS CONTROL
// ==========================================
function updateGifFpsDisplay(fps) {
  const label = document.getElementById("gifFpsValue");
  if (fps === 0) {
    label.textContent = "Original";
  } else {
    label.textContent = `${fps} FPS`;
  }
}

// Debounced API call for GIF FPS setting
const saveGifFpsDebounced = debounce((fps) => {
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ gifFps: fps }),
  }).catch(() => {});
}, 300);

function updateGifFps(value) {
  gifFps = parseInt(value);
  updateGifFpsDisplay(gifFps);

  // Debounced API call
  saveGifFpsDebounced(gifFps);
}

function resetGifFps() {
  gifFps = 0;
  document.getElementById("gifFpsSlider").value = 0;
  updateGifFpsDisplay(0);

  // Immediate call since this is a button action, not a slider drag
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ gifFps: 0 }),
  }).catch(() => {});
}

// ==========================================
// TIMEZONE CONTROL (Issue 13)
// ==========================================
function loadTimezone() {
  authFetch("/api/settings/timezone")
    .then((res) => res.json())
    .then((data) => {
      if (data.timezone) {
        const select = document.getElementById("timezoneSelect");
        if (select) {
          select.value = data.timezone;
        }
      }
    })
    .catch(() => {});
}

function updateTimezone() {
  const select = document.getElementById("timezoneSelect");
  if (!select) return;

  const timezone = select.value;

  authFetch("/api/settings/timezone", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ timezone: timezone }),
  })
    .then((res) => res.json())
    .then((data) => {
      if (data.status === "updated") {
        //(`Timezone updated to ${data.timezone}`);
      }
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("Failed to update timezone:", err);
      }
    });
}

// ==========================================
// AUTHENTICATION INIT
// ==========================================
// Initialize auth check on page load - this will call initAfterAuth() when ready
checkAuth();
