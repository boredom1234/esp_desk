


















const canvas = document.getElementById("preview");
const ctx = canvas.getContext("2d");
const scale = 4;

ctx.imageSmoothingEnabled = false;
ctx.scale(scale, scale);




let autoPlayEnabled = true;
let autoPlayInterval = null;
let frameSpeed = 200;
let espRefreshDuration = 3000;
let gifFps = 0; 
let settings = {};
let marqueeDirection = "left";
let marqueeSize = 2;
let textStyle = "normal";
let lastFrameHash = null; 
let lastUploadedImage = null; 






initDragAndDrop();
initClipboardPaste();
initCharCounters();


function initAfterAuth() {
  loadSettings();
  loadCurrent();
  loadWeather();
  loadTimezone(); 
  loadBCDSettings(); 
  loadAnalogSettings(); 
  loadSpotifyStatus(); 
  initPomodoro(); 

  
  startPolling();
}





let pollingInterval = null;
let settingsPollingInterval = null;
let weatherInterval = null;

function startPolling() {
  
  
  if (pollingInterval) clearInterval(pollingInterval);
  pollingInterval = setInterval(() => {
    loadCurrentWithChangeDetection();
  }, 1500);

  
  if (settingsPollingInterval) clearInterval(settingsPollingInterval);
  settingsPollingInterval = setInterval(() => {
    loadSettings();
  }, 10000);

  
  if (weatherInterval) clearInterval(weatherInterval);
  weatherInterval = setInterval(loadWeather, 60000);
}


function loadCurrentWithChangeDetection() {
  fetch("/frame/current")
    .then((res) => {
      if (!res.ok) {
        if (res.status === 503) {
          
          return null;
        }
        throw new Error(`HTTP ${res.status}`);
      }
      return res.json();
    })
    .then((frame) => {
      if (!frame) return;
      
      const frameHash = JSON.stringify(frame);
      if (frameHash !== lastFrameHash) {
        lastFrameHash = frameHash;
        drawFrame(frame);
        showRefreshIndicator();
      }
    })
    .catch(() => {});
}


function showRefreshIndicator() {
  const badge = document.getElementById("mode-badge");
  if (badge) {
    badge.classList.add("refresh-pulse");
    setTimeout(() => badge.classList.remove("refresh-pulse"), 300);
  }
}






document.getElementById("customText").addEventListener("keypress", (e) => {
  if (e.key === "Enter" && !e.shiftKey) {
    e.preventDefault();
    sendCustomText();
  }
});

document.getElementById("marqueeText").addEventListener("keypress", (e) => {
  if (e.key === "Enter") {
    sendMarquee();
  }
});




function updateGifFpsDisplay(fps) {
  const label = document.getElementById("gifFpsValue");
  if (fps === 0) {
    label.textContent = "Original";
  } else {
    label.textContent = `${fps} FPS`;
  }
}


const saveGifFpsDebounced = debounce((fps) => {
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ gifFps: fps }),
  }).catch(() => {});
}, 300);

function updateGifFps(value) {
  gifFps = parseInt(value);
  updateGifFpsDisplay(gifFps);

  
  saveGifFpsDebounced(gifFps);
}

function resetGifFps() {
  gifFps = 0;
  document.getElementById("gifFpsSlider").value = 0;
  updateGifFpsDisplay(0);

  
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ gifFps: 0 }),
  }).catch(() => {});
}




function loadTimezone() {
  authFetch("/api/settings/timezone")
    .then((res) => res.json())
    .then((data) => {
      if (data.timezone) {
        const select = document.getElementById("timezoneSelect");
        if (select) {
          select.value = data.timezone;
        }
      }
    })
    .catch(() => {});
}

function updateTimezone() {
  const select = document.getElementById("timezoneSelect");
  if (!select) return;

  const timezone = select.value;

  authFetch("/api/settings/timezone", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ timezone: timezone }),
  })
    .then((res) => res.json())
    .then((data) => {
      if (data.status === "updated") {
        
      }
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("Failed to update timezone:", err);
      }
    });
}





checkAuth();
