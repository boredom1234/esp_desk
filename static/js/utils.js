







 
function debounce(func, wait) {
  let timeout;
  return function (...args) {
    clearTimeout(timeout);
    timeout = setTimeout(() => func.apply(this, args), wait);
  };
}





 
function escapeHtml(str) {
  if (str === null || str === undefined) return "";
  const div = document.createElement("div");
  div.textContent = String(str);
  return div.innerHTML;
}

 
function safeText(text) {
  return document.createTextNode(String(text || ""));
}

 
function safeElement(tag, text, className) {
  const el = document.createElement(tag);
  if (className) el.className = className;
  el.textContent = String(text || "");
  return el;
}
