



let pomodoroSession = {
  active: false,
  mode: "work",
  timeRemaining: 25 * 60,
  isPaused: false,
  cyclesCompleted: 0,
};

let pomodoroSettings = {
  workDuration: 25 * 60,
  breakDuration: 5 * 60,
  longBreak: 15 * 60,
  cyclesUntilLong: 4,
  showInCycle: false,
};

let pomodoroPollingInterval = null;
let pomodoroSlidersEditing = false; 
let pomodoroLastSliderChange = 0; 
const POMODORO_EDIT_GRACE_PERIOD = 3000; 





function initPomodoro() {
  loadPomodoroState();
  
  if (pomodoroPollingInterval) clearInterval(pomodoroPollingInterval);
  pomodoroPollingInterval = setInterval(loadPomodoroState, 1000);

  
  setupSliderTracking();
}

function setupSliderTracking() {
  const sliders = [
    "workDuration",
    "breakDuration",
    "longBreakDuration",
    "cycleCount",
  ];
  sliders.forEach((id) => {
    const el = document.getElementById(id);
    if (el) {
      el.addEventListener("focus", () => {
        pomodoroSlidersEditing = true;
      });
      el.addEventListener("blur", () => {
        pomodoroSlidersEditing = false;
      });
      el.addEventListener("input", () => {
        pomodoroLastSliderChange = Date.now();
      });
      el.addEventListener("mousedown", () => {
        pomodoroSlidersEditing = true;
      });
      el.addEventListener("mouseup", () => {
        pomodoroSlidersEditing = false;
      });
      el.addEventListener("touchstart", () => {
        pomodoroSlidersEditing = true;
      });
      el.addEventListener("touchend", () => {
        pomodoroSlidersEditing = false;
      });
    }
  });
}





function loadPomodoroState() {
  authFetch("/api/pomodoro")
    .then((res) => res.json())
    .then((data) => {
      if (data.session) {
        pomodoroSession = data.session;
      }
      if (data.settings) {
        pomodoroSettings = data.settings;
      }
      renderPomodoroUI();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("Pomodoro load error:", err);
      }
    });
}

function pomodoroAction(action) {
  authFetch("/api/pomodoro", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ action: action }),
  })
    .then((res) => res.json())
    .then((data) => {
      if (data.session) {
        pomodoroSession = data.session;
      }
      if (data.settings) {
        pomodoroSettings = data.settings;
      }
      renderPomodoroUI();
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("Pomodoro action error:", err);
      }
    });
}

function savePomodoroSettings() {
  const workMin = parseInt(document.getElementById("workDuration").value);
  const breakMin = parseInt(document.getElementById("breakDuration").value);
  const longMin = parseInt(document.getElementById("longBreakDuration").value);
  const cycles = parseInt(document.getElementById("cycleCount").value);
  const showInCycle = document.getElementById("pomodoroInCycle").checked;

  authFetch("/api/pomodoro", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      workDuration: workMin * 60,
      breakDuration: breakMin * 60,
      longBreak: longMin * 60,
      cyclesUntilLong: cycles,
      showInCycle: showInCycle,
    }),
  })
    .then((res) => res.json())
    .then((data) => {
      if (data.settings) {
        pomodoroSettings = data.settings;
      }
      if (data.session) {
        pomodoroSession = data.session;
      }
      renderPomodoroUI();
      
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("Pomodoro settings save error:", err);
      }
    });
}





function startPomodoro() {
  pomodoroAction("start");
}

function pausePomodoro() {
  pomodoroAction("pause");
}

function resumePomodoro() {
  pomodoroAction("resume");
}

function resetPomodoro() {
  pomodoroAction("reset");
}

function skipPhase() {
  pomodoroAction("skip");
}

function togglePomodoroInCycle() {
  savePomodoroSettings();
}





