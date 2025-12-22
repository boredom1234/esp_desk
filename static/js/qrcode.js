// ==========================================
// ESP DESK_OS - QR Code Management
// ==========================================

// Display QR immediately (bypass cycle)
async function displayQRNow() {
  const data = document.getElementById("qrDataInput").value.trim();
  if (!data) {
    alert("Please enter text or URL to encode!");
    return;
  }

  try {
    const response = await authFetch("/api/qrcode", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ data: data }),
    });

    if (response.ok) {
      //("QR code displayed on ESP32");
      // Optionally hide the config panel
      document.getElementById("qrItemConfig").style.display = "none";
      document.getElementById("qrDataInput").value = "";
    } else {
      const err = await response.json();
      alert("Failed to display QR: " + (err.error || "Unknown error"));
    }
  } catch (e) {
    if (e.message !== "Unauthorized") {
      console.error("QR display error:", e);
      alert("Failed to display QR code");
    }
  }
}

// Save QR to display cycle
function confirmAddQR() {
  const data = document.getElementById("qrDataInput").value.trim();
  if (!data) {
    alert("Please enter text or URL!");
    return;
  }

  cycleItemIdCounter++;
  const id = `qr-${Date.now()}-${cycleItemIdCounter}`;

  const newItem = {
    id: id,
    type: "qr",
    label: "📱 QR Code",
    qrData: data,
    enabled: true,
    duration: 5000, // Longer duration for scanning
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);

  // Reset and hide config
  document.getElementById("qrDataInput").value = "";
  document.getElementById("qrItemConfig").style.display = "none";
}

// Add countdown to display cycle
function confirmAddCountdown() {
  const label = document.getElementById("countdownLabel").value.trim();
  const date = document.getElementById("countdownDate").value;

  if (!label) {
    alert("Please enter an event name!");
    return;
  }
  if (!date) {
    alert("Please select a target date!");
    return;
  }

  cycleItemIdCounter++;
  const id = `countdown-${Date.now()}-${cycleItemIdCounter}`;

  const newItem = {
    id: id,
    type: "countdown",
    label: `⏳ ${label}`,
    targetDate: date,
    targetLabel: label,
    enabled: true,
    duration: 3000,
  };

  cycleItems.push(newItem);
  saveCycleItems();
  renderCycleItems(cycleItems);

  // Reset and hide config
  document.getElementById("countdownLabel").value = "";
  document.getElementById("countdownDate").value = "";
  document.getElementById("countdownItemConfig").style.display = "none";
}
