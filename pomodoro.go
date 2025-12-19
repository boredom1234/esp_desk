package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// ==========================================
// POMODORO TIMER HANDLER
// ==========================================

// handlePomodoro manages the Pomodoro timer state and settings
func handlePomodoro(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		// Return current Pomodoro state and settings
		mutex.Lock()
		response := map[string]interface{}{
			"session":  pomodoroSession,
			"settings": pomodoroSettings,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Action string `json:"action"` // "start", "pause", "resume", "reset", "skip"
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		switch req.Action {
		case "start":
			pomodoroSession.Active = true
			pomodoroSession.IsPaused = false
			pomodoroSession.StartedAt = time.Now()
			pomodoroSession.Mode = "work"
			pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
			log.Printf("🍅 Pomodoro started: %d min work session", pomodoroSettings.WorkDuration/60)

		case "pause":
			if pomodoroSession.Active && !pomodoroSession.IsPaused {
				pomodoroSession.IsPaused = true
				pomodoroSession.PausedRemaining = pomodoroSession.TimeRemaining
				log.Printf("🍅 Pomodoro paused: %d seconds remaining", pomodoroSession.TimeRemaining)
			}

		case "resume":
			if pomodoroSession.Active && pomodoroSession.IsPaused {
				pomodoroSession.IsPaused = false
				pomodoroSession.StartedAt = time.Now()
				log.Printf("🍅 Pomodoro resumed: %d seconds remaining", pomodoroSession.TimeRemaining)
			}

		case "reset":
			pomodoroSession.Active = false
			pomodoroSession.IsPaused = false
			pomodoroSession.Mode = "work"
			pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
			pomodoroSession.CyclesCompleted = 0
			log.Printf("🍅 Pomodoro reset")

		case "skip":
			// Skip to next phase
			if pomodoroSession.Mode == "work" {
				pomodoroSession.CyclesCompleted++
				if pomodoroSession.CyclesCompleted >= pomodoroSettings.CyclesUntilLong {
					pomodoroSession.Mode = "longBreak"
					pomodoroSession.TimeRemaining = pomodoroSettings.LongBreak
					pomodoroSession.CyclesCompleted = 0
					log.Printf("🍅 Pomodoro: Long break started (%d min)", pomodoroSettings.LongBreak/60)
				} else {
					pomodoroSession.Mode = "break"
					pomodoroSession.TimeRemaining = pomodoroSettings.BreakDuration
					log.Printf("🍅 Pomodoro: Break started (%d min), cycle %d/%d",
						pomodoroSettings.BreakDuration/60, pomodoroSession.CyclesCompleted, pomodoroSettings.CyclesUntilLong)
				}
			} else {
				pomodoroSession.Mode = "work"
				pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
				log.Printf("🍅 Pomodoro: Work session started (%d min)", pomodoroSettings.WorkDuration/60)
			}
			pomodoroSession.StartedAt = time.Now()
			pomodoroSession.IsPaused = false

		default:
			mutex.Unlock()
			jsonError(w, "Invalid action: "+req.Action, http.StatusBadRequest)
			return
		}
		response := map[string]interface{}{
			"session":  pomodoroSession,
			"settings": pomodoroSettings,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPut {
		// Update Pomodoro settings
		var req struct {
			WorkDuration    *int  `json:"workDuration"`
			BreakDuration   *int  `json:"breakDuration"`
			LongBreak       *int  `json:"longBreak"`
			CyclesUntilLong *int  `json:"cyclesUntilLong"`
			ShowInCycle     *bool `json:"showInCycle"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		if req.WorkDuration != nil && *req.WorkDuration >= 60 && *req.WorkDuration <= 3600 {
			pomodoroSettings.WorkDuration = *req.WorkDuration
		}
		if req.BreakDuration != nil && *req.BreakDuration >= 60 && *req.BreakDuration <= 1800 {
			pomodoroSettings.BreakDuration = *req.BreakDuration
		}
		if req.LongBreak != nil && *req.LongBreak >= 300 && *req.LongBreak <= 2700 {
			pomodoroSettings.LongBreak = *req.LongBreak
		}
		if req.CyclesUntilLong != nil && *req.CyclesUntilLong >= 2 && *req.CyclesUntilLong <= 8 {
			pomodoroSettings.CyclesUntilLong = *req.CyclesUntilLong
		}
		if req.ShowInCycle != nil {
			pomodoroSettings.ShowInCycle = *req.ShowInCycle
		}

		// Update session time if not active
		if !pomodoroSession.Active {
			pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
		}

		response := map[string]interface{}{
			"session":  pomodoroSession,
			"settings": pomodoroSettings,
		}
		mutex.Unlock()

		go saveConfig()
		log.Printf("🍅 Pomodoro settings updated: work=%dmin, break=%dmin, long=%dmin, cycles=%d",
			pomodoroSettings.WorkDuration/60, pomodoroSettings.BreakDuration/60,
			pomodoroSettings.LongBreak/60, pomodoroSettings.CyclesUntilLong)

		json.NewEncoder(w).Encode(response)
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}
