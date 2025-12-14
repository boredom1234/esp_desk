// ==========================================
// ESP DESK_OS - Frontend Controller
// ==========================================

const canvas = document.getElementById("preview");
const ctx = canvas.getContext("2d");
const scale = 4;

ctx.imageSmoothingEnabled = false;
ctx.scale(scale, scale);

// ==========================================
// STATE
// ==========================================
let autoPlayEnabled = true;
let autoPlayInterval = null;
let frameSpeed = 200;
let settings = {};
let marqueeDirection = "left";
let marqueeSize = 2;
let textStyle = "normal";

// ==========================================
// RENDERING
// ==========================================
function drawFrame(frame) {
  ctx.fillStyle = "#050505";
  ctx.fillRect(0, 0, 128, 64);

  if (!frame || !frame.elements) return;

  frame.elements.forEach((el) => {
    if (el.type === "text") {
      ctx.fillStyle = "#00f3ff";
      let x = el.x || 0;
      let y = el.y || 0;
      let size = el.size || 1;
      let value = el.value || "";

      if (size === 1) {
        ctx.font = "10px 'JetBrains Mono', monospace";
      } else if (size === 2) {
        ctx.font = "18px 'JetBrains Mono', monospace";
      } else {
        ctx.font = `${size * 8}px 'JetBrains Mono', monospace`;
      }

      ctx.textBaseline = "top";
      ctx.fillText(value, x, y);
    } else if (el.type === "bitmap") {
      const x = el.x || 0;
      const y = el.y || 0;
      const w = el.width || 0;
      const h = el.height || 0;
      const data = el.bitmap || [];

      ctx.fillStyle = "#00f3ff";
      const bytesPerRow = Math.ceil(w / 8);

      for (let r = 0; r < h; r++) {
        for (let c = 0; c < w; c++) {
          const byteIndex = r * bytesPerRow + Math.floor(c / 8);
          const byte = data[byteIndex];
          if (byte & (0x80 >> c % 8)) {
            ctx.fillRect(x + c, y + r, 1, 1);
          }
        }
      }
    } else if (el.type === "line") {
      ctx.fillStyle = "#00f3ff";
      ctx.fillRect(el.x || 0, el.y || 0, el.width || 1, el.height || 1);
    }
  });

  updateModeUI(settings.frameCount > 1);
}

function updateModeUI(isCustom) {
  const badge = document.getElementById("mode-badge");
  if (isCustom && settings.frameCount > 1) {
    badge.textContent = `ANIM (${settings.frameCount})`;
    badge.classList.add("custom-active");
  } else if (isCustom) {
    badge.textContent = "CUSTOM";
    badge.classList.add("custom-active");
  } else {
    badge.textContent = "AUTO";
    badge.classList.remove("custom-active");
  }

  // Update frame badge
  const frameBadge = document.getElementById("frameBadge");
  frameBadge.textContent = `Frame ${(settings.currentIndex || 0) + 1}/${
    settings.frameCount || 1
  }`;
}

// ==========================================
// API CALLS
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
      document.getElementById("speedSlider").value = frameSpeed;
      document.getElementById("speedValue").textContent = `${frameSpeed}ms`;
      updateAutoPlayButton();
      updateHeadersToggle(data.showHeaders);
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
    .then(() => {
      document.getElementById("customText").value = "";
      document.getElementById("marqueeText").value = "";
      loadSettings();
      loadCurrent();
    })
    .catch((err) => console.error(err));
}

function processAndUploadImage() {
  const fileInput = document.getElementById("imageUpload");
  if (!fileInput.files || !fileInput.files[0]) {
    alert("Please select an image or GIF first!");
    return;
  }

  const file = fileInput.files[0];
  const formData = new FormData();
  formData.append("file", file);

  fetch("/api/upload", {
    method: "POST",
    body: formData,
  })
    .then((res) => {
      if (!res.ok) throw new Error("Upload failed");
      return res.json();
    })
    .then((data) => {
      console.log(`Upload successful: ${data.frameCount} frame(s)`);
      loadSettings();
      if (data.frameCount > 1) {
        startAutoPlay();
      } else {
        loadCurrent();
      }
    })
    .catch(() => {
      alert("Error uploading file");
    });
}

// ==========================================
// AUTO-PLAY
// ==========================================
function toggleAutoPlay() {
  if (autoPlayEnabled) {
    stopAutoPlay();
  } else {
    startAutoPlay();
  }
}

function startAutoPlay() {
  autoPlayEnabled = true;
  updateAutoPlayButton();

  if (autoPlayInterval) clearInterval(autoPlayInterval);
  autoPlayInterval = setInterval(() => {
    nextFrame();
  }, frameSpeed);
}

function stopAutoPlay() {
  autoPlayEnabled = false;
  updateAutoPlayButton();

  if (autoPlayInterval) {
    clearInterval(autoPlayInterval);
    autoPlayInterval = null;
  }
}

function updateAutoPlayButton() {
  const btn = document.getElementById("autoPlayBtn");
  if (autoPlayEnabled) {
    btn.textContent = "⏸ Pause";
    btn.classList.add("playing");
  } else {
    btn.textContent = "▶ Play";
    btn.classList.remove("playing");
  }
}

function updateSpeed(value) {
  frameSpeed = parseInt(value);
  document.getElementById("speedValue").textContent = `${frameSpeed}ms`;

  if (autoPlayEnabled) {
    startAutoPlay(); // Restart with new speed
  }

  // Save to server
  fetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ frameDuration: frameSpeed }),
  }).catch(() => {});
}

// ==========================================
// UI CONTROLS
// ==========================================
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
  fetch("/api/settings/toggle-headers", { method: "POST" })
    .then((res) => res.json())
    .then((data) => {
      updateHeadersToggle(data.headersVisible);
      loadCurrent();
    })
    .catch(() => {});
}

function updateHeadersToggle(isOn) {
  const toggle = document.getElementById("headersToggle");
  toggle.classList.toggle("active", isOn);
}

// ==========================================
// WEATHER
// ==========================================
function loadWeather() {
  fetch("/api/weather")
    .then((res) => res.json())
    .then((data) => {
      const display = document.getElementById("weatherDisplay");
      if (display && data.city) {
        display.innerHTML = `
          <span class="weather-icon">${data.icon || "🌡"}</span>
          <span class="weather-info">${data.temperature} · ${
          data.condition
        }</span>
          <span class="weather-wind">${data.windspeed}</span>
        `;
      }
    })
    .catch(() => {});
}

function changeCity() {
  const select = document.getElementById("citySelect");
  const value = select.value;
  const [lat, lng, city] = value.split(",");

  fetch("/api/weather", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      city: city,
      latitude: parseFloat(lat),
      longitude: parseFloat(lng),
    }),
  })
    .then((res) => res.json())
    .then((data) => {
      const display = document.getElementById("weatherDisplay");
      if (display) {
        display.innerHTML = `
          <span class="weather-icon">${data.icon || "🌡"}</span>
          <span class="weather-info">${data.temperature} · ${
          data.condition
        }</span>
          <span class="weather-wind">${data.windspeed}</span>
        `;
      }
    })
    .catch(() => {});
}

// ==========================================
// INIT
// ==========================================
loadSettings();
loadCurrent();
loadWeather();

// Poll less frequently when not auto-playing
setInterval(() => {
  if (!autoPlayEnabled) {
    loadCurrent();
  }
  loadSettings();
}, 2000);

// Update weather every minute
setInterval(loadWeather, 60000);

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
