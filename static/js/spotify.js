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
  const envNotice = document.getElementById("spotifyEnvNotice");

  if (!statusEl) return;

  // Show/hide env notice if credentials are from environment variables
  if (envNotice) {
    envNotice.style.display = spotifyStatus.credsFromEnv ? "block" : "none";
  }

  if (spotifyStatus.isConnected) {
    statusEl.textContent = "✅ Connected";
    statusEl.className = "spotify-status connected";
    if (connectBtn) connectBtn.style.display = "none";
    if (disconnectBtn) disconnectBtn.style.display = "inline-block";
  } else if (spotifyStatus.hasCredentials) {
    statusEl.textContent = "🔗 Ready to connect";
    statusEl.className = "spotify-status ready";
    if (connectBtn) connectBtn.style.display = "inline-block";
    if (disconnectBtn) disconnectBtn.style.display = "none";
  } else {
    statusEl.textContent = "⚙️ Keys Missing (Check Env Vars)";
    statusEl.className = "spotify-status";
    if (connectBtn) connectBtn.style.display = "none";
    if (disconnectBtn) disconnectBtn.style.display = "none";
  }
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
