// ==========================================
// LED EFFECT CONTROLS
// ==========================================

let ledEffectMode = "auto";
let ledCustomColor = "#0064FF";
let ledFlashSpeed = 500;
let ledPulseSpeed = 1000;

// Debounced version for slider inputs
const saveLedSettingsDebounced = debounce(() => {
  saveLedSettings();
}, 300);

/**
 * Set the LED effect mode
 * @param {string} mode - Effect mode: 'auto', 'static', 'flash', 'pulse', 'rainbow'
 */
function setLedEffect(mode) {
  ledEffectMode = mode;

  // Update button states
  document.querySelectorAll(".effect-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.effect === mode);
  });

  // Show/hide color and speed controls based on mode
  const colorRow = document.getElementById("ledColorRow");
  const speedRow = document.getElementById("ledSpeedRow");

  if (colorRow) {
    colorRow.style.display = ["static", "flash", "pulse"].includes(mode)
      ? "flex"
      : "none";
  }
  // Show RGB sliders for modes that use custom color
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

/**
 * Set the custom LED color
 * @param {string} color - Hex color string e.g. '#FF5500'
 */
function setLedColor(color) {
  ledCustomColor = color;
  const picker = document.getElementById("ledColorPicker");
  if (picker) {
    picker.value = color;
  }

  // Update active state on preset buttons
  document.querySelectorAll(".color-preset").forEach((btn) => {
    const btnColor = btn.style.background.toUpperCase();
    const selectedColor = color.toUpperCase();
    btn.classList.toggle("active", btnColor.includes(selectedColor.slice(1)));
  });

  // Sync RGB sliders with the selected color
  syncRgbSlidersFromHex(color);

  saveLedSettings();
}

/**
 * Update LED flash/pulse speed from slider
 * @param {number} value - Speed in milliseconds
 */
function updateLedSpeed(value) {
  const speed = parseInt(value);
  ledFlashSpeed = speed;
  ledPulseSpeed = speed;

  const speedLabel = document.getElementById("ledSpeedValue");
  if (speedLabel) {
    speedLabel.textContent = speed + "ms";
  }

  // Use debounced version for slider inputs
  saveLedSettingsDebounced();
}

/**
 * Save LED settings to backend
 */
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
    console.log(
      "🎨 LED settings saved:",
      ledEffectMode,
      ledCustomColor,
      ledFlashSpeed + "ms"
    );
  } catch (err) {
    console.error("Failed to save LED settings:", err);
  }
}

/**
 * Initialize LED settings from backend data
 * @param {boolean} enabled - LED beacon enabled
 * @param {number} brightness - Brightness (0-100)
 * @param {string} effectMode - Effect mode
 * @param {string} customColor - Hex color
 * @param {number} flashSpeed - Flash interval ms
 * @param {number} pulseSpeed - Pulse cycle ms
 */
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

  // Update effect buttons
  document.querySelectorAll(".effect-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.effect === ledEffectMode);
  });

  // Update color picker
  const colorPicker = document.getElementById("ledColorPicker");
  if (colorPicker) {
    colorPicker.value = ledCustomColor;
  }

  // Update speed slider
  const speedSlider = document.getElementById("ledSpeedSlider");
  const speedLabel = document.getElementById("ledSpeedValue");
  if (speedSlider) {
    speedSlider.value = ledFlashSpeed;
  }
  if (speedLabel) {
    speedLabel.textContent = ledFlashSpeed + "ms";
  }

  // Show/hide appropriate controls
  const colorRow = document.getElementById("ledColorRow");
  const speedRow = document.getElementById("ledSpeedRow");

  if (colorRow) {
    colorRow.style.display = ["static", "flash", "pulse"].includes(
      ledEffectMode
    )
      ? "flex"
      : "none";
  }
  // Show RGB sliders for modes that use custom color
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

  // Sync RGB sliders with the loaded color
  syncRgbSlidersFromHex(ledCustomColor);

  console.log("🎨 LED settings initialized:", ledEffectMode, ledCustomColor);
}

// ==========================================
// RGB SLIDER FUNCTIONS
// ==========================================

/**
 * Sync RGB sliders from hex color
 */
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

/**
 * Update color from RGB sliders
 */
function updateRgbSlider() {
  const r = parseInt(document.getElementById("ledSliderR").value);
  const g = parseInt(document.getElementById("ledSliderG").value);
  const b = parseInt(document.getElementById("ledSliderB").value);

  // Update number inputs
  document.getElementById("ledValueR").value = r;
  document.getElementById("ledValueG").value = g;
  document.getElementById("ledValueB").value = b;

  // Convert to hex and update
  const hexColor = rgbToHex(r, g, b);
  updateColorFromRgb(hexColor);
}

/**
 * Update color from RGB number inputs
 */
function updateRgbValue() {
  let r = parseInt(document.getElementById("ledValueR").value) || 0;
  let g = parseInt(document.getElementById("ledValueG").value) || 0;
  let b = parseInt(document.getElementById("ledValueB").value) || 0;

  // Clamp values
  r = Math.max(0, Math.min(255, r));
  g = Math.max(0, Math.min(255, g));
  b = Math.max(0, Math.min(255, b));

  // Update sliders
  document.getElementById("ledSliderR").value = r;
  document.getElementById("ledSliderG").value = g;
  document.getElementById("ledSliderB").value = b;

  // Convert to hex and update
  const hexColor = rgbToHex(r, g, b);
  updateColorFromRgb(hexColor);
}

/**
 * Helper: RGB to hex conversion
 */
function rgbToHex(r, g, b) {
  return (
    "#" +
    [r, g, b]
      .map((x) => x.toString(16).padStart(2, "0"))
      .join("")
      .toUpperCase()
  );
}

/**
 * Helper: Update UI and save from RGB values
 */
function updateColorFromRgb(hexColor) {
  ledCustomColor = hexColor;

  // Update preview swatch and hex input
  const swatch = document.getElementById("rgbPreviewSwatch");
  const hexInput = document.getElementById("rgbHexInput");
  const picker = document.getElementById("ledColorPicker");

  if (swatch) swatch.style.background = hexColor;
  if (hexInput) hexInput.value = hexColor;
  if (picker) picker.value = hexColor;

  // Clear preset active states (manual value doesn't match presets)
  document.querySelectorAll(".color-preset").forEach((btn) => {
    btn.classList.remove("active");
  });

  // Use debounced version for slider inputs
  saveLedSettingsDebounced();
}

/**
 * Update color from hex input field
 */
function updateFromHexInput() {
  const hexInput = document.getElementById("rgbHexInput");
  if (!hexInput) return;

  let hex = hexInput.value.trim().toUpperCase();

  // Add # if missing
  if (hex.length > 0 && hex[0] !== "#") {
    hex = "#" + hex;
  }

  // Validate hex format
  if (!/^#[0-9A-F]{6}$/i.test(hex)) {
    // Invalid format - restore previous value
    hexInput.value = ledCustomColor;
    return;
  }

  // Update the color
  setLedColor(hex);
}
