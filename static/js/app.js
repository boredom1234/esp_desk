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
// XSS SANITIZATION UTILITIES
// ==========================================

/**
 * Escapes HTML entities to prevent XSS attacks
 * @param {string} str - The string to escape
 * @returns {string} - The escaped string safe for HTML insertion
 */
function escapeHtml(str) {
  if (str === null || str === undefined) return "";
  const div = document.createElement("div");
  div.textContent = String(str);
  return div.innerHTML;
}

/**
 * Creates a text node safely (immune to XSS)
 * @param {string} text - The text content
 * @returns {Text} - A safe text node
 */
function safeText(text) {
  return document.createTextNode(String(text || ""));
}

/**
 * Creates an element with safe text content
 * @param {string} tag - HTML tag name
 * @param {string} text - Text content
 * @param {string} className - Optional CSS class
 * @returns {HTMLElement}
 */
function safeElement(tag, text, className) {
  const el = document.createElement(tag);
  if (className) el.className = className;
  el.textContent = String(text || "");
  return el;
}

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

// Upload file with status updates
function uploadFile(file) {
  if (!file || !file.type.startsWith("image/")) {
    setUploadStatus("error", "Invalid file type");
    return;
  }

  const formData = new FormData();
  formData.append("file", file);

  // Update UI
  setUploadStatus("uploading", "Uploading...");
  document.getElementById("dropZone").classList.add("uploading");

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
      setUploadStatus("success", `${data.frameCount} frame(s)`);
      clearUploadPreview();
      loadSettings();
      if (data.frameCount > 1) {
        startAutoPlay();
      } else {
        loadCurrent();
      }

      // Store bitmap if available (non-GIF images) for saving to cycle
      if (data.bitmap) {
        lastUploadedImage = {
          bitmap: data.bitmap,
          width: data.width,
          height: data.height,
        };
        document.getElementById("saveToCycleBtn").style.display =
          "inline-block";
      } else {
        lastUploadedImage = null;
        document.getElementById("saveToCycleBtn").style.display = "none";
      }
    })
    .catch(() => {
      setUploadStatus("error", "Upload failed");
    })
    .finally(() => {
      document.getElementById("dropZone").classList.remove("uploading");
    });
}

// Set upload status badge
function setUploadStatus(state, text) {
  const badge = document.getElementById("uploadStatus");
  badge.className = "badge";
  badge.classList.add(state);
  badge.textContent = text;

  // Reset to ready after 3 seconds
  if (state !== "uploading") {
    setTimeout(() => {
      badge.className = "badge";
      badge.textContent = "Ready";
    }, 3000);
  }
}

// Show file preview
function showUploadPreview(file) {
  const preview = document.getElementById("uploadPreview");
  const thumbnail = document.getElementById("previewThumbnail");
  const fileName = document.getElementById("previewFileName");

  fileName.textContent = file.name || "Pasted image";

  const reader = new FileReader();
  reader.onload = (e) => {
    thumbnail.src = e.target.result;
    preview.style.display = "flex";
  };
  reader.readAsDataURL(file);
}

// Clear file preview
function clearUploadPreview() {
  const fileInput = document.getElementById("imageUpload");
  fileInput.value = "";
  document.getElementById("uploadPreview").style.display = "none";
  document.getElementById("previewThumbnail").src = "";
}

// ==========================================
// DRAG AND DROP
// ==========================================
function initDragAndDrop() {
  const dropZone = document.getElementById("dropZone");
  const fileInput = document.getElementById("imageUpload");

  // Click to browse
  dropZone.addEventListener("click", () => fileInput.click());

  // Keyboard accessibility
  dropZone.addEventListener("keypress", (e) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      fileInput.click();
    }
  });

  // File input change
  fileInput.addEventListener("change", () => {
    if (fileInput.files && fileInput.files[0]) {
      showUploadPreview(fileInput.files[0]);
    }
  });

  // Drag events
  dropZone.addEventListener("dragenter", handleDragEnter);
  dropZone.addEventListener("dragover", handleDragOver);
  dropZone.addEventListener("dragleave", handleDragLeave);
  dropZone.addEventListener("drop", handleDrop);

  // Prevent default drag behavior on document
  document.addEventListener("dragover", (e) => e.preventDefault());
  document.addEventListener("drop", (e) => e.preventDefault());
}

function handleDragEnter(e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.add("drag-over");
}

function handleDragOver(e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.add("drag-over");
}

function handleDragLeave(e) {
  e.preventDefault();
  e.stopPropagation();
  // Only remove class if leaving the dropzone entirely
  if (!this.contains(e.relatedTarget)) {
    this.classList.remove("drag-over");
  }
}

