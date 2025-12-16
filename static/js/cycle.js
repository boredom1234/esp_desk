// ==========================================
// ESP DESK_OS - Display Cycle Management
// ==========================================

let cycleItems = [];
let newItemStyle = "normal";
let cycleItemIdCounter = 0;

// Render cycle items from server data
function renderCycleItems(items) {
  cycleItems = items || [];
  const list = document.getElementById("displayCycleList");
  list.innerHTML = "";

  cycleItems.forEach((item, index) => {
    const div = document.createElement("div");
    div.className = "cycle-item";
    div.dataset.id = item.id;
    div.dataset.index = index;
    div.draggable = true;

    const typeIcon = getTypeIcon(item.type);
    const labelText = item.label || item.type;
    const extraInfo =
      item.type === "text" ? ` "${truncate(item.text, 15)}"` : "";

    // XSS-safe rendering using DOM manipulation instead of innerHTML
    // Handle element
    const handleSpan = document.createElement("span");
    handleSpan.className = "cycle-handle";
    handleSpan.textContent = "⋮⋮";

    // Checkbox label
    const checkboxLabel = document.createElement("label");
    checkboxLabel.className = "cycle-checkbox";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = item.enabled;
    // Use escaped ID to prevent injection in event handler
    const safeId = escapeHtml(item.id);
    checkbox.addEventListener("change", () => toggleCycleItem(item.id));

    const checkmark = document.createElement("span");
    checkmark.className = "checkmark";

    checkboxLabel.appendChild(checkbox);
    checkboxLabel.appendChild(checkmark);

    // Label span - use textContent for safety
    const labelSpan = document.createElement("span");
    labelSpan.className = "cycle-label";
    labelSpan.textContent = `${typeIcon} ${labelText}${extraInfo}`;

    // Delete button
    const deleteBtn = document.createElement("button");
    deleteBtn.className = "cycle-delete-btn";
    deleteBtn.title = "Remove";
    deleteBtn.textContent = "✕";
    deleteBtn.addEventListener("click", () => deleteCycleItem(item.id));

    // Assemble the element
    div.appendChild(handleSpan);
    div.appendChild(checkboxLabel);
    div.appendChild(labelSpan);
    div.appendChild(deleteBtn);

    list.appendChild(div);
  });

  initDisplayCycleDragDrop();
}

function getTypeIcon(type) {
  const icons = {
    time: "🕐",
    weather: "🌤",
    uptime: "⏱",
    text: "💬",
    image: "🖼",
  };
  return icons[type] || "📋";
}

function truncate(str, len) {
  if (!str) return "";
  return str.length > len ? str.substring(0, len) + "..." : str;
}

// Toggle item enabled state
function toggleCycleItem(id) {
  const item = cycleItems.find((i) => i.id === id);
  if (item) {
    item.enabled = !item.enabled;
    saveCycleItems();
  }
}

// Delete item
function deleteCycleItem(id) {
  cycleItems = cycleItems.filter((i) => i.id !== id);
  if (cycleItems.length === 0) {
    alert("You need at least one item in the cycle!");
    // Re-add time as fallback
    cycleItems.push({
      id: "time-fallback",
      type: "time",
      label: "🕐 Time",
      enabled: true,
      duration: 3000,
    });
  }
  saveCycleItems();
  renderCycleItems(cycleItems);
}

// Add new item
function addCycleItem() {
  const type = document.getElementById("addItemType").value;

  if (type === "text") {
    // Show text input panel
    document.getElementById("textItemConfig").style.display = "block";
    document.getElementById("newItemText").focus();
    return;
  }

  if (type === "image") {
    alert(
      "Upload an image first, then use the '💾 Save to Cycle' button that appears after upload."
    );
    return;
  }

  // Generate unique ID
  cycleItemIdCounter++;
  const id = `${type}-${Date.now()}-${cycleItemIdCounter}`;

  const labelMap = {
    time: "🕐 Time",
    weather: "🌤 Weather",
    uptime: "⏱ Uptime",
    image: "🖼 Image",
  };

  const newItem = {
    id: id,
    type: type,
    label: labelMap[type] || type,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);
}

