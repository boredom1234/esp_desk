// ==========================================
// ESP DESK_OS - Canvas Rendering
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

      // OLED displays ~10 characters per line at size 1
      // Text automatically wraps on the actual OLED
      const CHARS_PER_LINE = 10;

      if (size === 1) {
        ctx.font = "10px 'JetBrains Mono', monospace";
      } else if (size === 2) {
        ctx.font = "18px 'JetBrains Mono', monospace";
      } else {
        ctx.font = `${size * 8}px 'JetBrains Mono', monospace`;
      }

      ctx.textBaseline = "top";

      // Split text into lines if it exceeds character limit
      // This mimics how the actual OLED display wraps text
      const lines = [];
      let currentLine = "";

      for (let i = 0; i < value.length; i++) {
        currentLine += value[i];
        if (currentLine.length >= CHARS_PER_LINE || i === value.length - 1) {
          lines.push(currentLine);
          currentLine = "";
        }
      }

      // Draw each line with proper vertical spacing
      const lineHeight = size * 8 + 2; // Approximate line height
      lines.forEach((line, index) => {
        ctx.fillText(line, x, y + index * lineHeight);
      });
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

  updateModeUI(settings && settings.frameCount > 1);
}

function updateModeUI(isCustom) {
  const badge = document.getElementById("mode-badge");
  const frameCount = (settings && settings.frameCount) || 1;
  const currentIndex = (settings && settings.currentIndex) || 0;

  if (isCustom && frameCount > 1) {
    badge.textContent = `ANIM (${frameCount})`;
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
  frameBadge.textContent = `Frame ${currentIndex + 1}/${frameCount}`;
}