function handleDrop(e) {
  e.preventDefault();
  e.stopPropagation();
  this.classList.remove("drag-over");

  const files = e.dataTransfer.files;
  if (files && files.length > 0) {
    const file = files[0];
    if (file.type.startsWith("image/")) {
      // Set the file to the input and show preview
      const fileInput = document.getElementById("imageUpload");
      const dataTransfer = new DataTransfer();
      dataTransfer.items.add(file);
      fileInput.files = dataTransfer.files;
      showUploadPreview(file);
    } else {
      setUploadStatus("error", "Not an image");
    }
  }
}

// ==========================================
// CLIPBOARD PASTE
// ==========================================
function initClipboardPaste() {
  document.addEventListener("paste", handlePaste);
}

function handlePaste(e) {
  const items = e.clipboardData?.items;
  if (!items) return;

  for (const item of items) {
    if (item.type.startsWith("image/")) {
      e.preventDefault();
      const file = item.getAsFile();
      if (file) {
        // Set the file to the input and show preview
        const fileInput = document.getElementById("imageUpload");
        const dataTransfer = new DataTransfer();
        dataTransfer.items.add(file);
        fileInput.files = dataTransfer.files;
        showUploadPreview(file);

        // Focus the drop zone to indicate where the image went
        document.getElementById("dropZone").focus();
      }
      break;
    }
  }
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

function updateEspRefresh(value) {
  espRefreshDuration = parseInt(value);
  document.getElementById("espRefreshValue").textContent = `${(
    espRefreshDuration / 1000
  ).toFixed(1)}s`;

  // Save to server
  fetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ espRefreshDuration: espRefreshDuration }),
  }).catch(() => {});
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
        // XSS-safe rendering using DOM manipulation
        display.innerHTML = "";

        const iconSpan = safeElement("span", data.icon || "🌡", "weather-icon");
        const infoSpan = safeElement(
          "span",
          `${data.temperature} · ${data.condition}`,
          "weather-info"
        );
        const windSpan = safeElement("span", data.windspeed, "weather-wind");

        display.appendChild(iconSpan);
        display.appendChild(infoSpan);
        display.appendChild(windSpan);
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
        // XSS-safe rendering using DOM manipulation
        display.innerHTML = "";

        const iconSpan = safeElement("span", data.icon || "🌡", "weather-icon");
        const infoSpan = safeElement(
          "span",
          `${data.temperature} · ${data.condition}`,
          "weather-info"
        );
        const windSpan = safeElement("span", data.windspeed, "weather-wind");

        display.appendChild(iconSpan);
        display.appendChild(infoSpan);
        display.appendChild(windSpan);
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
initDragAndDrop();
initClipboardPaste();

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
// DISPLAY CYCLE CONTROL (FLEXIBLE)
// ==========================================

let cycleItems = [];
let newItemStyle = "normal";
let cycleItemIdCounter = 0;

// Render cycle items from server data
function renderCycleItems(items) {
  cycleItems = items || [];
  const list = document.getElementById("displayCycleList");
  list.innerHTML = "";

  cycleItems.forEach((item, index) => {
    const div = document.createElement("div");
    div.className = "cycle-item";
    div.dataset.id = item.id;
    div.dataset.index = index;
    div.draggable = true;

    const typeIcon = getTypeIcon(item.type);
    const labelText = item.label || item.type;
    const extraInfo =
      item.type === "text" ? ` "${truncate(item.text, 15)}"` : "";

    // XSS-safe rendering using DOM manipulation instead of innerHTML
    // Handle element
    const handleSpan = document.createElement("span");
    handleSpan.className = "cycle-handle";
    handleSpan.textContent = "⋮⋮";

    // Checkbox label
    const checkboxLabel = document.createElement("label");
    checkboxLabel.className = "cycle-checkbox";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = item.enabled;
    // Use escaped ID to prevent injection in event handler
    const safeId = escapeHtml(item.id);
    checkbox.addEventListener("change", () => toggleCycleItem(item.id));

    const checkmark = document.createElement("span");
    checkmark.className = "checkmark";

    checkboxLabel.appendChild(checkbox);
    checkboxLabel.appendChild(checkmark);

    // Label span - use textContent for safety
    const labelSpan = document.createElement("span");
    labelSpan.className = "cycle-label";
    labelSpan.textContent = `${typeIcon} ${labelText}${extraInfo}`;

    // Delete button
    const deleteBtn = document.createElement("button");
    deleteBtn.className = "cycle-delete-btn";
    deleteBtn.title = "Remove";
    deleteBtn.textContent = "✕";
    deleteBtn.addEventListener("click", () => deleteCycleItem(item.id));

    // Assemble the element
    div.appendChild(handleSpan);
    div.appendChild(checkboxLabel);
    div.appendChild(labelSpan);
    div.appendChild(deleteBtn);

    list.appendChild(div);
  });

  initDisplayCycleDragDrop();
}

