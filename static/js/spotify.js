// ==========================================
// ESP DESK_OS - Spotify Integration
// ==========================================

let spotifyStatus = {
  enabled: false,
  hasCredentials: false,
  isConnected: false,
  currentTrack: null,
};

// Load Spotify status from server
function loadSpotifyStatus() {
  authFetch("/api/settings/spotify")
    .then((res) => res.json())
    .then((data) => {
      spotifyStatus = data;
      updateSpotifyUI();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("loadSpotifyStatus error:", err);
      }
    });
}

// Update Spotify UI elements
function updateSpotifyUI() {
  const statusEl = document.getElementById("spotifyStatus");
  const connectBtn = document.getElementById("spotifyConnectBtn");
  const disconnectBtn = document.getElementById("spotifyDisconnectBtn");
  const configSection = document.getElementById("spotifyConfigSection");

  if (!statusEl) return;

  if (spotifyStatus.isConnected) {
    statusEl.textContent = "✅ Connected";
    statusEl.className = "spotify-status connected";
    if (connectBtn) connectBtn.style.display = "none";
    if (disconnectBtn) disconnectBtn.style.display = "inline-block";
    if (configSection) configSection.style.display = "none";
  } else if (spotifyStatus.hasCredentials) {
    statusEl.textContent = "🔗 Ready to connect";
    statusEl.className = "spotify-status ready";
    if (connectBtn) connectBtn.style.display = "inline-block";
    if (disconnectBtn) disconnectBtn.style.display = "none";
    if (configSection) configSection.style.display = "block";
  } else {
    statusEl.textContent = "⚙️ Not configured";
    statusEl.className = "spotify-status";
    if (connectBtn) connectBtn.style.display = "none";
    if (disconnectBtn) disconnectBtn.style.display = "none";
    if (configSection) configSection.style.display = "block";
  }
}

// Toggle config section visibility
function toggleSpotifyConfig() {
  const configSection = document.getElementById("spotifyConfigSection");
  if (configSection) {
    configSection.style.display =
      configSection.style.display === "none" ? "block" : "none";
  }
}

// Save Spotify credentials
function saveSpotifyCredentials() {
  const clientId = document.getElementById("spotifyClientId").value.trim();
  const clientSecret = document
    .getElementById("spotifyClientSecret")
    .value.trim();

  if (!clientId || !clientSecret) {
    alert("Please enter both Client ID and Client Secret");
    return;
  }

  authFetch("/api/settings/spotify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ clientId, clientSecret }),
  })
    .then((res) => res.json())
    .then((data) => {
      spotifyStatus = data;
      updateSpotifyUI();
      // Clear the input fields for security
      document.getElementById("spotifyClientSecret").value = "";
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("saveSpotifyCredentials error:", err);
        alert("Failed to save credentials");
      }
    });
}

// Connect to Spotify (opens OAuth flow in new window)
function connectSpotify() {
  // Open auth URL in a popup window
  const width = 500;
  const height = 700;
  const left = screen.width / 2 - width / 2;
  const top = screen.height / 2 - height / 2;

  const popup = window.open(
    "/api/spotify/auth",
    "SpotifyAuth",
    `width=${width},height=${height},left=${left},top=${top},menubar=no,toolbar=no,location=no,status=no`
  );

  // Poll for popup close and refresh status
  const pollTimer = setInterval(() => {
    if (popup.closed) {
      clearInterval(pollTimer);
      loadSpotifyStatus();
    }
  }, 500);
}

// Disconnect from Spotify
function disconnectSpotify() {
  if (!confirm("Disconnect from Spotify?")) return;

  authFetch("/api/settings/spotify", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ disconnect: true }),
  })
    .then((res) => res.json())
    .then((data) => {
      spotifyStatus = data;
      updateSpotifyUI();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("disconnectSpotify error:", err);
      }
    });
}
