// ==========================================
// ESP DESK_OS - API Calls
// ==========================================

function loadCurrent() {
  fetch("/frame/current")
    .then((res) => res.json())
    .then((frame) => drawFrame(frame))
    .catch(() => {});
}

function loadSettings() {
  fetch("/api/settings")
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

      // Update display cycle UI
      if (data.cycleItems) {
        updateDisplayCycleUI(data.cycleItems);
      }
    })
    .catch(() => {});
}

function nextFrame() {
  fetch("/api/control/next", { method: "POST" })
    .then((res) => res.json())
    .then((frame) => {
      drawFrame(frame);
      loadSettings();
    })
    .catch(() => {});
}

function sendCustomText() {
  const text = document.getElementById("customText").value;
  if (!text) return;

  fetch("/api/custom/text", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      text: text,
      style: textStyle,
      size: 2,
    }),
  })
    .then((res) => res.json())
    .then(() => {
      loadSettings();
      loadCurrent();
    })
    .catch((err) => console.error(err));
}

function sendMarquee() {
  const text = document.getElementById("marqueeText").value;
  if (!text) return;

  const speed = parseInt(document.getElementById("marqueeSpeed").value);

  fetch("/api/custom/marquee", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      text: text,
      direction: marqueeDirection,
      size: marqueeSize,
      speed: speed,
      loops: 3,
    }),
  })
    .then((res) => res.json())
    .then((data) => {
      console.log(`Marquee started: ${data.frameCount} frames`);
      loadSettings();
      startAutoPlay();
    })
    .catch((err) => console.error(err));
}

function resetSystem() {
  fetch("/api/reset", { method: "POST" })
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
    .catch((err) => console.error(err));
}

function processAndUploadImage() {
  const fileInput = document.getElementById("imageUpload");
  if (!fileInput.files || !fileInput.files[0]) {
    alert("Please select an image or GIF first!");
    return;
  }

  uploadFile(fileInput.files[0]);
}
