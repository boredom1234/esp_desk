// ==========================================
// ESP DESK_OS - Display Cycle Management
// ==========================================

let cycleItems = [];
let newItemStyle = "normal";
let cycleItemIdCounter = 0;
let pendingSaveCount = 0;
let lastSaveTimestamp = 0;

// Render cycle items from server data
function renderCycleItems(items, updateLocalState = true) {
  // Only update local state if explicitly requested (from server load)
  if (updateLocalState) {
    cycleItems = items || [];
  }
  const itemsToRender = updateLocalState ? cycleItems : items;
  const list = document.getElementById("displayCycleList");
  list.innerHTML = "";

  itemsToRender.forEach((item, index) => {
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
    // Issue 14: Removed unused safeId variable - DOM manipulation is already XSS-safe

    // Prevent drag interference
    checkbox.addEventListener("mousedown", (e) => {
      e.stopPropagation();
    });

    checkbox.addEventListener("change", (e) => {
      //("Checkbox change event fired for:", item.id);
      toggleCycleItem(item.id);
    });

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
    bcd: "🔢",
    analog: "🧮",
    spotify: "🎵",
    weather: "🌤",
    uptime: "⏱",
    text: "💬",
    image: "🖼",
    pomodoro: "🍅",
    countdown: "⏳",
    qr: "📱",
  };
  return icons[type] || "📋";
}

function truncate(str, len) {
  if (!str) return "";
  return str.length > len ? str.substring(0, len) + "..." : str;
}

// Toggle item enabled state
function toggleCycleItem(id) {
  //("toggleCycleItem called for:", id);
  const item = cycleItems.find((i) => i.id === id);
  if (item) {
    //("Item found, current enabled:", item.enabled);
    item.enabled = !item.enabled;
    //("Item toggled, new enabled:", item.enabled);
    // Update the checkbox visually immediately (without re-render)
    const checkbox = document.querySelector(
      `.cycle-item[data-id="${CSS.escape(id)}"] input[type="checkbox"]`
    );
    if (checkbox) {
      checkbox.checked = item.enabled;
    }
    saveCycleItems();
  } else {
    console.error("Item not found with id:", id);
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

  if (type === "countdown") {
    // Show countdown input panel
    document.getElementById("countdownItemConfig").style.display = "block";
    document.getElementById("countdownLabel").focus();
    return;
  }

  if (type === "qr") {
    // Show QR input panel
    document.getElementById("qrItemConfig").style.display = "block";
    document.getElementById("qrDataInput").focus();
    return;
  }

  // Generate unique ID
  cycleItemIdCounter++;
  const id = `${type}-${Date.now()}-${cycleItemIdCounter}`;

  const labelMap = {
    time: "🕐 Time",
    bcd: "🔢 BCD Clock",
    analog: "🧮 Analog Clock",
    spotify: "🎵 Now Playing",
    weather: "🌤 Weather",
    uptime: "⏱ Uptime",
    image: "🖼 Image",
    pomodoro: "🍅 Pomodoro",
    countdown: "⏳ Countdown",
    qr: "📱 QR Code",
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
  pendingSaveCount++;
  //("=== SAVING CYCLE ITEMS ===");
  //("Pending saves:", pendingSaveCount);
  //("Items to save:", cycleItems);

  // Issue 1: Use authFetch for protected endpoint
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cycleItems: cycleItems }),
  })
    .then((res) => {
      //("Save response status:", res.status);
      return res.json();
    })
    .then((data) => {
      lastSaveTimestamp = Date.now();
      //("✓ Cycle items saved successfully");
      //("Server response:", data);
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("❌ Save failed:", err);
      }
    })
    .finally(() => {
      pendingSaveCount--;
      if (pendingSaveCount < 0) pendingSaveCount = 0;
      //("Save complete, pending now:", pendingSaveCount);
    });
}

// Update from loadSettings
function updateDisplayCycleUI(items) {
  // Don't overwrite local state if we're in the middle of saving
  // Also add a 5-second grace period after last save to ensure we don't fetch stale data
  if (pendingSaveCount > 0 || Date.now() - lastSaveTimestamp < 5000) {
    // //("Skipping UI update - save in progress or recent");
    return;
  }
  renderCycleItems(items, true);
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

  // Re-bind checkbox and delete button event listeners after cloning
  // (cloneNode(true) copies the DOM but not event listeners)
  newList.querySelectorAll(".cycle-item").forEach((item) => {
    const itemId = item.dataset.id;
    const checkbox = item.querySelector('input[type="checkbox"]');
    const deleteBtn = item.querySelector(".cycle-delete-btn");

    if (checkbox) {
      checkbox.addEventListener("mousedown", (e) => e.stopPropagation());
      checkbox.addEventListener("change", () => toggleCycleItem(itemId));
    }
    if (deleteBtn) {
      deleteBtn.addEventListener("click", () => deleteCycleItem(itemId));
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
