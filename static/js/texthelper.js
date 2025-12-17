// ==========================================
// ESP DESK_OS - Text Helper Utilities
// ==========================================

// Character limit per line on 1.3" OLED display (128x64)
const CHARS_PER_LINE = 10;

/**
 * Count characters and lines in text input
 * @param {string} text - Input text
 * @returns {object} - { lines: number, charCounts: number[], maxChars: number, hasOverflow: boolean }
 */
function analyzeText(text) {
  if (!text) {
    return { lines: 0, charCounts: [], maxChars: 0, hasOverflow: false };
  }

  const lines = text.split("\n");
  const charCounts = lines.map((line) => line.length);
  const maxChars = Math.max(...charCounts, 0);
  const hasOverflow = charCounts.some((count) => count > CHARS_PER_LINE);

  return {
    lines: lines.length,
    charCounts,
    maxChars,
    hasOverflow,
  };
}

/**
 * Update character counter display
 * @param {string} inputId - ID of the input element
 * @param {string} counterId - ID of the counter display element
 */
function updateCharCounter(inputId, counterId) {
  const input = document.getElementById(inputId);
  const counter = document.getElementById(counterId);

  if (!input || !counter) return;

  const text = input.value;
  const analysis = analyzeText(text);

  if (!text) {
    counter.textContent = `0/${CHARS_PER_LINE} chars per line`;
    counter.className = "char-counter";
    return;
  }

  // For single line
  if (analysis.lines === 1) {
    const count = analysis.charCounts[0];
    counter.textContent = `${count}/${CHARS_PER_LINE} chars`;
    counter.className =
      count > CHARS_PER_LINE ? "char-counter warning" : "char-counter";
  }
  // For multiple lines
  else {
    const maxLine = Math.max(...analysis.charCounts);
    const maxLineIndex = analysis.charCounts.indexOf(maxLine) + 1;
    counter.textContent = `Line ${maxLineIndex}: ${maxLine}/${CHARS_PER_LINE} chars`;
    counter.className = analysis.hasOverflow
      ? "char-counter warning"
      : "char-counter";
  }
}

/**
 * Initialize character counters for text inputs
 */
function initCharCounters() {
  // Quick Text counter
  const customText = document.getElementById("customText");
  if (customText) {
    customText.addEventListener("input", () => {
      updateCharCounter("customText", "customTextCounter");
    });
    // Initial update
    updateCharCounter("customText", "customTextCounter");
  }

  // Marquee Text counter
  const marqueeText = document.getElementById("marqueeText");
  if (marqueeText) {
    marqueeText.addEventListener("input", () => {
      updateCharCounter("marqueeText", "marqueeTextCounter");
    });
    // Initial update
    updateCharCounter("marqueeText", "marqueeTextCounter");
  }
}
