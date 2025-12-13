const canvas = document.getElementById("preview");
const ctx = canvas.getContext("2d");
const scale = 4;

ctx.imageSmoothingEnabled = false;
ctx.scale(scale, scale);

function drawFrame(frame) {
  ctx.fillStyle = "#000";
  ctx.fillRect(0, 0, 128, 64);

  if (!frame || !frame.elements) return;

  frame.elements.forEach((el) => {
    if (el.type === "text") {
      ctx.fillStyle = "#fff";
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
    }
  });

  // Detect if "MESSAGE:" is present to update UI state (hacky but reliable for this simple API)
  const isCustom = frame.elements.some((el) => el.value === "MESSAGE:");
  updateModeUI(isCustom);
}

function updateModeUI(isCustom) {
  const badge = document.getElementById("mode-badge");
  if (isCustom) {
    badge.textContent = "● MANUAL OVERRIDE";
    badge.classList.add("custom-active");
  } else {
    badge.textContent = "AUTO MODE";
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
document
  .getElementById("customText")
  .addEventListener("keypress", function (e) {
    if (e.key === "Enter") {
      sendCustom();
    }
  });
