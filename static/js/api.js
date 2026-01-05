function loadCurrent() {
  fetch("/frame/current")
    .then((res) => {
      if (!res.ok) {
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

function showNoFramesMessage() {
  const badge = document.getElementById("mode-badge");
  if (badge) {
    badge.textContent = "LOADING...";
    badge.classList.remove("custom-active");
  }
}

function hideNoFramesMessage() {}

function loadSettings() {
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

      if (typeof updateRotationToggle === "function") {
        displayRotation = data.displayRotation || 0;
        updateRotationToggle(data.displayRotation === 2);
      }

      if (data.cycleItems) {
        updateDisplayCycleUI(data.cycleItems);
      }

      if (typeof updateBeaconUI === "function") {
        updateBeaconUI(
          data.ledBrightness || 50,
          data.ledBeaconEnabled !== false
        );
      }

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

      if (typeof updateDisplayScaleUI === "function") {
        updateDisplayScaleUI(data.displayScale || "normal");
      }
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.warn("Failed to load settings:", err.message);
      }
    });
}

function prevFrame() {
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

  const centered = document.getElementById("styleCentered")?.checked || false;
  const framed = document.getElementById("styleFramed")?.checked || false;
  const large = document.getElementById("styleLarge")?.checked || false;
  const inverted = document.getElementById("styleInverted")?.checked || false;

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
      loadSettings();

      startAutoPlay();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error(err);
      }
    });
}

function resetSystem() {
  authFetch("/api/reset", { method: "POST" })
    .then((res) => res.json())
    .then(() => {
      document.getElementById("customText").value = "";
      document.getElementById("marqueeText").value = "";
      document.getElementById("imageUpload").value = "";

      document.getElementById("citySelect").value = "12.97,80.27,Bangalore";

      loadSettings();
      loadCurrent();
      loadWeather();

      stopAutoPlay();
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

function refreshMoonPhase() {
  const btn = document.getElementById("moonRefreshBtn");
  const statusEl = document.getElementById("moonPhaseStatus");
  const constEl = document.getElementById("moonConstellation");
  const sourceEl = document.getElementById("moonDataSource");

  if (btn) {
    btn.disabled = true;
    btn.textContent = "⏳ Fetching...";
  }
  if (statusEl) statusEl.textContent = "Fetching from API...";

  authFetch("/api/moonphase/refresh", { method: "POST" })
    .then((res) => res.json())
    .then((data) => {
      if (data.success) {
        const illum = Math.round((data.illumination || 0) * 100);
        if (statusEl) statusEl.textContent = `${data.phaseName} (${illum}%)`;
        if (constEl) constEl.textContent = data.constellation || "--";
        if (sourceEl) sourceEl.textContent = "✅ Live Data";
      } else {
        if (statusEl) statusEl.textContent = data.phaseName || "Error";
        if (constEl) constEl.textContent = "--";
        if (sourceEl) sourceEl.textContent = `⚠️ ${data.source || "Fallback"}`;
        console.warn("Moon phase refresh failed:", data.error);
      }
    })
    .catch((err) => {
      if (statusEl) statusEl.textContent = "Error fetching";
      if (sourceEl) sourceEl.textContent = "❌ Failed";
      console.error("Moon phase refresh error:", err);
    })
    .finally(() => {
      if (btn) {
        btn.disabled = false;
        btn.textContent = "🔄 Refresh";
      }
    });
}
