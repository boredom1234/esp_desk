// ==========================================
// ESP DESK_OS - API Calls
// ==========================================
// Issue 1: All protected API calls now use authFetch()
// Issue 12: Graceful 503 handling for /frame/current

function loadCurrent() {
  // Note: /frame/current is NOT protected (ESP32 access) - uses regular fetch
  fetch("/frame/current")
    .then((res) => {
      if (!res.ok) {
        // Issue 12: Graceful 503 handling
        if (res.status === 503) {
          showNoFramesMessage();
          return null;
        }
        throw new Error(`HTTP ${res.status}`);
      }
      return res.json();
    })
    .then((frame) => {
      if (frame) {
        hideNoFramesMessage();
        drawFrame(frame);
      }
    })
    .catch((err) => {
      console.warn("Failed to load current frame:", err.message);
      showNoFramesMessage();
    });
}

// Helper for Issue 12: Show "no frames" message
function showNoFramesMessage() {
  const badge = document.getElementById("mode-badge");
  if (badge) {
    badge.textContent = "LOADING...";
    badge.classList.remove("custom-active");
  }
}

function hideNoFramesMessage() {
  // Badge will be updated by drawFrame/updateModeUI
}

function loadSettings() {
  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/settings")
    .then((res) => res.json())
    .then((data) => {
      settings = data;
      autoPlayEnabled = data.autoPlay;
      frameSpeed = data.frameDuration || 200;
      espRefreshDuration = data.espRefreshDuration || 3000;
      gifFps = data.gifFps || 0;
      document.getElementById("speedSlider").value = frameSpeed;
      document.getElementById("speedValue").textContent = `${frameSpeed}ms`;
      document.getElementById("espRefreshSlider").value = espRefreshDuration;
      document.getElementById("espRefreshValue").textContent = `${(
        espRefreshDuration / 1000
      ).toFixed(1)}s`;
      document.getElementById("gifFpsSlider").value = gifFps;
      updateGifFpsDisplay(gifFps);
      updateAutoPlayButton();
      updateHeadersToggle(data.showHeaders);

      // Update display rotation toggle (0 = normal, 2 = 180 degrees)
      if (typeof updateRotationToggle === "function") {
        displayRotation = data.displayRotation || 0;
        updateRotationToggle(data.displayRotation === 2);
      }

      // Update display cycle UI
      if (data.cycleItems) {
        updateDisplayCycleUI(data.cycleItems);
      }

      // Update LED beacon settings
      if (typeof updateBeaconUI === "function") {
        updateBeaconUI(
          data.ledBrightness || 50,
          data.ledBeaconEnabled !== false
        );
      }

      // Update LED effect settings
      if (typeof initLedSettings === "function") {
        initLedSettings(
          data.ledBeaconEnabled !== false,
          data.ledBrightness || 50,
          data.ledEffectMode || "auto",
          data.ledCustomColor || "#0064FF",
          data.ledFlashSpeed || 500,
          data.ledPulseSpeed || 1000
        );
      }
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.warn("Failed to load settings:", err.message);
      }
    });
}

function prevFrame() {
  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/control/prev", { method: "POST" })
    .then((res) => res.json())
    .then((frame) => {
      drawFrame(frame);
      loadSettings();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("prevFrame error:", err);
      }
    });
}

function nextFrame() {
  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/control/next", { method: "POST" })
    .then((res) => res.json())
    .then((frame) => {
      drawFrame(frame);
      loadSettings();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("nextFrame error:", err);
      }
    });
}

function sendCustomText() {
  const text = document.getElementById("customText").value;
  if (!text) return;

  // Read style toggle states
  const centered = document.getElementById("styleCentered")?.checked || false;
  const framed = document.getElementById("styleFramed")?.checked || false;
  const large = document.getElementById("styleLarge")?.checked || false;
  const inverted = document.getElementById("styleInverted")?.checked || false;

  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/custom/text", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      text: text,
      centered: centered,
      framed: framed,
      large: large,
      inverted: inverted,
    }),
  })
    .then((res) => res.json())
    .then(() => {
      loadSettings();
      loadCurrent();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error(err);
      }
    });
}

function sendMarquee() {
  const text = document.getElementById("marqueeText").value;
  if (!text) return;

  const speed = parseInt(document.getElementById("marqueeSpeed").value);
  const maxFrames = parseInt(document.getElementById("marqueeMaxFrames").value);
  const framed = document.getElementById("marqueeFramed")?.checked || false;

  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/custom/marquee", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      text: text,
      direction: marqueeDirection,
      size: marqueeSize,
      speed: speed,
      loops: 3,
      maxFrames: maxFrames,
      framed: framed,
    }),
  })
    .then((res) => res.json())
    .then((data) => {
      console.log(`Marquee started: ${data.frameCount} frames`);
      loadSettings();
      // Start auto-play for frontend preview (matches GIF upload behavior)
      startAutoPlay();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error(err);
      }
    });
}

function resetSystem() {
  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/reset", { method: "POST" })
    .then((res) => res.json())
    .then(() => {
      // Clear all inputs
      document.getElementById("customText").value = "";
      document.getElementById("marqueeText").value = "";
      document.getElementById("imageUpload").value = "";

      // Reset city selector to default
      document.getElementById("citySelect").value = "22.57,88.36,Kolkata";

      // Reload all state
      loadSettings();
      loadCurrent();
      loadWeather();

      // Stop auto-play
      stopAutoPlay();

      console.log("System reset to defaults");
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error(err);
      }
    });
}

function processAndUploadImage() {
  const fileInput = document.getElementById("imageUpload");
  if (!fileInput.files || !fileInput.files[0]) {
    alert("Please select an image or GIF first!");
    return;
  }

  uploadFile(fileInput.files[0]);
}
