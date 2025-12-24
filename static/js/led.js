



let ledEffectMode = "auto";
let ledCustomColor = "#0064FF";
let ledFlashSpeed = 500;
let ledPulseSpeed = 1000;



const safeDebounce =
  typeof debounce === "function"
    ? debounce
    : (func, wait) => {
        let timeout;
        return function (...args) {
          clearTimeout(timeout);
          timeout = setTimeout(() => func.apply(this, args), wait);
        };
      };

const saveLedSettingsDebounced = safeDebounce(() => {
  saveLedSettings();
}, 300);

 
function setLedEffect(mode) {
  ledEffectMode = mode;

  
  document.querySelectorAll(".effect-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.effect === mode);
  });

  
  const colorRow = document.getElementById("ledColorRow");
  const speedRow = document.getElementById("ledSpeedRow");

  if (colorRow) {
    colorRow.style.display = ["static", "flash", "pulse"].includes(mode)
      ? "flex"
      : "none";
  }
  
  const rgbSliders = document.getElementById("ledRgbSliders");
  if (rgbSliders) {
    rgbSliders.style.display = ["static", "flash", "pulse"].includes(mode)
      ? "block"
      : "none";
  }
  if (speedRow) {
    speedRow.style.display = ["flash", "pulse"].includes(mode)
      ? "flex"
      : "none";
  }

  saveLedSettings();
}

 
function setLedColor(color) {
  ledCustomColor = color;
  const picker = document.getElementById("ledColorPicker");
  if (picker) {
    picker.value = color;
  }

  
  document.querySelectorAll(".color-preset").forEach((btn) => {
    const btnColor = btn.style.background.toUpperCase();
    const selectedColor = color.toUpperCase();
    btn.classList.toggle("active", btnColor.includes(selectedColor.slice(1)));
  });

  
  syncRgbSlidersFromHex(color);

  saveLedSettings();
}

 
function updateLedSpeed(value) {
  const speed = parseInt(value);
  ledFlashSpeed = speed;
  ledPulseSpeed = speed;

  const speedLabel = document.getElementById("ledSpeedValue");
  if (speedLabel) {
    speedLabel.textContent = speed + "ms";
  }

  
  saveLedSettingsDebounced();
}

 
async function saveLedSettings() {
  try {
    await authFetch("/api/settings", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        ledEffectMode: ledEffectMode,
        ledCustomColor: ledCustomColor,
        ledFlashSpeed: ledFlashSpeed,
        ledPulseSpeed: ledPulseSpeed,
      }),
    });
     
  } catch (err) {
    console.error("Failed to save LED settings:", err);
  }
}

 
function initLedSettings(
  enabled,
  brightness,
  effectMode,
  customColor,
  flashSpeed,
  pulseSpeed
) {
  ledEffectMode = effectMode || "auto";
  ledCustomColor = customColor || "#0064FF";
  ledFlashSpeed = flashSpeed || 500;
  ledPulseSpeed = pulseSpeed || 1000;

  
  document.querySelectorAll(".effect-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.effect === ledEffectMode);
  });

  
  const colorPicker = document.getElementById("ledColorPicker");
  if (colorPicker) {
    colorPicker.value = ledCustomColor;
  }

  
  const speedSlider = document.getElementById("ledSpeedSlider");
  const speedLabel = document.getElementById("ledSpeedValue");
  if (speedSlider) {
    speedSlider.value = ledFlashSpeed;
  }
  if (speedLabel) {
    speedLabel.textContent = ledFlashSpeed + "ms";
  }

  
  const colorRow = document.getElementById("ledColorRow");
  const speedRow = document.getElementById("ledSpeedRow");

  if (colorRow) {
    colorRow.style.display = ["static", "flash", "pulse"].includes(
      ledEffectMode
    )
      ? "flex"
      : "none";
  }
  
  const rgbSliders = document.getElementById("ledRgbSliders");
  if (rgbSliders) {
    rgbSliders.style.display = ["static", "flash", "pulse"].includes(
      ledEffectMode
    )
      ? "block"
      : "none";
  }
  if (speedRow) {
    speedRow.style.display = ["flash", "pulse"].includes(ledEffectMode)
      ? "flex"
      : "none";
  }

  
  syncRgbSlidersFromHex(ledCustomColor);

  
}





 
function syncRgbSlidersFromHex(hexColor) {
  if (!hexColor || hexColor.length !== 7) return;

  const r = parseInt(hexColor.slice(1, 3), 16);
  const g = parseInt(hexColor.slice(3, 5), 16);
  const b = parseInt(hexColor.slice(5, 7), 16);

  const sliderR = document.getElementById("ledSliderR");
  const sliderG = document.getElementById("ledSliderG");
  const sliderB = document.getElementById("ledSliderB");
  const valueR = document.getElementById("ledValueR");
  const valueG = document.getElementById("ledValueG");
  const valueB = document.getElementById("ledValueB");
  const swatch = document.getElementById("rgbPreviewSwatch");
  const hexInput = document.getElementById("rgbHexInput");

  if (sliderR) sliderR.value = r;
  if (sliderG) sliderG.value = g;
  if (sliderB) sliderB.value = b;
  if (valueR) valueR.value = r;
  if (valueG) valueG.value = g;
  if (valueB) valueB.value = b;
  if (swatch) swatch.style.background = hexColor;
  if (hexInput) hexInput.value = hexColor.toUpperCase();
}

 
function updateRgbSlider() {
  const r = parseInt(document.getElementById("ledSliderR").value);
  const g = parseInt(document.getElementById("ledSliderG").value);
  const b = parseInt(document.getElementById("ledSliderB").value);

  
  document.getElementById("ledValueR").value = r;
  document.getElementById("ledValueG").value = g;
  document.getElementById("ledValueB").value = b;

  
  const hexColor = rgbToHex(r, g, b);
  updateColorFromRgb(hexColor);
}

 
function updateRgbValue() {
  let r = parseInt(document.getElementById("ledValueR").value) || 0;
  let g = parseInt(document.getElementById("ledValueG").value) || 0;
  let b = parseInt(document.getElementById("ledValueB").value) || 0;

  
  r = Math.max(0, Math.min(255, r));
  g = Math.max(0, Math.min(255, g));
  b = Math.max(0, Math.min(255, b));

  
  document.getElementById("ledSliderR").value = r;
  document.getElementById("ledSliderG").value = g;
  document.getElementById("ledSliderB").value = b;

  
  const hexColor = rgbToHex(r, g, b);
  updateColorFromRgb(hexColor);
}

 
function rgbToHex(r, g, b) {
  return (
    "#" +
    [r, g, b]
      .map((x) => x.toString(16).padStart(2, "0"))
      .join("")
      .toUpperCase()
  );
}

 
function updateColorFromRgb(hexColor) {
  ledCustomColor = hexColor;

  
  const swatch = document.getElementById("rgbPreviewSwatch");
  const hexInput = document.getElementById("rgbHexInput");
  const picker = document.getElementById("ledColorPicker");

  if (swatch) swatch.style.background = hexColor;
  if (hexInput) hexInput.value = hexColor;
  if (picker) picker.value = hexColor;

  
  document.querySelectorAll(".color-preset").forEach((btn) => {
    btn.classList.remove("active");
  });

  
  saveLedSettingsDebounced();
}

 
function updateFromHexInput() {
  const hexInput = document.getElementById("rgbHexInput");
  if (!hexInput) return;

  let hex = hexInput.value.trim().toUpperCase();

  
  if (hex.length > 0 && hex[0] !== "#") {
    hex = "#" + hex;
  }

  
  if (!/^#[0-9A-F]{6}$/i.test(hex)) {
    
    hexInput.value = ledCustomColor;
    return;
  }

  
  setLedColor(hex);
}