function getTypeIcon(type) {
  const icons = {
    time: "🕐",
    weather: "🌤",
    uptime: "⏱",
    text: "💬",
    image: "🖼",
  };
  return icons[type] || "📋";
}

function truncate(str, len) {
  if (!str) return "";
  return str.length > len ? str.substring(0, len) + "..." : str;
}

// Toggle item enabled state
function toggleCycleItem(id) {
  const item = cycleItems.find((i) => i.id === id);
  if (item) {
    item.enabled = !item.enabled;
    saveCycleItems();
  }
}

// Delete item
function deleteCycleItem(id) {
  cycleItems = cycleItems.filter((i) => i.id !== id);
  if (cycleItems.length === 0) {
    alert("You need at least one item in the cycle!");
    // Re-add time as fallback
    cycleItems.push({
      id: "time-fallback",
      type: "time",
      label: "🕐 Time",
      enabled: true,
      duration: 3000,
    });
  }
  saveCycleItems();
  renderCycleItems(cycleItems);
}

// Add new item
function addCycleItem() {
  const type = document.getElementById("addItemType").value;

  if (type === "text") {
    // Show text input panel
    document.getElementById("textItemConfig").style.display = "block";
    document.getElementById("newItemText").focus();
    return;
  }

  if (type === "image") {
    alert(
      "Upload an image first, then use the '💾 Save to Cycle' button that appears after upload."
    );
    return;
  }

  // Generate unique ID
  cycleItemIdCounter++;
  const id = `${type}-${Date.now()}-${cycleItemIdCounter}`;

  const labelMap = {
    time: "🕐 Time",
    weather: "🌤 Weather",
    uptime: "⏱ Uptime",
    image: "🖼 Image",
  };

  const newItem = {
    id: id,
    type: type,
    label: labelMap[type] || type,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);
}

// Set style for new text item
function setNewItemStyle(style) {
  newItemStyle = style;
  document.querySelectorAll(".style-mini-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.style === style);
  });
}

// Confirm adding text item
function confirmAddText() {
  const text = document.getElementById("newItemText").value.trim();
  if (!text) {
    alert("Please enter some text!");
    return;
  }

  cycleItemIdCounter++;
  const id = `text-${Date.now()}-${cycleItemIdCounter}`;

  const newItem = {
    id: id,
    type: "text",
    label: "💬 Message",
    text: text,
    style: newItemStyle,
    size: 2,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);

  // Reset and hide config
  document.getElementById("newItemText").value = "";
  document.getElementById("textItemConfig").style.display = "none";
}

// Save cycle items to server
function saveCycleItems() {
  fetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cycleItems: cycleItems }),
  })
    .then(() => console.log("Cycle items saved:", cycleItems.length))
    .catch((err) => console.error("Save failed:", err));
}

// Update from loadSettings
function updateDisplayCycleUI(items) {
  renderCycleItems(items);
}

// Initialize drag and drop for display cycle items
function initDisplayCycleDragDrop() {
  const list = document.getElementById("displayCycleList");
  let draggedItem = null;
  let draggedIndex = -1;

  // Remove old listeners by cloning
  const newList = list.cloneNode(true);
  list.parentNode.replaceChild(newList, list);

  newList.addEventListener("dragstart", (e) => {
    if (e.target.classList.contains("cycle-item")) {
      draggedItem = e.target;
      draggedIndex = parseInt(e.target.dataset.index);
      e.target.classList.add("dragging");
    }
  });

  newList.addEventListener("dragend", (e) => {
    if (e.target.classList.contains("cycle-item")) {
      e.target.classList.remove("dragging");

      // Get new order from DOM
      const items = Array.from(newList.querySelectorAll(".cycle-item"));
      const newOrder = items.map((el) => el.dataset.id);

      // Reorder cycleItems array
      const reordered = [];
      newOrder.forEach((id) => {
        const item = cycleItems.find((i) => i.id === id);
        if (item) reordered.push(item);
      });
      cycleItems = reordered;

      saveCycleItems();
      draggedItem = null;
      draggedIndex = -1;
    }
  });

  newList.addEventListener("dragover", (e) => {
    e.preventDefault();
    const afterElement = getDragAfterElement(newList, e.clientY);
    if (draggedItem) {
      if (afterElement == null) {
        newList.appendChild(draggedItem);
      } else {
        newList.insertBefore(draggedItem, afterElement);
      }
    }
  });
}