// Set style for new text item
function setNewItemStyle(style) {
  newItemStyle = style;
  document.querySelectorAll(".style-mini-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.style === style);
  });
}

// Confirm adding text item
function confirmAddText() {
  const text = document.getElementById("newItemText").value.trim();
  if (!text) {
    alert("Please enter some text!");
    return;
  }

  cycleItemIdCounter++;
  const id = `text-${Date.now()}-${cycleItemIdCounter}`;

  const newItem = {
    id: id,
    type: "text",
    label: "💬 Message",
    text: text,
    style: newItemStyle,
    size: 2,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);

  // Reset and hide config
  document.getElementById("newItemText").value = "";
  document.getElementById("textItemConfig").style.display = "none";
}

// Save cycle items to server
function saveCycleItems() {
  fetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cycleItems: cycleItems }),
  })
    .then(() => console.log("Cycle items saved:", cycleItems.length))
    .catch((err) => console.error("Save failed:", err));
}

// Update from loadSettings
function updateDisplayCycleUI(items) {
  renderCycleItems(items);
}

// Initialize drag and drop for display cycle items
function initDisplayCycleDragDrop() {
  const list = document.getElementById("displayCycleList");
  let draggedItem = null;
  let draggedIndex = -1;

  // Remove old listeners by cloning
  const newList = list.cloneNode(true);
  list.parentNode.replaceChild(newList, list);

  newList.addEventListener("dragstart", (e) => {
    if (e.target.classList.contains("cycle-item")) {
      draggedItem = e.target;
      draggedIndex = parseInt(e.target.dataset.index);
      e.target.classList.add("dragging");
    }
  });

  newList.addEventListener("dragend", (e) => {
    if (e.target.classList.contains("cycle-item")) {
      e.target.classList.remove("dragging");

      // Get new order from DOM
      const items = Array.from(newList.querySelectorAll(".cycle-item"));
      const newOrder = items.map((el) => el.dataset.id);

      // Reorder cycleItems array
      const reordered = [];
      newOrder.forEach((id) => {
        const item = cycleItems.find((i) => i.id === id);
        if (item) reordered.push(item);
      });
      cycleItems = reordered;

      saveCycleItems();
      draggedItem = null;
      draggedIndex = -1;
    }
  });

  newList.addEventListener("dragover", (e) => {
    e.preventDefault();
    const afterElement = getDragAfterElement(newList, e.clientY);
    if (draggedItem) {
      if (afterElement == null) {
        newList.appendChild(draggedItem);
      } else {
        newList.insertBefore(draggedItem, afterElement);
      }
    }
  });
}

function getDragAfterElement(container, y) {
  const elements = [
    ...container.querySelectorAll(".cycle-item:not(.dragging)"),
  ];

  return elements.reduce(
    (closest, child) => {
      const box = child.getBoundingClientRect();
      const offset = y - box.top - box.height / 2;

      if (offset < 0 && offset > closest.offset) {
        return { offset: offset, element: child };
      } else {
        return closest;
      }
    },
    { offset: Number.NEGATIVE_INFINITY }
  ).element;
}

// Legacy function - now redirects
function updateDisplayCycle() {
  saveCycleItems();
}

// ==========================================
// SAVE IMAGE TO CYCLE
// ==========================================

function saveImageToCycle() {
  if (!lastUploadedImage) {
    alert("No image available to save! Upload an image first.");
    return;
  }

  cycleItemIdCounter++;
  const id = `image-${Date.now()}-${cycleItemIdCounter}`;

  const newItem = {
    id: id,
    type: "image",
    label: "🖼 Image",
    bitmap: lastUploadedImage.bitmap,
    width: lastUploadedImage.width,
    height: lastUploadedImage.height,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);

  // Hide button after saving
  document.getElementById("saveToCycleBtn").style.display = "none";
  lastUploadedImage = null;

  // Show feedback
  setUploadStatus("success", "Saved to cycle!");
}
