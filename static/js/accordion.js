



 
function toggleAccordion(section) {
  const item = document.querySelector(
    `.accordion-item[data-section="${section}"]`
  );
  if (item) {
    item.classList.toggle("expanded");
  }
}

 
function expandAccordion(section) {
  const item = document.querySelector(
    `.accordion-item[data-section="${section}"]`
  );
  if (item) {
    item.classList.add("expanded");
  }
}

 
function collapseAccordion(section) {
  const item = document.querySelector(
    `.accordion-item[data-section="${section}"]`
  );
  if (item) {
    item.classList.remove("expanded");
  }
}

 
function collapseAllAccordions() {
  document.querySelectorAll(".accordion-item").forEach((item) => {
    item.classList.remove("expanded");
  });
}

 
function expandAllAccordions() {
  document.querySelectorAll(".accordion-item").forEach((item) => {
    item.classList.add("expanded");
  });
}
