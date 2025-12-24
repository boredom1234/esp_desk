



function toggleAutoPlay() {
  if (autoPlayEnabled) {
    stopAutoPlay();
  } else {
    startAutoPlay();
  }
}

function startAutoPlay() {
  autoPlayEnabled = true;
  updateAutoPlayButton();

  if (autoPlayInterval) clearInterval(autoPlayInterval);
  autoPlayInterval = setInterval(() => {
    nextFrame();
  }, frameSpeed);
}

function stopAutoPlay() {
  autoPlayEnabled = false;
  updateAutoPlayButton();

  if (autoPlayInterval) {
    clearInterval(autoPlayInterval);
    autoPlayInterval = null;
  }
}

function updateAutoPlayButton() {
  const btn = document.getElementById("autoPlayBtn");
  if (autoPlayEnabled) {
    btn.textContent = "⏸ Pause";
    btn.classList.add("playing");
  } else {
    btn.textContent = "▶ Play";
    btn.classList.remove("playing");
  }
}


const saveSpeedDebounced = debounce((speed) => {
  authFetch("/api/settings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ frameDuration: speed }),
  }).catch((err) => {
    if (err.message !== "Unauthorized") {
      console.error("updateSpeed error:", err);
    }
  });
}, 300);

function updateSpeed(value) {
  frameSpeed = parseInt(value);
  document.getElementById("speedValue").textContent = `${frameSpeed}ms`;

  if (autoPlayEnabled) {
    startAutoPlay(); 
  }

  
  saveSpeedDebounced(frameSpeed);
}
