let cycleItems = [];
let newItemStyle = "normal";
let cycleItemIdCounter = 0;
let pendingSaveCount = 0;
let lastSaveTimestamp = 0;

function renderCycleItems(items, updateLocalState = true) {
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

    const handleSpan = document.createElement("span");
    handleSpan.className = "cycle-handle";
    handleSpan.textContent = "⋮⋮";

    const checkboxLabel = document.createElement("label");
    checkboxLabel.className = "cycle-checkbox";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.checked = item.enabled;

    checkbox.addEventListener("mousedown", (e) => {
      e.stopPropagation();
    });

    checkbox.addEventListener("change", (e) => {
      toggleCycleItem(item.id);
    });

    const checkmark = document.createElement("span");
    checkmark.className = "checkmark";

    checkboxLabel.appendChild(checkbox);
    checkboxLabel.appendChild(checkmark);

    const labelSpan = document.createElement("span");
    labelSpan.className = "cycle-label";
    labelSpan.textContent = `${typeIcon} ${labelText}${extraInfo}`;

    const deleteBtn = document.createElement("button");
    deleteBtn.className = "cycle-delete-btn";
    deleteBtn.title = "Remove";
    deleteBtn.textContent = "✕";
    deleteBtn.addEventListener("click", () => deleteCycleItem(item.id));

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
    moonphase: "🌙",
    wordclock: "🕰️",
  };
  return icons[type] || "📋";
}

function truncate(str, len) {
  if (!str) return "";
  return str.length > len ? str.substring(0, len) + "..." : str;
}

function toggleCycleItem(id) {
  const item = cycleItems.find((i) => i.id === id);
  if (item) {
    item.enabled = !item.enabled;

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

function deleteCycleItem(id) {
  cycleItems = cycleItems.filter((i) => i.id !== id);
  if (cycleItems.length === 0) {
    alert("You need at least one item in the cycle!");

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

function addCycleItem() {
  const type = document.getElementById("addItemType").value;

  if (type === "text") {
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
    document.getElementById("countdownItemConfig").style.display = "block";
    document.getElementById("countdownLabel").focus();
    return;
  }

  if (type === "qr") {
    document.getElementById("qrItemConfig").style.display = "block";
    document.getElementById("qrDataInput").focus();
    return;
  }

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
    moonphase: "🌙 Moon Phase",
    wordclock: "🕰️ Word Clock",
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

function setNewItemStyle(style) {
  newItemStyle = style;
  document.querySelectorAll(".style-mini-btn").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.style === style);
  });
}

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

  document.getElementById("newItemText").value = "";
  document.getElementById("textItemConfig").style.display = "none";
}

function saveCycleItems() {
  pendingSaveCount++;

  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cycleItems: cycleItems }),
  })
    .then((res) => {
      return res.json();
    })
    .then((data) => {
      lastSaveTimestamp = Date.now();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("❌ Save failed:", err);
      }
    })
    .finally(() => {
      pendingSaveCount--;
      if (pendingSaveCount < 0) pendingSaveCount = 0;
    });
}

function updateDisplayCycleUI(items) {
  if (pendingSaveCount > 0 || Date.now() - lastSaveTimestamp < 5000) {
    return;
  }
  renderCycleItems(items, true);
}

function initDisplayCycleDragDrop() {
  const list = document.getElementById("displayCycleList");
  let draggedItem = null;
  let draggedIndex = -1;

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

      const items = Array.from(newList.querySelectorAll(".cycle-item"));
      const newOrder = items.map((el) => el.dataset.id);

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

function updateDisplayCycle() {
  saveCycleItems();
}

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

  document.getElementById("saveToCycleBtn").style.display = "none";
  lastUploadedImage = null;

  setUploadStatus("success", "Saved to cycle!");
}
