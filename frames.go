package main

import (
	"encoding/json"
	"net/http"
)





func currentFrame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(frames) == 0 {
		http.Error(w, "No frames available", http.StatusServiceUnavailable)
		return
	}

	
	frame := frames[index]
	frame.Duration = espRefreshDuration

	w.Header().Set("Content-Type", "application/json")

	
	response := map[string]interface{}{
		"version":          frame.Version,
		"duration":         frame.Duration,
		"clear":            frame.Clear,
		"elements":         frame.Elements,
		"isGifMode":        isGifMode,        
		"displayRotation":  displayRotation,  
		"ledBrightness":    ledBrightness,    
		"ledBeaconEnabled": ledBeaconEnabled, 
		"ledEffectMode":    ledEffectMode,    
		"ledCustomColor":   ledCustomColor,   
		"ledFlashSpeed":    ledFlashSpeed,    
		"ledPulseSpeed":    ledPulseSpeed,    
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

	
	frame := frames[index]
	frame.Duration = espRefreshDuration

	w.Header().Set("Content-Type", "application/json")

	
	response := map[string]interface{}{
		"version":          frame.Version,
		"duration":         frame.Duration,
		"clear":            frame.Clear,
		"elements":         frame.Elements,
		"isGifMode":        isGifMode,        
		"displayRotation":  displayRotation,  
		"ledBrightness":    ledBrightness,    
		"ledBeaconEnabled": ledBeaconEnabled, 
		"ledEffectMode":    ledEffectMode,    
		"ledCustomColor":   ledCustomColor,   
		"ledFlashSpeed":    ledFlashSpeed,    
		"ledPulseSpeed":    ledPulseSpeed,    
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
