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
let authToken = null; // Session token for authentication
let authRequired = false; // Whether authentication is enabled on server

// ==========================================
// INITIALIZATION
// ==========================================
loadSettings();
loadCurrent();
loadWeather();
initDragAndDrop();
initClipboardPaste();
initCharCounters();

// ==========================================
// POLLING & AUTO-REFRESH
// ==========================================

// Live refresh - always poll for the latest frame data
// This ensures the Visual Feed stays updated without manual refresh
setInterval(() => {
  // Always load the current frame to detect backend changes
  loadCurrentWithChangeDetection();
  loadSettings();
}, 1500);

// Update weather every minute
setInterval(loadWeather, 60000);

// Smart frame loading that detects if content has changed
function loadCurrentWithChangeDetection() {
  fetch("/frame/current")
    .then((res) => res.json())
    .then((frame) => {
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

function updateGifFps(value) {
  gifFps = parseInt(value);
  updateGifFpsDisplay(gifFps);

  // Save to server
  fetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ gifFps: gifFps }),
  }).catch(() => {});
}

function resetGifFps() {
  gifFps = 0;
  document.getElementById("gifFpsSlider").value = 0;
  updateGifFpsDisplay(0);

  // Save to server
  fetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ gifFps: 0 }),
  }).catch(() => {});
}

// ==========================================
// AUTHENTICATION INIT
// ==========================================
// Initialize auth check on page load
checkAuth();
