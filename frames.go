package main

import (
	"encoding/json"
	"net/http"
)

// ==========================================
// FRAME HANDLERS
// ==========================================

func currentFrame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(frames) == 0 {
		http.Error(w, "No frames available", http.StatusServiceUnavailable)
		return
	}

	// Create a copy of the frame with ESP refresh duration
	frame := frames[index]
	frame.Duration = espRefreshDuration

	w.Header().Set("Content-Type", "application/json")

	// Include isGifMode hint so ESP32 can detect mode change immediately
	response := map[string]interface{}{
		"version":          frame.Version,
		"duration":         frame.Duration,
		"clear":            frame.Clear,
		"elements":         frame.Elements,
		"isGifMode":        isGifMode,        // Hint for ESP32 to fetch /api/gif/full immediately
		"displayRotation":  displayRotation,  // 0 = normal, 2 = 180 degrees
		"ledBrightness":    ledBrightness,    // 0-100 for RGB LED beacon
		"ledBeaconEnabled": ledBeaconEnabled, // Enable/disable satellite beacon
	}
	json.NewEncoder(w).Encode(response)
}

func nextFrame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(frames) == 0 {
		return
	}

	index = (index + 1) % len(frames)

	// Create a copy of the frame with ESP refresh duration
	frame := frames[index]
	frame.Duration = espRefreshDuration

	w.Header().Set("Content-Type", "application/json")

	// Include isGifMode hint so ESP32 can detect mode change immediately
	response := map[string]interface{}{
		"version":          frame.Version,
		"duration":         frame.Duration,
		"clear":            frame.Clear,
		"elements":         frame.Elements,
		"isGifMode":        isGifMode,        // Hint for ESP32 to fetch /api/gif/full immediately
		"displayRotation":  displayRotation,  // 0 = normal, 2 = 180 degrees
		"ledBrightness":    ledBrightness,    // 0-100 for RGB LED beacon
		"ledBeaconEnabled": ledBeaconEnabled, // Enable/disable satellite beacon
	}
	json.NewEncoder(w).Encode(response)
}

func prevFrame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(frames) == 0 {
		return
	}

	index = index - 1
	if index < 0 {
		index = len(frames) - 1
	}

	// Create a copy of the frame with ESP refresh duration
	frame := frames[index]
	frame.Duration = espRefreshDuration

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(frame)
}

func handleFrames(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(frames)
		return
	}

	if r.Method == http.MethodPost {
		nextFrame(w, r)
		return
	}
}
