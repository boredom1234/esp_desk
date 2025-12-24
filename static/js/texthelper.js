




const CHARS_PER_LINE = 10;

 
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

  
  if (analysis.lines === 1) {
    const count = analysis.charCounts[0];
    counter.textContent = `${count}/${CHARS_PER_LINE} chars`;
    counter.className =
      count > CHARS_PER_LINE ? "char-counter warning" : "char-counter";
  }
  
  else {
    const maxLine = Math.max(...analysis.charCounts);
    const maxLineIndex = analysis.charCounts.indexOf(maxLine) + 1;
    counter.textContent = `Line ${maxLineIndex}: ${maxLine}/${CHARS_PER_LINE} chars`;
    counter.className = analysis.hasOverflow
      ? "char-counter warning"
      : "char-counter";
  }
}

 
function initCharCounters() {
  
  const customText = document.getElementById("customText");
  if (customText) {
    customText.addEventListener("input", () => {
      updateCharCounter("customText", "customTextCounter");
    });
    
    updateCharCounter("customText", "customTextCounter");
  }

  
  const marqueeText = document.getElementById("marqueeText");
  if (marqueeText) {
    marqueeText.addEventListener("input", () => {
      updateCharCounter("marqueeText", "marqueeTextCounter");
    });
    
    updateCharCounter("marqueeText", "marqueeTextCounter");
  }
}
