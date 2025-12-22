package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ==========================================
// SETTINGS HANDLERS
// ==========================================

func handleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		settings := Settings{
			AutoPlay:           autoPlay,
			FrameDuration:      frameDuration,
			EspRefreshDuration: espRefreshDuration,
			GifFps:             gifFps,
			ShowHeaders:        showHeaders,
			DisplayRotation:    displayRotation,
			FrameCount:         len(frames),
			CurrentIndex:       index,
			CycleItems:         cycleItems,
			LedBrightness:      ledBrightness,
			LedBeaconEnabled:   ledBeaconEnabled,
			LedEffectMode:      ledEffectMode,
			LedCustomColor:     ledCustomColor,
			LedFlashSpeed:      ledFlashSpeed,
			LedPulseSpeed:      ledPulseSpeed,
			DisplayScale:       displayScale,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(settings)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			AutoPlay           *bool       `json:"autoPlay,omitempty"`
			FrameDuration      *int        `json:"frameDuration,omitempty"`
			EspRefreshDuration *int        `json:"espRefreshDuration,omitempty"`
			GifFps             *int        `json:"gifFps,omitempty"`
			ShowHeaders        *bool       `json:"showHeaders,omitempty"`
			DisplayRotation    *int        `json:"displayRotation,omitempty"`
			CycleItems         []CycleItem `json:"cycleItems,omitempty"`
			LedBrightness      *int        `json:"ledBrightness,omitempty"`
			LedBeaconEnabled   *bool       `json:"ledBeaconEnabled,omitempty"`
			LedEffectMode      *string     `json:"ledEffectMode,omitempty"`
			LedCustomColor     *string     `json:"ledCustomColor,omitempty"`
			LedFlashSpeed      *int        `json:"ledFlashSpeed,omitempty"`
			LedPulseSpeed      *int        `json:"ledPulseSpeed,omitempty"`
			DisplayScale       *string     `json:"displayScale,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		mutex.Lock()
		var changes []string
		if req.AutoPlay != nil {
			autoPlay = *req.AutoPlay
			changes = append(changes, fmt.Sprintf("autoPlay=%v", autoPlay))
		}
		if req.FrameDuration != nil {
			frameDuration = *req.FrameDuration
			if frameDuration < 50 {
				frameDuration = 50
			}
			if frameDuration > 5000 {
				frameDuration = 5000
			}
			changes = append(changes, fmt.Sprintf("frameDuration=%dms", frameDuration))
		}
		if req.EspRefreshDuration != nil {
			espRefreshDuration = *req.EspRefreshDuration
			if espRefreshDuration < 500 {
				espRefreshDuration = 500
			}
			if espRefreshDuration > 30000 {
				espRefreshDuration = 30000
			}
			changes = append(changes, fmt.Sprintf("espRefreshDuration=%dms", espRefreshDuration))
		}
		if req.GifFps != nil {
			gifFps = *req.GifFps
			if gifFps < 0 {
				gifFps = 0
			}
			if gifFps > 30 {
				gifFps = 30
			}
			changes = append(changes, fmt.Sprintf("gifFps=%d", gifFps))
		}
		if req.ShowHeaders != nil {
			showHeaders = *req.ShowHeaders
			changes = append(changes, fmt.Sprintf("showHeaders=%v", showHeaders))
		}
		if req.CycleItems != nil {
			// Replace entire cycle items list
			cycleItems = req.CycleItems
			changes = append(changes, fmt.Sprintf("cycleItems=%d items", len(cycleItems)))
		}
		if req.DisplayRotation != nil {
			if *req.DisplayRotation == 0 || *req.DisplayRotation == 2 {
				displayRotation = *req.DisplayRotation
				changes = append(changes, fmt.Sprintf("displayRotation=%d", displayRotation))
			}
		}
		if req.LedBrightness != nil {
			ledBrightness = *req.LedBrightness
			if ledBrightness < 0 {
				ledBrightness = 0
			}
			if ledBrightness > 100 {
				ledBrightness = 100
			}
			changes = append(changes, fmt.Sprintf("ledBrightness=%d%%", ledBrightness))
		}
		if req.LedBeaconEnabled != nil {
			ledBeaconEnabled = *req.LedBeaconEnabled
			changes = append(changes, fmt.Sprintf("ledBeaconEnabled=%v", ledBeaconEnabled))
		}
		if req.LedEffectMode != nil {
			// Validate effect mode
			validModes := map[string]bool{"auto": true, "static": true, "flash": true, "pulse": true, "rainbow": true}
			if validModes[*req.LedEffectMode] {
				ledEffectMode = *req.LedEffectMode
				changes = append(changes, fmt.Sprintf("ledEffectMode=%s", ledEffectMode))
			}
		}
		if req.LedCustomColor != nil {
			// Basic hex color validation
			if len(*req.LedCustomColor) == 7 && (*req.LedCustomColor)[0] == '#' {
				ledCustomColor = *req.LedCustomColor
				changes = append(changes, fmt.Sprintf("ledCustomColor=%s", ledCustomColor))
			}
		}
		if req.LedFlashSpeed != nil {
			ledFlashSpeed = *req.LedFlashSpeed
			if ledFlashSpeed < 100 {
				ledFlashSpeed = 100
			}
			if ledFlashSpeed > 2000 {
				ledFlashSpeed = 2000
			}
			changes = append(changes, fmt.Sprintf("ledFlashSpeed=%dms", ledFlashSpeed))
		}
		if req.LedPulseSpeed != nil {
			ledPulseSpeed = *req.LedPulseSpeed
			if ledPulseSpeed < 500 {
				ledPulseSpeed = 500
			}
			if ledPulseSpeed > 3000 {
				ledPulseSpeed = 3000
			}
			changes = append(changes, fmt.Sprintf("ledPulseSpeed=%dms", ledPulseSpeed))
		}
		if req.DisplayScale != nil {
			// Validate scale value
			validScales := map[string]bool{"compact": true, "normal": true, "large": true}
			if validScales[*req.DisplayScale] {
				displayScale = *req.DisplayScale
				changes = append(changes, fmt.Sprintf("displayScale=%s", displayScale))
			}
		}
		settings := Settings{
			AutoPlay:           autoPlay,
			FrameDuration:      frameDuration,
			EspRefreshDuration: espRefreshDuration,
			GifFps:             gifFps,
			ShowHeaders:        showHeaders,
			DisplayRotation:    displayRotation,
			FrameCount:         len(frames),
			CurrentIndex:       index,
			CycleItems:         cycleItems,
			LedBrightness:      ledBrightness,
			LedBeaconEnabled:   ledBeaconEnabled,
			LedEffectMode:      ledEffectMode,
			LedCustomColor:     ledCustomColor,
			LedFlashSpeed:      ledFlashSpeed,
			LedPulseSpeed:      ledPulseSpeed,
			DisplayScale:       displayScale,
		}
		mutex.Unlock()

		// Persist settings (Issue 2)
		go saveConfig()

		// Log what was updated
		if len(changes) > 0 {
			log.Printf("⚙️  Settings updated: %s", strings.Join(changes, ", "))
		}

		json.NewEncoder(w).Encode(settings)
		return
	}
}

func handleToggleHeaders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	showHeaders = !showHeaders
	currentState := showHeaders
	mutex.Unlock()

	log.Printf("👁️  Headers visibility toggled: showHeaders=%v", currentState)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"headersVisible": currentState})
}

func handleGetHeadersState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	currentState := showHeaders
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"headersVisible": currentState})
}

// ==========================================
// PERSISTENCE (Issue 2)
// ==========================================

// loadConfig loads persistent settings from config.json
func loadConfig() {
	file, err := os.Open(configFile)
	if err != nil {
		// Config file doesn't exist yet, use defaults
		log.Println("No config.json found, using defaults")
		return
	}
	defer file.Close()

	var config PersistentConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Printf("Error decoding config.json: %v, using defaults", err)
		return
	}

	// Apply loaded settings
	mutex.Lock()
	showHeaders = config.ShowHeaders
	autoPlay = config.AutoPlay
	if config.FrameDuration >= 50 && config.FrameDuration <= 5000 {
		frameDuration = config.FrameDuration
	}
	if config.EspRefreshDuration >= 500 && config.EspRefreshDuration <= 30000 {
		espRefreshDuration = config.EspRefreshDuration
	}
	if config.GifFps >= 0 && config.GifFps <= 30 {
		gifFps = config.GifFps
	}
	if len(config.CycleItems) > 0 {
		cycleItems = config.CycleItems
	}
	if config.CycleItemCounter > 0 {
		cycleItemCounter = config.CycleItemCounter
	}
	if config.CurrentCity != "" {
		currentCity = config.CurrentCity
	}
	if config.CityLat != 0 || config.CityLng != 0 {
		cityLat = config.CityLat
		cityLng = config.CityLng
	}
	if config.TimezoneName != "" {
		timezoneName = config.TimezoneName
	}
	if config.DisplayRotation == 0 || config.DisplayRotation == 2 {
		displayRotation = config.DisplayRotation
	}
	// LED beacon settings
	if config.LedBrightness >= 0 && config.LedBrightness <= 100 {
		ledBrightness = config.LedBrightness
	}
	ledBeaconEnabled = config.LedBeaconEnabled
	// LED effect settings
	if config.LedEffectMode != "" {
		validModes := map[string]bool{"auto": true, "static": true, "flash": true, "pulse": true, "rainbow": true}
		if validModes[config.LedEffectMode] {
			ledEffectMode = config.LedEffectMode
		}
	}
	if config.LedCustomColor != "" && len(config.LedCustomColor) == 7 {
		ledCustomColor = config.LedCustomColor
	}
	if config.LedFlashSpeed >= 100 && config.LedFlashSpeed <= 2000 {
		ledFlashSpeed = config.LedFlashSpeed
	}
	if config.LedPulseSpeed >= 500 && config.LedPulseSpeed <= 3000 {
		ledPulseSpeed = config.LedPulseSpeed
	}
	// Display scale
	if config.DisplayScale != "" {
		validScales := map[string]bool{"compact": true, "normal": true, "large": true}
		if validScales[config.DisplayScale] {
			displayScale = config.DisplayScale
		}
	}
	// Pomodoro settings
	if config.PomodoroWorkDuration > 0 {
		pomodoroSettings.WorkDuration = config.PomodoroWorkDuration
		pomodoroSession.TimeRemaining = config.PomodoroWorkDuration
	}
	if config.PomodoroBreakDuration > 0 {
		pomodoroSettings.BreakDuration = config.PomodoroBreakDuration
	}
	if config.PomodoroLongBreak > 0 {
		pomodoroSettings.LongBreak = config.PomodoroLongBreak
	}
	if config.PomodoroCyclesUntil > 0 {
		pomodoroSettings.CyclesUntilLong = config.PomodoroCyclesUntil
	}
	pomodoroSettings.ShowInCycle = config.PomodoroShowInCycle
	// BCD Clock settings
	bcd24HourMode = config.BCD24HourMode
	bcdShowSeconds = config.BCDShowSeconds
	// Analog Clock settings
	analogShowSeconds = config.AnalogShowSeconds
	analogShowRoman = config.AnalogShowRoman
	// Spotify settings
	if config.SpotifyClientID != "" {
		spotifyCredentials.ClientID = config.SpotifyClientID
		spotifyCredentials.ClientSecret = config.SpotifyClientSecret
		spotifyCredentials.RefreshToken = config.SpotifyRefreshToken
		if config.SpotifyRefreshToken != "" {
			spotifyEnabled = true
		}
	}
	mutex.Unlock()

	log.Println("Loaded settings from config.json")
}

// saveConfig saves persistent settings to config.json (fault-tolerant)
func saveConfig() {
	mutex.Lock()
	config := PersistentConfig{
		ShowHeaders:           showHeaders,
		AutoPlay:              autoPlay,
		FrameDuration:         frameDuration,
		EspRefreshDuration:    espRefreshDuration,
		GifFps:                gifFps,
		DisplayRotation:       displayRotation,
		CycleItems:            cycleItems,
		CycleItemCounter:      cycleItemCounter,
		CurrentCity:           currentCity,
		CityLat:               cityLat,
		CityLng:               cityLng,
		TimezoneName:          timezoneName,
		LedBrightness:         ledBrightness,
		LedBeaconEnabled:      ledBeaconEnabled,
		LedEffectMode:         ledEffectMode,
		LedCustomColor:        ledCustomColor,
		LedFlashSpeed:         ledFlashSpeed,
		LedPulseSpeed:         ledPulseSpeed,
		DisplayScale:          displayScale,
		PomodoroWorkDuration:  pomodoroSettings.WorkDuration,
		PomodoroBreakDuration: pomodoroSettings.BreakDuration,
		PomodoroLongBreak:     pomodoroSettings.LongBreak,
		PomodoroCyclesUntil:   pomodoroSettings.CyclesUntilLong,
		PomodoroShowInCycle:   pomodoroSettings.ShowInCycle,
		BCD24HourMode:         bcd24HourMode,
		BCDShowSeconds:        bcdShowSeconds,
		AnalogShowSeconds:     analogShowSeconds,
		AnalogShowRoman:       analogShowRoman,
		SpotifyClientID:       spotifyCredentials.ClientID,
		SpotifyClientSecret:   spotifyCredentials.ClientSecret,
		SpotifyRefreshToken:   spotifyCredentials.RefreshToken,
	}
	mutex.Unlock()

	// Write to temp file first, then rename (atomic operation)
	tempFile := configFile + ".tmp"
	file, err := os.Create(tempFile)
	if err != nil {
		log.Printf("Error creating temp config file: %v", err)
		return
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		file.Close()
		os.Remove(tempFile)
		log.Printf("Error encoding config: %v", err)
		return
	}
	file.Close()

	// Atomic rename for fault tolerance
	if err := os.Rename(tempFile, configFile); err != nil {
		log.Printf("Error renaming config file: %v", err)
		os.Remove(tempFile)
		return
	}

	log.Println("Settings saved to config.json")
}

// ==========================================
// TIMEZONE (Issue 13)
// ==========================================

// initializeTimezone sets up the display timezone
func initializeTimezone() {
	// Try to load from configured timezone name
	var err error
	displayLocation, err = time.LoadLocation(timezoneName)
	if err != nil {
		// Fallback to fixed offset for common timezones
		log.Printf("Could not load timezone %s, using UTC: %v", timezoneName, err)
		displayLocation = time.UTC
	} else {
		log.Printf("Loaded timezone: %s", timezoneName)
	}
}

// handleTimezone handles timezone get/set
func handleTimezone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		tz := timezoneName
		mutex.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"timezone": tz})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Timezone string `json:"timezone"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Validate timezone
		loc, err := time.LoadLocation(req.Timezone)
		if err != nil {
			jsonError(w, "Invalid timezone: "+req.Timezone, http.StatusBadRequest)
			return
		}

		mutex.Lock()
		timezoneName = req.Timezone
		displayLocation = loc
		mutex.Unlock()

		saveConfig()

		log.Printf("🌍 Timezone updated: %s", req.Timezone)

		json.NewEncoder(w).Encode(map[string]string{"timezone": req.Timezone, "status": "updated"})
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	// Reset all state to defaults
	isCustomMode = false
	isGifMode = false
	showHeaders = true
	autoPlay = true
	frameDuration = 200
	espRefreshDuration = 3000
	gifFps = 0
	cycleItems = []CycleItem{
		{ID: "time-1", Type: "time", Label: "🕐 Time", Enabled: true, Duration: 3000},
		{ID: "bcd-1", Type: "bcd", Label: "🔢 BCD Clock", Enabled: true, Duration: 3000},
		{ID: "analog-1", Type: "analog", Label: "🧮 Analog Clock", Enabled: true, Duration: 3000},
		{ID: "weather-1", Type: "weather", Label: "🌤 Weather", Enabled: true, Duration: 3000},
	}
	cycleItemCounter = 4
	bcd24HourMode = true
	bcdShowSeconds = true
	analogShowSeconds = false
	analogShowRoman = false
	currentCity = "Bangalore"
	cityLat = 12.96
	cityLng = 77.57
	index = 0
	mutex.Unlock()

	// Refresh weather for default city
	go fetchWeather()

	log.Printf("🔄 System reset to defaults: city=%s, timezone=%s", currentCity, timezoneName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reset_complete"})
}

// ==========================================
// BCD CLOCK SETTINGS
// ==========================================

// handleBCDSettings handles BCD clock configuration get/set
func handleBCDSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		response := map[string]interface{}{
			"bcd24HourMode":  bcd24HourMode,
			"bcdShowSeconds": bcdShowSeconds,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			BCD24HourMode  *bool `json:"bcd24HourMode,omitempty"`
			BCDShowSeconds *bool `json:"bcdShowSeconds,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		var changes []string
		if req.BCD24HourMode != nil {
			bcd24HourMode = *req.BCD24HourMode
			if bcd24HourMode {
				changes = append(changes, "format=24hr")
			} else {
				changes = append(changes, "format=12hr")
			}
		}
		if req.BCDShowSeconds != nil {
			bcdShowSeconds = *req.BCDShowSeconds
			if bcdShowSeconds {
				changes = append(changes, "seconds=visible")
			} else {
				changes = append(changes, "seconds=hidden")
			}
		}
		response := map[string]interface{}{
			"bcd24HourMode":  bcd24HourMode,
			"bcdShowSeconds": bcdShowSeconds,
			"status":         "updated",
		}
		mutex.Unlock()

		// Persist settings
		go saveConfig()

		if len(changes) > 0 {
			log.Printf("🔢 BCD Clock settings updated: %s", strings.Join(changes, ", "))
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// ==========================================
// ANALOG CLOCK SETTINGS
// ==========================================

// handleAnalogSettings handles Analog clock configuration get/set
func handleAnalogSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		response := map[string]interface{}{
			"analogShowSeconds": analogShowSeconds,
			"analogShowRoman":   analogShowRoman,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			AnalogShowSeconds *bool `json:"analogShowSeconds,omitempty"`
			AnalogShowRoman   *bool `json:"analogShowRoman,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		var changes []string
		if req.AnalogShowSeconds != nil {
			analogShowSeconds = *req.AnalogShowSeconds
			if analogShowSeconds {
				changes = append(changes, "seconds=visible")
			} else {
				changes = append(changes, "seconds=hidden")
			}
		}
		if req.AnalogShowRoman != nil {
			analogShowRoman = *req.AnalogShowRoman
			if analogShowRoman {
				changes = append(changes, "numerals=Roman")
			} else {
				changes = append(changes, "numerals=markers")
			}
		}
		response := map[string]interface{}{
			"analogShowSeconds": analogShowSeconds,
			"analogShowRoman":   analogShowRoman,
			"status":            "updated",
		}
		mutex.Unlock()

		// Persist settings
		go saveConfig()

		if len(changes) > 0 {
			log.Printf("🧮 Analog Clock settings updated: %s", strings.Join(changes, ", "))
		}

		json.NewEncoder(w).Encode(response)
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}
