// ============================================
// ACCORDION FUNCTIONALITY
// ============================================

/**
 * Toggle an accordion section open/closed
 * @param {string} section - The data-section identifier of the accordion item
 */
function toggleAccordion(section) {
  const item = document.querySelector(
    `.accordion-item[data-section="${section}"]`
  );
  if (item) {
    item.classList.toggle("expanded");
  }
}

/**
 * Expand a specific accordion section
 * @param {string} section - The data-section identifier of the accordion item
 */
function expandAccordion(section) {
  const item = document.querySelector(
    `.accordion-item[data-section="${section}"]`
  );
  if (item) {
    item.classList.add("expanded");
  }
}

/**
 * Collapse a specific accordion section
 * @param {string} section - The data-section identifier of the accordion item
 */
function collapseAccordion(section) {
  const item = document.querySelector(
    `.accordion-item[data-section="${section}"]`
  );
  if (item) {
    item.classList.remove("expanded");
  }
}

/**
 * Collapse all accordion sections
 */
function collapseAllAccordions() {
  document.querySelectorAll(".accordion-item").forEach((item) => {
    item.classList.remove("expanded");
  });
}

/**
 * Expand all accordion sections
 */
function expandAllAccordions() {
  document.querySelectorAll(".accordion-item").forEach((item) => {
    item.classList.add("expanded");
  });
}
