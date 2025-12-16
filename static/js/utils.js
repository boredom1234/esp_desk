// ==========================================
// ESP DESK_OS - XSS Sanitization Utilities
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
