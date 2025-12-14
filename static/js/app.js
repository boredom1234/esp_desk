const canvas = document.getElementById("preview");
const ctx = canvas.getContext("2d");
const scale = 4;

ctx.imageSmoothingEnabled = false;
ctx.scale(scale, scale);

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

      // Improved font handling
      if (size === 1) {
        ctx.font = "10px monospace";
      } else if (size === 2) {
        ctx.font = "20px monospace";
      } else {
        ctx.font = `${size * 10}px monospace`;
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
      // Bytes per row = ceil(width / 8)
      const bytesPerRow = Math.ceil(w / 8);

      for (let r = 0; r < h; r++) {
        for (let c = 0; c < w; c++) {
          const byteIndex = r * bytesPerRow + Math.floor(c / 8);
          const byte = data[byteIndex];
          // Check bit (MSB first)
          if (byte & (0x80 >> c % 8)) {
            ctx.fillRect(x + c, y + r, 1, 1);
          }
        }
      }
    }
  });

  // Detect if "MESSAGE:" is present to update UI state (hacky but reliable for this simple API)
  const isCustom = frame.elements.some((el) => el.value === "MESSAGE:");
  updateModeUI(isCustom);
}

function updateModeUI(isCustom) {
  const badge = document.getElementById("mode-badge");
  if (isCustom) {
    badge.textContent = "● MANUAL_OVERRIDE";
    badge.classList.add("custom-active");
  } else {
    badge.textContent = "AUTO_SEQ";
    badge.classList.remove("custom-active");
  }
}

function loadCurrent() {
  fetch("/frame/current")
    .then((res) => res.json())
    .then((frame) => drawFrame(frame))
    .catch((err) => console.error(err));
}

function nextFrame() {
  fetch("/api/control/next", { method: "POST" })
    .then((res) => res.json())
    .then((frame) => drawFrame(frame))
    .catch((err) => console.error(err));
}

function sendCustom() {
  const text = document.getElementById("customText").value;
  if (!text) return;

  fetch("/api/custom", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ text: text }),
  })
    .then(() => loadCurrent())
    .catch((err) => console.error(err));
}

function resetSystem() {
  fetch("/api/reset", { method: "POST" })
    .then(() => {
      document.getElementById("customText").value = "";
      loadCurrent();
    })
    .catch((err) => console.error(err));
}

// Initial load
loadCurrent();

// Poll every 1s
setInterval(loadCurrent, 1000);

// Allow Enter key to submit
// Allow Enter key to submit
document
  .getElementById("customText")
  .addEventListener("keypress", function (e) {
    if (e.key === "Enter") {
      sendCustom();
    }
  });

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
      loadCurrent();
    })
    .catch((err) => {
      alert("Error uploading file");
    });
}

function toggleHeaders() {
  fetch("/api/settings/toggle-headers", { method: "POST" })
    .then((res) => res.json())
    .then((data) => {
      const btn = document.getElementById("toggleHeadersBtn");
      if (data.headersVisible) {
        btn.textContent = "[ HEADERS: ON ]";
        btn.classList.remove("btn-danger");
        btn.classList.add("btn-secondary");
      } else {
        btn.textContent = "[ HEADERS: OFF ]";
        btn.classList.remove("btn-secondary");
        btn.classList.add("btn-danger");
      }
      loadCurrent(); // Refresh preview
    })
    .catch((err) => console.error(err));
}