function getDragAfterElement(container, y) {
  const elements = [
    ...container.querySelectorAll(".cycle-item:not(.dragging)"),
  ];

  return elements.reduce(
    (closest, child) => {
      const box = child.getBoundingClientRect();
      const offset = y - box.top - box.height / 2;

      if (offset < 0 && offset > closest.offset) {
        return { offset: offset, element: child };
      } else {
        return closest;
      }
    },
    { offset: Number.NEGATIVE_INFINITY }
  ).element;
}

// Legacy function - now redirects
function updateDisplayCycle() {
  saveCycleItems();
}

// ==========================================
// SAVE IMAGE TO CYCLE
// ==========================================

function saveImageToCycle() {
  if (!lastUploadedImage) {
    alert("No image available to save! Upload an image first.");
    return;
  }

  cycleItemIdCounter++;
  const id = `image-${Date.now()}-${cycleItemIdCounter}`;

  const newItem = {
    id: id,
    type: "image",
    label: "🖼 Image",
    bitmap: lastUploadedImage.bitmap,
    width: lastUploadedImage.width,
    height: lastUploadedImage.height,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);

  // Hide button after saving
  document.getElementById("saveToCycleBtn").style.display = "none";
  lastUploadedImage = null;

  // Show feedback
  setUploadStatus("success", "Saved to cycle!");
}

// ==========================================
// AUTHENTICATION
// ==========================================

// Check authentication status on page load
async function checkAuth() {
  try {
    const res = await fetch("/api/auth/verify", {
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
    });
    const data = await res.json();

    authRequired = data.authRequired;

    if (!data.authRequired) {
      // Auth not enabled on server, show dashboard directly
      showDashboard();
      return;
    }

    if (data.authenticated) {
      showDashboard();
    } else {
      showLogin();
    }
  } catch (err) {
    console.error("Auth check failed:", err);
    // On error, try to show dashboard (might work if no auth)
    showDashboard();
  }
}

// Show login overlay
function showLogin() {
  document.getElementById("loginOverlay").style.display = "flex";
  document.getElementById("mainContainer").classList.add("blur");
  document.getElementById("loginPassword").focus();
}

// Show dashboard (hide login)
function showDashboard() {
  document.getElementById("loginOverlay").style.display = "none";
  document.getElementById("mainContainer").classList.remove("blur");

  // Show logout button if auth is enabled
  if (authRequired) {
    document.getElementById("logoutBtn").style.display = "block";
  }
}

// Handle login form submission
async function handleLogin(event) {
  event.preventDefault();

  const password = document.getElementById("loginPassword").value;
  const loginBtn = document.getElementById("loginBtn");
  const errorDiv = document.getElementById("loginError");

  // Clear previous error
  errorDiv.textContent = "";
  errorDiv.style.display = "none";

  // Show loading state
  loginBtn.disabled = true;
  loginBtn.textContent = "Authenticating...";

  try {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    });

    const data = await res.json();

    if (data.success) {
      authToken = data.token;
      showDashboard();
      // Reload settings after login
      loadSettings();
      loadWeather();
    } else {
      errorDiv.textContent = data.error || "Invalid password";
      errorDiv.style.display = "block";
      document.getElementById("loginPassword").value = "";
      document.getElementById("loginPassword").focus();
    }
  } catch (err) {
    console.error("Login failed:", err);
    errorDiv.textContent = "Connection error. Please try again.";
    errorDiv.style.display = "block";
  } finally {
    loginBtn.disabled = false;
    loginBtn.textContent = "Access Dashboard";
  }
}

// Handle logout
async function handleLogout() {
  try {
    await fetch("/api/auth/logout", {
      method: "POST",
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
    });
  } catch (err) {
    console.error("Logout error:", err);
  }

  authToken = null;
  showLogin();
}

// Add auth header to fetch requests
function authFetch(url, options = {}) {
  if (authToken) {
    options.headers = options.headers || {};
    options.headers["Authorization"] = `Bearer ${authToken}`;
  }
  return fetch(url, options).then((res) => {
    // If we get 401, show login
    if (res.status === 401 && authRequired) {
      showLogin();
      throw new Error("Unauthorized");
    }
    return res;
  });
}

// Initialize auth check on page load
checkAuth();