function renderPomodoroUI() {
  
  const minutes = Math.floor(pomodoroSession.timeRemaining / 60);
  const seconds = pomodoroSession.timeRemaining % 60;
  const timeStr = `${String(minutes).padStart(2, "0")}:${String(
    seconds
  ).padStart(2, "0")}`;

  const timeEl = document.getElementById("pomodoroTime");
  if (timeEl) timeEl.textContent = timeStr;

  
  const modeEl = document.getElementById("pomodoroMode");
  if (modeEl) {
    let modeText = "READY";
    modeEl.classList.remove("break", "long-break");

    if (pomodoroSession.active) {
      switch (pomodoroSession.mode) {
        case "work":
          modeText = "FOCUS";
          break;
        case "break":
          modeText = "BREAK";
          modeEl.classList.add("break");
          break;
        case "longBreak":
          modeText = "LONG BREAK";
          modeEl.classList.add("long-break");
          break;
      }
      if (pomodoroSession.isPaused) {
        modeText += " (PAUSED)";
      }
    } else {
      modeText = "READY";
    }
    modeEl.textContent = modeText;
  }

  
  const cyclesEl = document.getElementById("pomodoroCycles");
  if (cyclesEl) {
    cyclesEl.textContent = `Cycle ${pomodoroSession.cyclesCompleted}/${pomodoroSettings.cyclesUntilLong}`;
  }

  
  const startBtn = document.getElementById("pomodoroStartBtn");
  if (startBtn) {
    if (!pomodoroSession.active) {
      startBtn.textContent = "▶ Start";
      startBtn.onclick = startPomodoro;
      startBtn.classList.remove("playing");
    } else if (pomodoroSession.isPaused) {
      startBtn.textContent = "▶ Resume";
      startBtn.onclick = resumePomodoro;
      startBtn.classList.remove("playing");
    } else {
      startBtn.textContent = "⏸ Pause";
      startBtn.onclick = pausePomodoro;
      startBtn.classList.add("playing");
    }
  }

  
  updateSettingsUI();
}

function updateSettingsUI() {
  
  const isWithinGracePeriod =
    Date.now() - pomodoroLastSliderChange < POMODORO_EDIT_GRACE_PERIOD;
  if (pomodoroSlidersEditing || isWithinGracePeriod) {
    
    const showInCycleEl = document.getElementById("pomodoroInCycle");
    if (showInCycleEl) {
      showInCycleEl.checked = pomodoroSettings.showInCycle;
    }
    return;
  }

  const workEl = document.getElementById("workDuration");
  const breakEl = document.getElementById("breakDuration");
  const longEl = document.getElementById("longBreakDuration");
  const cyclesEl = document.getElementById("cycleCount");
  const showInCycleEl = document.getElementById("pomodoroInCycle");

  if (workEl) {
    workEl.value = Math.round(pomodoroSettings.workDuration / 60);
    document.getElementById("workDurationVal").textContent = workEl.value;
  }
  if (breakEl) {
    breakEl.value = Math.round(pomodoroSettings.breakDuration / 60);
    document.getElementById("breakDurationVal").textContent = breakEl.value;
  }
  if (longEl) {
    longEl.value = Math.round(pomodoroSettings.longBreak / 60);
    document.getElementById("longBreakVal").textContent = longEl.value;
  }
  if (cyclesEl) {
    cyclesEl.value = pomodoroSettings.cyclesUntilLong;
    document.getElementById("cyclesVal").textContent = cyclesEl.value;
  }
  if (showInCycleEl) {
    showInCycleEl.checked = pomodoroSettings.showInCycle;
  }
}

function updateDurationDisplay(type, value) {
  switch (type) {
    case "work":
      document.getElementById("workDurationVal").textContent = value;
      break;
    case "break":
      document.getElementById("breakDurationVal").textContent = value;
      break;
    case "longBreak":
      document.getElementById("longBreakVal").textContent = value;
      break;
    case "cycles":
      document.getElementById("cyclesVal").textContent = value;
      break;
  }
}





function cleanupPomodoro() {
  if (pomodoroPollingInterval) {
    clearInterval(pomodoroPollingInterval);
    pomodoroPollingInterval = null;
  }
}
