package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ==========================================
// DATA STRUCTURES
// ==========================================

type Element struct {
	Type      string `json:"type"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Size      int    `json:"size,omitempty"`
	Value     string `json:"value,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Bitmap    []int  `json:"bitmap,omitempty"`
	Speed     int    `json:"speed,omitempty"`     // For marquee
	Direction string `json:"direction,omitempty"` // "left" or "right"
}

type Frame struct {
	Version  int       `json:"version"`
	Duration int       `json:"duration"`
	Clear    bool      `json:"clear"`
	Elements []Element `json:"elements"`
}

type Settings struct {
	AutoPlay           bool        `json:"autoPlay"`
	FrameDuration      int         `json:"frameDuration"`
	EspRefreshDuration int         `json:"espRefreshDuration"`
	GifFps             int         `json:"gifFps"`
	ShowHeaders        bool        `json:"showHeaders"`
	DisplayRotation    int         `json:"displayRotation"` // 0 = normal, 2 = 180 degrees
	FrameCount         int         `json:"frameCount"`
	CurrentIndex       int         `json:"currentIndex"`
	CycleItems         []CycleItem `json:"cycleItems"`
	LedBrightness      int         `json:"ledBrightness"`    // 0-100 percentage for RGB LED beacon
	LedBeaconEnabled   bool        `json:"ledBeaconEnabled"` // Enable/disable satellite beacon
}

// CycleItem represents a single item in the display cycle
// Type can be: "time", "weather", "uptime", "text", "image"
type CycleItem struct {
	ID       string `json:"id"`                 // Unique ID for the item
	Type     string `json:"type"`               // "time", "weather", "uptime", "text", "image"
	Label    string `json:"label"`              // Display label for UI
	Text     string `json:"text,omitempty"`     // For text type: the message
	Style    string `json:"style,omitempty"`    // For text: "normal", "centered", "framed"
	Size     int    `json:"size,omitempty"`     // For text: font size
	Duration int    `json:"duration,omitempty"` // Display duration in ms (0 = use default)
	Bitmap   []int  `json:"bitmap,omitempty"`   // For image: bitmap data
	Width    int    `json:"width,omitempty"`    // For image: width
	Height   int    `json:"height,omitempty"`   // For image: height
	Enabled  bool   `json:"enabled"`            // Whether this item is active
}

type WeatherResponse struct {
	CurrentWeather struct {
		Temperature   float64 `json:"temperature"`
		Windspeed     float64 `json:"windspeed"`
		WindDirection int     `json:"winddirection"`
		WeatherCode   int     `json:"weathercode"`
		IsDay         int     `json:"is_day"`
	} `json:"current_weather"`
}

// AirQualityResponse from Open-Meteo Air Quality API
type AirQualityResponse struct {
	Current struct {
		PM25            float64 `json:"pm2_5"`
		PM10            float64 `json:"pm10"`
		EuropeanAQI     int     `json:"european_aqi"`
		USAQI           int     `json:"us_aqi"`
		EuropeanAQIPM25 int     `json:"european_aqi_pm2_5"`
		EuropeanAQIPM10 int     `json:"european_aqi_pm10"`
	} `json:"current"`
}

type WeatherData struct {
	City        string `json:"city"`
	Temperature string `json:"temperature"`
	Condition   string `json:"condition"`
	Icon        string `json:"icon"`
	Windspeed   string `json:"windspeed"`
	IsDay       bool   `json:"isDay"`
	AQI         int    `json:"aqi"`      // US AQI (0-500 scale)
	AQILevel    string `json:"aqiLevel"` // "Good", "Moderate", "Unhealthy", etc.
	PM25        string `json:"pm25"`     // PM2.5 concentration
	PM10        string `json:"pm10"`     // PM10 concentration
}

// PersistentConfig stores settings that survive server restarts (Issue 2)
type PersistentConfig struct {
	ShowHeaders        bool        `json:"showHeaders"`
	AutoPlay           bool        `json:"autoPlay"`
	FrameDuration      int         `json:"frameDuration"`
	EspRefreshDuration int         `json:"espRefreshDuration"`
	GifFps             int         `json:"gifFps"`
	DisplayRotation    int         `json:"displayRotation"` // 0 = normal, 2 = 180 degrees
	CycleItems         []CycleItem `json:"cycleItems"`
	CycleItemCounter   int         `json:"cycleItemCounter"`
	CurrentCity        string      `json:"currentCity"`
	CityLat            float64     `json:"cityLat"`
	CityLng            float64     `json:"cityLng"`
	TimezoneName       string      `json:"timezoneName"`     // Issue 13: configurable timezone
	LedBrightness      int         `json:"ledBrightness"`    // 0-100 percentage
	LedBeaconEnabled   bool        `json:"ledBeaconEnabled"` // Enable/disable beacon
	// Pomodoro settings
	PomodoroWorkDuration  int  `json:"pomodoroWorkDuration"`  // seconds
	PomodoroBreakDuration int  `json:"pomodoroBreakDuration"` // seconds
	PomodoroLongBreak     int  `json:"pomodoroLongBreak"`     // seconds
	PomodoroCyclesUntil   int  `json:"pomodoroCyclesUntil"`   // cycles until long break
	PomodoroShowInCycle   bool `json:"pomodoroShowInCycle"`   // show in display cycle
}

// LoginAttempt tracks rate limiting for auth (Issue 9)
type LoginAttempt struct {
	Count     int
	LastReset time.Time
}

// PomodoroSession tracks the active Pomodoro timer state
type PomodoroSession struct {
	Active          bool      `json:"active"`
	Mode            string    `json:"mode"`          // "work", "break", "longBreak"
	TimeRemaining   int       `json:"timeRemaining"` // seconds remaining
	StartedAt       time.Time `json:"startedAt"`
	IsPaused        bool      `json:"isPaused"`
	PausedRemaining int       `json:"pausedRemaining"` // time left when paused
	CyclesCompleted int       `json:"cyclesCompleted"`
}

// PomodoroSettings stores customizable timer durations
type PomodoroSettings struct {
	WorkDuration    int  `json:"workDuration"`    // seconds (default 25*60)
	BreakDuration   int  `json:"breakDuration"`   // seconds (default 5*60)
	LongBreak       int  `json:"longBreak"`       // seconds (default 15*60)
	CyclesUntilLong int  `json:"cyclesUntilLong"` // default 4
	ShowInCycle     bool `json:"showInCycle"`     // whether to display in cycle
}

// ==========================================
// GLOBAL STATE
// ==========================================

const configFile = "config.json"

var (
	frames             []Frame
	index              int
	mutex              sync.Mutex
	startTime          time.Time
	isCustomMode       bool = false
	isGifMode          bool = false // True when playing multi-frame GIF animation
	showHeaders        bool = true
	autoPlay           bool = true
	frameDuration      int  = 200
	espRefreshDuration int  = 3000 // Duration ESP32 waits before fetching next frame (ms)
	gifFps             int  = 0    // 0 = use original timing, 5-30 = override FPS
	displayRotation    int  = 0    // 0 = normal, 2 = 180 degrees (for upside-down mounting)
	ledBrightness      int  = 50   // 0-100 percentage for RGB LED beacon
	ledBeaconEnabled   bool = true // Enable/disable satellite beacon pulse

	// Display cycle items - flexible list of what to display
	cycleItems = []CycleItem{
		{ID: "time-1", Type: "time", Label: "🕐 Time", Enabled: true, Duration: 3000},
		{ID: "weather-1", Type: "weather", Label: "🌤 Weather", Enabled: true, Duration: 3000},
		{ID: "uptime-1", Type: "uptime", Label: "⏱ Uptime", Enabled: true, Duration: 3000},
	}
	cycleItemCounter = 3 // For generating unique IDs

	// Weather state
	currentCity string  = "Kolkata"
	cityLat     float64 = 22.57
	cityLng     float64 = 88.36
	weatherData WeatherData

	// Authentication state
	dashboardPassword     string                       // Password from env (plain text)
	dashboardPasswordHash string                       // Hashed password for secure comparison (Issue 5)
	authTokens            = make(map[string]time.Time) // session token -> expiry time
	authMutex             sync.RWMutex
	authEnabled           bool = false // Only enable auth if password is set

	// Rate limiting for login attempts (Issue 9)
	loginAttempts      = make(map[string]*LoginAttempt)
	loginAttemptsMutex sync.RWMutex
	maxLoginAttempts   = 5               // Max attempts before lockout
	loginLockoutTime   = 1 * time.Minute // Lockout duration

	// Timezone for display (Issue 13: now configurable)
	timezoneName    string = "Asia/Kolkata" // Default timezone
	displayLocation *time.Location

	// Pomodoro timer state
	pomodoroSession = PomodoroSession{
		Active:          false,
		Mode:            "work",
		TimeRemaining:   25 * 60, // 25 minutes default
		CyclesCompleted: 0,
	}
	pomodoroSettings = PomodoroSettings{
		WorkDuration:    25 * 60, // 25 minutes
		BreakDuration:   5 * 60,  // 5 minutes
		LongBreak:       15 * 60, // 15 minutes
		CyclesUntilLong: 4,
		ShowInCycle:     false,
	}
)

// Decorative border for attractive display
var borderTop = []int{
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
}

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
// TEXT-TO-BITMAP RENDERING
// ==========================================

// Basic 5x7 font for text rendering to bitmap
// Each character is represented as 5 bytes (columns), 7 bits tall
var font5x7 = map[rune][]byte{
	' ': {0x00, 0x00, 0x00, 0x00, 0x00},
	'A': {0x7C, 0x12, 0x11, 0x12, 0x7C},
	'B': {0x7F, 0x49, 0x49, 0x49, 0x36},
	'C': {0x3E, 0x41, 0x41, 0x41, 0x22},
	'D': {0x7F, 0x41, 0x41, 0x22, 0x1C},
	'E': {0x7F, 0x49, 0x49, 0x49, 0x41},
	'F': {0x7F, 0x09, 0x09, 0x09, 0x01},
	'G': {0x3E, 0x41, 0x49, 0x49, 0x7A},
	'H': {0x7F, 0x08, 0x08, 0x08, 0x7F},
	'I': {0x00, 0x41, 0x7F, 0x41, 0x00},
	'J': {0x20, 0x40, 0x41, 0x3F, 0x01},
	'K': {0x7F, 0x08, 0x14, 0x22, 0x41},
	'L': {0x7F, 0x40, 0x40, 0x40, 0x40},
	'M': {0x7F, 0x02, 0x0C, 0x02, 0x7F},
	'N': {0x7F, 0x04, 0x08, 0x10, 0x7F},
	'O': {0x3E, 0x41, 0x41, 0x41, 0x3E},
	'P': {0x7F, 0x09, 0x09, 0x09, 0x06},
	'Q': {0x3E, 0x41, 0x51, 0x21, 0x5E},
	'R': {0x7F, 0x09, 0x19, 0x29, 0x46},
	'S': {0x46, 0x49, 0x49, 0x49, 0x31},
	'T': {0x01, 0x01, 0x7F, 0x01, 0x01},
	'U': {0x3F, 0x40, 0x40, 0x40, 0x3F},
	'V': {0x1F, 0x20, 0x40, 0x20, 0x1F},
	'W': {0x3F, 0x40, 0x38, 0x40, 0x3F},
	'X': {0x63, 0x14, 0x08, 0x14, 0x63},
	'Y': {0x07, 0x08, 0x70, 0x08, 0x07},
	'Z': {0x61, 0x51, 0x49, 0x45, 0x43},
	'a': {0x20, 0x54, 0x54, 0x54, 0x78},
	'b': {0x7F, 0x48, 0x44, 0x44, 0x38},
	'c': {0x38, 0x44, 0x44, 0x44, 0x20},
	'd': {0x38, 0x44, 0x44, 0x48, 0x7F},
	'e': {0x38, 0x54, 0x54, 0x54, 0x18},
	'f': {0x08, 0x7E, 0x09, 0x01, 0x02},
	'g': {0x0C, 0x52, 0x52, 0x52, 0x3E},
	'h': {0x7F, 0x08, 0x04, 0x04, 0x78},
	'i': {0x00, 0x44, 0x7D, 0x40, 0x00},
	'j': {0x20, 0x40, 0x44, 0x3D, 0x00},
	'k': {0x7F, 0x10, 0x28, 0x44, 0x00},
	'l': {0x00, 0x41, 0x7F, 0x40, 0x00},
	'm': {0x7C, 0x04, 0x18, 0x04, 0x78},
	'n': {0x7C, 0x08, 0x04, 0x04, 0x78},
	'o': {0x38, 0x44, 0x44, 0x44, 0x38},
	'p': {0x7C, 0x14, 0x14, 0x14, 0x08},
	'q': {0x08, 0x14, 0x14, 0x18, 0x7C},
	'r': {0x7C, 0x08, 0x04, 0x04, 0x08},
	's': {0x48, 0x54, 0x54, 0x54, 0x20},
	't': {0x04, 0x3F, 0x44, 0x40, 0x20},
	'u': {0x3C, 0x40, 0x40, 0x20, 0x7C},
	'v': {0x1C, 0x20, 0x40, 0x20, 0x1C},
	'w': {0x3C, 0x40, 0x30, 0x40, 0x3C},
	'x': {0x44, 0x28, 0x10, 0x28, 0x44},
	'y': {0x0C, 0x50, 0x50, 0x50, 0x3C},
	'z': {0x44, 0x64, 0x54, 0x4C, 0x44},
	'0': {0x3E, 0x51, 0x49, 0x45, 0x3E},
	'1': {0x00, 0x42, 0x7F, 0x40, 0x00},
	'2': {0x42, 0x61, 0x51, 0x49, 0x46},
	'3': {0x21, 0x41, 0x45, 0x4B, 0x31},
	'4': {0x18, 0x14, 0x12, 0x7F, 0x10},
	'5': {0x27, 0x45, 0x45, 0x45, 0x39},
	'6': {0x3C, 0x4A, 0x49, 0x49, 0x30},
	'7': {0x01, 0x71, 0x09, 0x05, 0x03},
	'8': {0x36, 0x49, 0x49, 0x49, 0x36},
	'9': {0x06, 0x49, 0x49, 0x29, 0x1E},
	'!': {0x00, 0x00, 0x5F, 0x00, 0x00},
	'?': {0x02, 0x01, 0x51, 0x09, 0x06},
	'.': {0x00, 0x60, 0x60, 0x00, 0x00},
	',': {0x00, 0x80, 0x60, 0x00, 0x00},
	':': {0x00, 0x36, 0x36, 0x00, 0x00},
	'-': {0x08, 0x08, 0x08, 0x08, 0x08},
	'_': {0x40, 0x40, 0x40, 0x40, 0x40},
	'(': {0x00, 0x1C, 0x22, 0x41, 0x00},
	')': {0x00, 0x41, 0x22, 0x1C, 0x00},
	'/': {0x60, 0x10, 0x08, 0x04, 0x03},
	'@': {0x3E, 0x41, 0x5D, 0x55, 0x1E},
	// Additional special characters (Issue 15)
	'%':  {0x23, 0x13, 0x08, 0x64, 0x62},
	'+':  {0x08, 0x08, 0x3E, 0x08, 0x08},
	'=':  {0x14, 0x14, 0x14, 0x14, 0x14},
	'<':  {0x08, 0x14, 0x22, 0x41, 0x00},
	'>':  {0x00, 0x41, 0x22, 0x14, 0x08},
	'#':  {0x14, 0x7F, 0x14, 0x7F, 0x14},
	'*':  {0x22, 0x14, 0x7F, 0x14, 0x22},
	'&':  {0x36, 0x49, 0x55, 0x22, 0x50},
	'[':  {0x00, 0x7F, 0x41, 0x41, 0x00},
	']':  {0x00, 0x41, 0x41, 0x7F, 0x00},
	';':  {0x00, 0x80, 0x56, 0x00, 0x00},
	'\'': {0x00, 0x00, 0x07, 0x00, 0x00},
	'"':  {0x00, 0x07, 0x00, 0x07, 0x00},
}

// calcCenteredX calculates the X position to center text on a 128-pixel wide OLED
// Text width = charCount * 5 * size + (charCount - 1) * size (no trailing space)
func calcCenteredX(text string, size int) int {
	charCount := len([]rune(text))
	if charCount <= 0 {
		return 64 // Default to center
	}
	textWidth := charCount*5*size + (charCount-1)*size
	x := (128 - textWidth) / 2
	if x < 0 {
		x = 0
	}
	return x
}

// renderTextToBitmap converts a text element to a bitmap for ESP32 local playback
// Returns bitmap array (128x64 = 1024 bytes) as []int
// Uses the same format as processImageToBitmap: row-major, MSB first
func renderTextToBitmap(text string, x, y, size int) []int {
	const width = 128
	const height = 64

	// Create bitmap buffer matching processImageToBitmap format
	bytesPerRow := (width + 7) / 8 // 16 bytes per row for 128 width
	bitmap := make([]int, bytesPerRow*height)

	// Helper function to set a pixel (matching processImageToBitmap format)
	setPixel := func(px, py int) {
		if px < 0 || px >= width || py < 0 || py >= height {
			return
		}
		byteIndex := py*bytesPerRow + px/8
		if byteIndex < len(bitmap) {
			bitmap[byteIndex] |= (0x80 >> (px % 8))
		}
	}

	// Render each character
	currentX := x
	for _, char := range text {
		charData, exists := font5x7[char]
		if !exists {
			charData = font5x7[' '] // Default to space for unknown chars
		}

		// Draw character with scaling
		for col := 0; col < 5; col++ {
			for row := 0; row < 7; row++ {
				if charData[col]&(1<<row) != 0 {
					// Draw scaled pixel
					for sx := 0; sx < size; sx++ {
						for sy := 0; sy < size; sy++ {
							setPixel(currentX+col*size+sx, y+row*size+sy)
						}
					}
				}
			}
		}

		// Move to next character position (5 pixels + 1 pixel spacing) * size
		currentX += 6 * size

		// Stop if we've gone off screen
		if currentX >= width {
			break
		}
	}

	return bitmap
}

// convertFrameToBitmap converts a frame with text/line elements to a frame with a single bitmap element
func convertFrameToBitmap(frame Frame) Frame {
	const width = 128
	const height = 64
	bytesPerRow := (width + 7) / 8 // 16 bytes per row for 128 width
	bitmap := make([]int, bytesPerRow*height)

	// Helper function to set a pixel
	setPixel := func(px, py int) {
		if px < 0 || px >= width || py < 0 || py >= height {
			return
		}
		byteIndex := py*bytesPerRow + px/8
		if byteIndex < len(bitmap) {
			bitmap[byteIndex] |= (0x80 >> (px % 8))
		}
	}

	// Render all elements
	for _, el := range frame.Elements {
		switch el.Type {
		case "text":
			// Render text element
			currentX := el.X
			size := el.Size
			if size == 0 {
				size = 1
			}
			for _, char := range el.Value {
				charData, exists := font5x7[char]
				if !exists {
					charData = font5x7[' '] // Default to space
				}
				// Draw character with scaling
				for col := 0; col < 5; col++ {
					for row := 0; row < 7; row++ {
						if charData[col]&(1<<row) != 0 {
							for sx := 0; sx < size; sx++ {
								for sy := 0; sy < size; sy++ {
									setPixel(currentX+col*size+sx, el.Y+row*size+sy)
								}
							}
						}
					}
				}
				currentX += 6 * size
				if currentX >= width {
					break
				}
			}

		case "line":
			// Render line/rectangle element
			for x := el.X; x < el.X+el.Width; x++ {
				for y := el.Y; y < el.Y+el.Height; y++ {
					setPixel(x, y)
				}
			}
		}
	}

	// Return new frame with bitmap element
	return Frame{
		Version:  frame.Version,
		Duration: frame.Duration,
		Clear:    frame.Clear,
		Elements: []Element{
			{
				Type:   "bitmap",
				X:      0,
				Y:      0,
				Width:  128,
				Height: 64,
				Bitmap: bitmap,
			},
		},
	}
}

// ==========================================
// CUSTOM CONTENT HANDLERS
// ==========================================

func handleCustomText(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text     string `json:"text"`
		X        int    `json:"x"`
		Y        int    `json:"y"`
		Size     int    `json:"size"`
		Style    string `json:"style"`    // Legacy: "normal", "centered", "framed"
		Centered bool   `json:"centered"` // New: combined style flags
		Framed   bool   `json:"framed"`
		Large    bool   `json:"large"`
		Inverted bool   `json:"inverted"`
		Duration int    `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Legacy style support - convert to new flags
	if req.Style == "centered" {
		req.Centered = true
	} else if req.Style == "framed" {
		req.Framed = true
	}

	// Defaults
	size := 2
	if req.Large {
		size = 2
	} else if req.Size > 0 {
		size = req.Size
	} else {
		size = 1 // Default to normal size unless Large is checked
	}

	// If Large is checked, always use size 2
	if req.Large {
		size = 2
	}

	if req.Duration == 0 {
		req.Duration = 5000
	}

	mutex.Lock()
	isCustomMode = true

	var elements []Element

	// Calculate text position
	charCount := len([]rune(req.Text))
	textWidth := charCount*5*size + (charCount-1)*size
	if charCount <= 0 {
		textWidth = 0
	}

	// Default position
	x := req.X
	y := req.Y

	// Frame insets (if framed, text must be inside the border)
	frameInset := 0
	if req.Framed {
		frameInset = 4 // Pixels inside the frame
	}

	// Calculate Y position (centered vertically based on size)
	if y == 0 {
		lineHeight := 7 * size
		if req.Framed {
			// Center within frame area (between y=4 and y=59)
			y = (64 - lineHeight) / 2
		} else {
			y = (64 - lineHeight) / 2
		}
	}

	// Calculate X position
	if req.Centered {
		availableWidth := 128
		if req.Framed {
			availableWidth = 128 - (frameInset * 2) // Account for frame borders
		}
		x = (availableWidth - textWidth) / 2
		if req.Framed {
			x += frameInset // Offset by frame inset
		}
		if x < frameInset {
			x = frameInset
		}
	} else if x == 0 {
		x = frameInset + 2 // Small padding from left
	}

	// Add frame elements first if framed
	if req.Framed {
		elements = append(elements,
			// Top border line
			Element{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
			// Bottom border line
			Element{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
			// Left border
			Element{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
			// Right border
			Element{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
		)
	}

	// Add text element
	elements = append(elements, Element{
		Type:  "text",
		X:     x,
		Y:     y,
		Size:  size,
		Value: req.Text,
	})

	// Handle inverted mode (swap foreground/background)
	// For inverted, we'll use a bitmap approach
	var finalFrames []Frame
	if req.Inverted {
		// Convert to bitmap and invert pixels
		textFrame := Frame{
			Version:  1,
			Duration: req.Duration,
			Clear:    true,
			Elements: elements,
		}
		bitmapFrame := convertFrameToBitmap(textFrame)
		// Invert the bitmap
		for i, el := range bitmapFrame.Elements {
			if el.Type == "bitmap" {
				for j := range el.Bitmap {
					bitmapFrame.Elements[i].Bitmap[j] = ^el.Bitmap[j] & 0xFF
				}
			}
		}
		finalFrames = []Frame{bitmapFrame}
	} else {
		finalFrames = []Frame{
			{Version: 1, Duration: req.Duration, Clear: true, Elements: elements},
		}
	}

	frames = finalFrames
	index = 0
	mutex.Unlock()

	log.Printf("📝 Custom text: centered=%v, framed=%v, large=%v, inverted=%v", req.Centered, req.Framed, req.Large, req.Inverted)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "frameCount": 1})
}

func handleMarquee(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text      string `json:"text"`
		Y         int    `json:"y"`
		Size      int    `json:"size"`
		Speed     int    `json:"speed"`     // pixels per frame
		Direction string `json:"direction"` // "left" or "right"
		Loops     int    `json:"loops"`     // number of complete scrolls
		MaxFrames int    `json:"maxFrames"` // max frames for ESP32 memory
		Framed    bool   `json:"framed"`    // Static frame around scrolling area
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Defaults
	if req.Size == 0 {
		req.Size = 2
	}
	if req.Speed == 0 {
		req.Speed = 3
	}
	if req.Y == 0 {
		req.Y = 25
	}
	if req.Direction == "" {
		req.Direction = "left"
	}
	if req.Loops == 0 {
		req.Loops = 2
	}

	// Calculate text width (approximate: 6 pixels per char at size 1)
	charWidth := req.Size * 6
	textWidth := len(req.Text) * charWidth
	totalDistance := 128 + textWidth // Full scroll distance

	// Generate all frame positions first
	var allPositions []int
	for loop := 0; loop < req.Loops; loop++ {
		for offset := 0; offset < totalDistance; offset += req.Speed {
			var x int
			if req.Direction == "left" {
				x = 128 - offset
			} else {
				x = offset - textWidth
			}
			allPositions = append(allPositions, x)
		}
	}

	// Use user-specified max frames, with sensible bounds (2-20)
	maxMarqueeFrames := req.MaxFrames
	if maxMarqueeFrames < 2 {
		maxMarqueeFrames = 5 // default
	}
	if maxMarqueeFrames > 20 {
		maxMarqueeFrames = 20
	}
	var selectedPositions []int
	totalPositions := len(allPositions)

	if totalPositions <= maxMarqueeFrames {
		selectedPositions = allPositions
	} else {
		// Sample frames evenly across the animation
		step := float64(totalPositions) / float64(maxMarqueeFrames)
		for i := 0; i < maxMarqueeFrames; i++ {
			idx := int(float64(i) * step)
			if idx >= totalPositions {
				idx = totalPositions - 1
			}
			selectedPositions = append(selectedPositions, allPositions[idx])
		}
		log.Printf("Marquee: sampling %d positions down to %d frames", totalPositions, maxMarqueeFrames)
	}

	// Generate frames for selected positions
	var marqueeFrames []Frame
	frameTime := 50 // ms per frame

	// Adjust frame time to maintain approximate total animation duration
	if totalPositions > maxMarqueeFrames {
		frameTime = (totalPositions * 50) / len(selectedPositions)
	}

	for _, x := range selectedPositions {
		// Build frame elements
		var frameElements []Element

		// Add static frame border if requested
		if req.Framed {
			frameElements = append(frameElements,
				// Top border line
				Element{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
				// Bottom border line
				Element{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
				// Left border
				Element{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
				// Right border
				Element{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
			)
		}

		// Add scrolling text
		frameElements = append(frameElements,
			Element{Type: "text", X: x, Y: req.Y, Size: req.Size, Value: req.Text},
		)

		// Create text frame
		textFrame := Frame{
			Version:  1,
			Duration: frameTime,
			Clear:    true,
			Elements: frameElements,
		}

		// Convert text frame to bitmap frame for ESP32 local playback
		bitmapFrame := convertFrameToBitmap(textFrame)
		marqueeFrames = append(marqueeFrames, bitmapFrame)
	}

	mutex.Lock()
	isCustomMode = true
	isGifMode = true // Treat marquee as GIF for local ESP32 playback
	frames = marqueeFrames
	index = 0
	mutex.Unlock()

	log.Printf("Marquee generated: %d bitmap frames for local ESP32 playback", len(marqueeFrames))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"frameCount": len(marqueeFrames),
		"message":    "Marquee frames converted to bitmaps for local playback",
	})
}

func handleCustom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text   string `json:"text"`
		Bitmap []int  `json:"bitmap"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mutex.Lock()
	isCustomMode = true

	var el Element
	if len(req.Bitmap) > 0 {
		el = Element{
			Type:   "bitmap",
			X:      0,
			Y:      0,
			Width:  req.Width,
			Height: req.Height,
			Bitmap: req.Bitmap,
		}
	} else {
		el = Element{
			Type:  "text",
			X:     0,
			Y:     30,
			Size:  2,
			Value: req.Text,
		}
	}

	var elements []Element
	if len(req.Bitmap) > 0 {
		elements = []Element{el}
	} else {
		elements = []Element{}
		if showHeaders {
			elements = append(elements, Element{Type: "text", X: 0, Y: 0, Size: 1, Value: "> MESSAGE"})
		}
		elements = append(elements, el)
	}

	frames = []Frame{
		{Version: 1, Duration: 5000, Clear: true, Elements: elements},
	}
	index = 0
	mutex.Unlock()

	w.WriteHeader(http.StatusOK)
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
		{ID: "weather-1", Type: "weather", Label: "🌤 Weather", Enabled: true, Duration: 3000},
		{ID: "uptime-1", Type: "uptime", Label: "⏱ Uptime", Enabled: true, Duration: 3000},
	}
	cycleItemCounter = 3
	currentCity = "Kolkata"
	cityLat = 22.57
	cityLng = 88.36
	index = 0
	mutex.Unlock()

	// Refresh weather for default city
	go fetchWeather()

	log.Printf("🔄 System reset to defaults: city=%s, timezone=%s", currentCity, timezoneName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "reset_complete"})
}

func getWeatherIcon(code int, isDay bool) string {
	// WMO Weather interpretation codes
	switch {
	case code == 0:
		if isDay {
			return "☀"
		}
		return "☽"
	case code == 1, code == 2, code == 3:
		return "⛅"
	case code >= 45 && code <= 48:
		return "🌫"
	case code >= 51 && code <= 57:
		return "🌧"
	case code >= 61 && code <= 67:
		return "🌧"
	case code >= 71 && code <= 77:
		return "❄"
	case code >= 80 && code <= 82:
		return "🌧"
	case code >= 95 && code <= 99:
		return "⛈"
	default:
		return "🌡"
	}
}

func getWeatherCondition(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code == 1:
		return "Mostly Clear"
	case code == 2:
		return "Partly Cloudy"
	case code == 3:
		return "Overcast"
	case code >= 45 && code <= 48:
		return "Foggy"
	case code >= 51 && code <= 55:
		return "Drizzle"
	case code >= 56 && code <= 57:
		return "Freezing Drizzle"
	case code >= 61 && code <= 65:
		return "Rain"
	case code >= 66 && code <= 67:
		return "Freezing Rain"
	case code >= 71 && code <= 75:
		return "Snowfall"
	case code == 77:
		return "Snow Grains"
	case code >= 80 && code <= 82:
		return "Rain Showers"
	case code >= 85 && code <= 86:
		return "Snow Showers"
	case code == 95:
		return "Thunderstorm"
	case code >= 96 && code <= 99:
		return "Thunderstorm + Hail"
	default:
		return "Unknown"
	}
}

// getAQILevel returns a human-readable AQI level based on US AQI scale
func getAQILevel(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy (SG)"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}

func fetchWeather() {
	// Read coordinates with mutex to avoid race conditions
	mutex.Lock()
	lat := cityLat
	lng := cityLng
	city := currentCity
	mutex.Unlock()

	// Fetch weather data from Open-Meteo
	weatherURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.2f&longitude=%.2f&current_weather=true", lat, lng)
	weatherResp, err := http.Get(weatherURL)
	if err != nil {
		log.Println("Error fetching weather:", err)
		return
	}
	defer weatherResp.Body.Close()

	var w WeatherResponse
	if err := json.NewDecoder(weatherResp.Body).Decode(&w); err != nil {
		log.Println("Error decoding weather:", err)
		return
	}

	isDay := w.CurrentWeather.IsDay == 1
	newData := WeatherData{
		City:        city,
		Temperature: fmt.Sprintf("%.1fC", w.CurrentWeather.Temperature),
		Condition:   getWeatherCondition(w.CurrentWeather.WeatherCode),
		Icon:        getWeatherIcon(w.CurrentWeather.WeatherCode, isDay),
		Windspeed:   fmt.Sprintf("%.0f km/h", w.CurrentWeather.Windspeed),
		IsDay:       isDay,
		AQI:         0,
		AQILevel:    "N/A",
		PM25:        "N/A",
		PM10:        "N/A",
	}

	// Fetch air quality data from Open-Meteo Air Quality API
	aqiURL := fmt.Sprintf("https://air-quality-api.open-meteo.com/v1/air-quality?latitude=%.2f&longitude=%.2f&current=pm2_5,pm10,european_aqi,us_aqi,european_aqi_pm2_5,european_aqi_pm10", lat, lng)
	aqiResp, err := http.Get(aqiURL)
	if err != nil {
		log.Println("Error fetching AQI (continuing with weather only):", err)
	} else {
		defer aqiResp.Body.Close()
		var aq AirQualityResponse
		if err := json.NewDecoder(aqiResp.Body).Decode(&aq); err != nil {
			log.Println("Error decoding AQI:", err)
		} else {
			newData.AQI = aq.Current.USAQI
			newData.AQILevel = getAQILevel(aq.Current.USAQI)
			newData.PM25 = fmt.Sprintf("%.1f", aq.Current.PM25)
			newData.PM10 = fmt.Sprintf("%.1f", aq.Current.PM10)
			log.Printf("AQI fetched: US AQI=%d, PM2.5=%.1f, PM10=%.1f", aq.Current.USAQI, aq.Current.PM25, aq.Current.PM10)
		}
	}

	// Write weather data with mutex protection
	mutex.Lock()
	weatherData = newData
	mutex.Unlock()
}

func handleWeather(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		data := weatherData
		mutex.Unlock()
		json.NewEncoder(w).Encode(data)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			City      string  `json:"city"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate coordinates (Issue 7)
		if req.Latitude < -90 || req.Latitude > 90 {
			jsonError(w, "Invalid latitude: must be between -90 and 90", http.StatusBadRequest)
			return
		}
		if req.Longitude < -180 || req.Longitude > 180 {
			jsonError(w, "Invalid longitude: must be between -180 and 180", http.StatusBadRequest)
			return
		}
		if req.City == "" {
			jsonError(w, "City name is required", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		currentCity = req.City
		cityLat = req.Latitude
		cityLng = req.Longitude
		mutex.Unlock()

		// Persist settings (Issue 2)
		go saveConfig()

		// Fetch weather for new location
		fetchWeather()

		log.Printf("🌤️  Weather city changed: %s (%.2f, %.2f)", req.City, req.Latitude, req.Longitude)

		mutex.Lock()
		data := weatherData
		mutex.Unlock()

		json.NewEncoder(w).Encode(data)
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func updateLoop() {
	go func() {
		fetchWeather()
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			fetchWeather()
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mutex.Lock()

		// Pomodoro timer countdown (runs every second)
		if pomodoroSession.Active && !pomodoroSession.IsPaused {
			if pomodoroSession.TimeRemaining > 0 {
				pomodoroSession.TimeRemaining--
			} else {
				// Timer reached 0 - auto-transition to next phase
				if pomodoroSession.Mode == "work" {
					pomodoroSession.CyclesCompleted++
					if pomodoroSession.CyclesCompleted >= pomodoroSettings.CyclesUntilLong {
						pomodoroSession.Mode = "longBreak"
						pomodoroSession.TimeRemaining = pomodoroSettings.LongBreak
						pomodoroSession.CyclesCompleted = 0
						log.Printf("🍅 Pomodoro: Auto-started long break (%d min)", pomodoroSettings.LongBreak/60)
					} else {
						pomodoroSession.Mode = "break"
						pomodoroSession.TimeRemaining = pomodoroSettings.BreakDuration
						log.Printf("🍅 Pomodoro: Auto-started break (%d min)", pomodoroSettings.BreakDuration/60)
					}
				} else {
					// Break ended - start new work session
					pomodoroSession.Mode = "work"
					pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
					log.Printf("🍅 Pomodoro: Auto-started work session (%d min)", pomodoroSettings.WorkDuration/60)
				}
				pomodoroSession.StartedAt = time.Now()
			}
		}

		if !isCustomMode {
			// Use configurable timezone for time display
			now := time.Now()
			if displayLocation != nil {
				now = now.In(displayLocation)
			}
			currentTime := now.Format("15:04:05")
			uptime := time.Since(startTime).Round(time.Second).String()

			// Build frame map for each type
			frameMap := make(map[string]Frame)

			// Time frame
			// Get timezone abbreviation from current time in selected location
			tzAbbrev, _ := now.Zone()
			timeElements := []Element{
				{Type: "text", X: calcCenteredX(currentTime, 2), Y: 22, Size: 2, Value: currentTime},
			}
			if showHeaders {
				timeHeaderText := "= TIME ="
				timeElements = append([]Element{
					{Type: "text", X: calcCenteredX(timeHeaderText, 1), Y: 2, Size: 1, Value: timeHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, timeElements...)
				timeElements = append(timeElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				timeElements = append(timeElements, Element{Type: "text", X: calcCenteredX(tzAbbrev, 1), Y: 55, Size: 1, Value: tzAbbrev})
			}
			frameMap["time"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: timeElements}

			// Weather frame - now includes AQI
			// Build compact AQI display (just number, fits OLED width)
			aqiDisplay := ""
			if weatherData.AQI > 0 {
				aqiDisplay = fmt.Sprintf("AQI:%d", weatherData.AQI)
			}

			weatherElements := []Element{
				{Type: "text", X: calcCenteredX(weatherData.Temperature, 2), Y: 20, Size: 2, Value: weatherData.Temperature},
			}
			if showHeaders {
				weatherHeaderText := "= WEATHER ="
				weatherElements = append([]Element{
					{Type: "text", X: calcCenteredX(weatherHeaderText, 1), Y: 2, Size: 1, Value: weatherHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, weatherElements...)
				// Condition and AQI on same line if both fit
				if aqiDisplay != "" {
					// Show condition + AQI side by side
					weatherElements = append(weatherElements, Element{Type: "text", X: 5, Y: 42, Size: 1, Value: weatherData.Condition})
					weatherElements = append(weatherElements, Element{Type: "text", X: 75, Y: 42, Size: 1, Value: aqiDisplay})
				} else {
					weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(weatherData.Condition, 1), Y: 42, Size: 1, Value: weatherData.Condition})
				}
				weatherElements = append(weatherElements, Element{Type: "line", X: 0, Y: 53, Width: 128, Height: 1})
				weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(weatherData.City, 1), Y: 56, Size: 1, Value: weatherData.City})
			} else {
				// Without headers, show compact info
				weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(weatherData.Condition, 1), Y: 42, Size: 1, Value: weatherData.Condition})
				if aqiDisplay != "" {
					weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(aqiDisplay, 1), Y: 52, Size: 1, Value: aqiDisplay})
				}
			}
			frameMap["weather"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: weatherElements}

			// Uptime frame
			uptimeElements := []Element{
				{Type: "text", X: calcCenteredX(uptime, 1), Y: 28, Size: 1, Value: uptime},
			}
			if showHeaders {
				uptimeHeaderText := "= UPTIME ="
				uptimeElements = append([]Element{
					{Type: "text", X: calcCenteredX(uptimeHeaderText, 1), Y: 2, Size: 1, Value: uptimeHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, uptimeElements...)
			}
			frameMap["uptime"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: uptimeElements}

			// Pomodoro frame - clean, centered countdown display
			pomodoroMinutes := pomodoroSession.TimeRemaining / 60
			pomodoroSeconds := pomodoroSession.TimeRemaining % 60
			pomodoroTimeStr := fmt.Sprintf("%02d:%02d", pomodoroMinutes, pomodoroSeconds)

			// Mode display text
			var modeText string
			switch pomodoroSession.Mode {
			case "work":
				modeText = "FOCUS"
			case "break":
				modeText = "BREAK"
			case "longBreak":
				modeText = "LONG BREAK"
			default:
				modeText = "READY"
			}

			// Status indicator
			statusText := ""
			if pomodoroSession.IsPaused {
				statusText = "PAUSED"
			} else if !pomodoroSession.Active {
				statusText = "READY"
				modeText = "POMODORO"
			}

			// Cycle progress
			cycleText := fmt.Sprintf("%d/%d", pomodoroSession.CyclesCompleted, pomodoroSettings.CyclesUntilLong)

			// Build clean Pomodoro display elements
			// Large centered countdown timer for visibility
			timeX := calcCenteredX(pomodoroTimeStr, 2)
			pomodoroElements := []Element{
				// Large time display in center
				{Type: "text", X: timeX, Y: 22, Size: 2, Value: pomodoroTimeStr},
			}

			if showHeaders {
				// Header with mode
				headerText := fmt.Sprintf("= %s =", modeText)
				headerX := calcCenteredX(headerText, 1)
				pomodoroElements = append([]Element{
					{Type: "text", X: headerX, Y: 2, Size: 1, Value: headerText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, pomodoroElements...)

				// Footer with status and cycles
				pomodoroElements = append(pomodoroElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				if statusText != "" {
					pomodoroElements = append(pomodoroElements, Element{Type: "text", X: 8, Y: 55, Size: 1, Value: statusText})
				}
				pomodoroElements = append(pomodoroElements, Element{Type: "text", X: 90, Y: 55, Size: 1, Value: cycleText})
			} else {
				// Without headers, add mode text below timer (centered)
				modeX := calcCenteredX(modeText, 1)
				pomodoroElements = append(pomodoroElements, Element{Type: "text", X: modeX, Y: 48, Size: 1, Value: modeText})
			}

			frameMap["pomodoro"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: pomodoroElements}

			// Build frames array based on cycleItems
			frames = []Frame{}
			for _, item := range cycleItems {
				if !item.Enabled {
					continue
				}

				duration := item.Duration
				if duration <= 0 {
					duration = 3000 // Default duration
				}

				switch item.Type {
				case "time":
					frame := frameMap["time"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "weather":
					frame := frameMap["weather"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "uptime":
					frame := frameMap["uptime"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "text":
					// Custom text message
					var elements []Element
					textSize := item.Size
					if textSize <= 0 {
						textSize = 2
					}

					switch item.Style {
					case "centered":
						charWidth := textSize * 6
						textWidth := len(item.Text) * charWidth
						x := (128 - textWidth) / 2
						if x < 0 {
							x = 0
						}
						elements = []Element{
							{Type: "text", X: x, Y: 28, Size: textSize, Value: item.Text},
						}
					case "framed":
						elements = []Element{
							{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
							{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
							{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
							{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
							{Type: "text", X: 8, Y: 28, Size: textSize, Value: item.Text},
						}
					default: // "normal"
						elements = []Element{
							{Type: "text", X: 4, Y: 28, Size: textSize, Value: item.Text},
						}
					}

					if showHeaders && item.Label != "" {
						elements = append([]Element{
							{Type: "text", X: 32, Y: 2, Size: 1, Value: "= MESSAGE ="},
							{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
						}, elements...)
					}

					frames = append(frames, Frame{Version: 1, Duration: duration, Clear: true, Elements: elements})

				case "image":
					// Custom image
					if len(item.Bitmap) > 0 {
						elements := []Element{
							{Type: "bitmap", X: 0, Y: 0, Width: item.Width, Height: item.Height, Bitmap: item.Bitmap},
						}
						frames = append(frames, Frame{Version: 1, Duration: duration, Clear: true, Elements: elements})
					}

				case "pomodoro":
					// Pomodoro timer display
					frame := frameMap["pomodoro"]
					frame.Duration = duration
					frames = append(frames, frame)
				}
			}

			// Auto-include Pomodoro in cycle if enabled in settings and not already in cycle
			if pomodoroSettings.ShowInCycle {
				hasPomodoroInCycle := false
				for _, item := range cycleItems {
					if item.Type == "pomodoro" && item.Enabled {
						hasPomodoroInCycle = true
						break
					}
				}
				if !hasPomodoroInCycle {
					frame := frameMap["pomodoro"]
					frame.Duration = 3000
					frames = append(frames, frame)
				}
			}

			// Fallback: if no frames, show at least time
			if len(frames) == 0 {
				frames = append(frames, frameMap["time"])
			}
		}

		mutex.Unlock()
	}
}

// ==========================================
// GIF FULL DOWNLOAD FOR ESP32 LOCAL PLAYBACK
// ==========================================

// GifFullResponse contains all frames for local ESP32 playback
type GifFullResponse struct {
	IsGifMode  bool    `json:"isGifMode"`
	FrameCount int     `json:"frameCount"`
	GifFps     int     `json:"gifFps"`
	Frames     []Frame `json:"frames"`
}

// handleGifFull returns all GIF frames at once for ESP32 to store and play locally
// This eliminates per-frame API calls during animation playback
func handleGifFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")

	// If not in GIF mode or no frames, return empty response
	if !isGifMode || len(frames) == 0 {
		log.Printf("📡 ESP32 check: isGifMode=false (polling mode)")
		json.NewEncoder(w).Encode(GifFullResponse{
			IsGifMode:  false,
			FrameCount: len(frames),
			GifFps:     gifFps,
			Frames:     nil,
		})
		return
	}

	// Limit frames to match ESP32 MAX_GIF_FRAMES (10 frames max for memory safety)
	maxFrames := 10
	framesToSend := make([]Frame, 0, maxFrames)

	// Calculate frame duration based on FPS override
	fpsOverrideDuration := 0
	if gifFps > 0 {
		fpsOverrideDuration = 1000 / gifFps // Convert FPS to ms per frame
	}

	for i, frame := range frames {
		if i >= maxFrames {
			log.Printf("Warning: Limiting GIF to %d frames for ESP32 memory", maxFrames)
			break
		}

		// Apply FPS override if set
		frameCopy := frame
		if fpsOverrideDuration > 0 {
			frameCopy.Duration = fpsOverrideDuration
		}
		framesToSend = append(framesToSend, frameCopy)
	}

	log.Printf("📡 ESP32 check: isGifMode=true (%d frames sent for local playback)", len(framesToSend))

	resp := GifFullResponse{
		IsGifMode:  true,
		FrameCount: len(framesToSend),
		GifFps:     gifFps,
		Frames:     framesToSend,
	}

	// Buffer the JSON to calculate precise length
	// This avoids chunked transfer encoding which confuses ESP32 streaming parser
	jsonData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling GIF JSON: %v", err)
		http.Error(w, "JSON marshal error", http.StatusInternalServerError)
		return
	}

	log.Printf("📡 Sending GIF payload: %d bytes", len(jsonData))

	// Explicitly set Content-Length so ESP32 knows exactly how much to read
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// ==========================================
// MIDDLEWARE
// ==========================================

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	}
}

// ==========================================
// AUTHENTICATION
// ==========================================

// Generate a secure random token
func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Hash password with SHA256
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// Verify session token
func isValidToken(token string) bool {
	if !authEnabled || token == "" {
		return !authEnabled // If auth disabled, always valid; if enabled and no token, invalid
	}

	authMutex.RLock()
	expiry, exists := authTokens[token]
	authMutex.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		// Token expired, clean it up
		authMutex.Lock()
		delete(authTokens, token)
		authMutex.Unlock()
		return false
	}

	return true
}

// Create a new session token
func createSession() string {
	token := generateToken()
	expiry := time.Now().Add(24 * time.Hour) // 24 hour session

	authMutex.Lock()
	authTokens[token] = expiry
	authMutex.Unlock()

	return token
}

// Authentication middleware
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If auth not enabled, pass through
		if !authEnabled {
			next(w, r)
			return
		}

		// Check Authorization header
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if isValidToken(token) {
			next(w, r)
			return
		}

		// Check cookie as fallback
		cookie, err := r.Cookie("esp_desk_token")
		if err == nil && isValidToken(cookie.Value) {
			next(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

// Handle login request
func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client IP for rate limiting (Issue 9)
	clientIP := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = strings.Split(forwardedFor, ",")[0]
	}

	// Check rate limit
	if checkRateLimit(clientIP) {
		log.Printf("Rate limited login attempt from %s", clientIP)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Too many login attempts. Please try again later.",
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Compare passwords using constant-time comparison (Issue 5)
	submittedHash := hashPassword(req.Password)
	if subtle.ConstantTimeCompare([]byte(submittedHash), []byte(dashboardPasswordHash)) != 1 {
		recordFailedLogin(clientIP)
		log.Printf("Failed login attempt from %s", clientIP)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid password",
		})
		return
	}

	// Clear rate limit on successful login
	clearLoginAttempts(clientIP)

	// Create session
	token := createSession()
	log.Printf("Successful login from %s", clientIP)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "esp_desk_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, // 24 hours
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}

// Check if user is authenticated
func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// If auth not enabled, always return authenticated
	if !authEnabled {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"authRequired":  false,
		})
		return
	}

	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if isValidToken(token) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"authRequired":  true,
		})
		return
	}

	// Check cookie
	cookie, err := r.Cookie("esp_desk_token")
	if err == nil && isValidToken(cookie.Value) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"authRequired":  true,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": false,
		"authRequired":  true,
	})
}

// Handle logout
func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Remove token from storage
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if token != "" {
		authMutex.Lock()
		delete(authTokens, token)
		authMutex.Unlock()
	}

	// Also check cookie
	cookie, err := r.Cookie("esp_desk_token")
	if err == nil {
		authMutex.Lock()
		delete(authTokens, cookie.Value)
		authMutex.Unlock()
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "esp_desk_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
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
		PomodoroWorkDuration:  pomodoroSettings.WorkDuration,
		PomodoroBreakDuration: pomodoroSettings.BreakDuration,
		PomodoroLongBreak:     pomodoroSettings.LongBreak,
		PomodoroCyclesUntil:   pomodoroSettings.CyclesUntilLong,
		PomodoroShowInCycle:   pomodoroSettings.ShowInCycle,
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
// TOKEN CLEANUP (Issue 4)
// ==========================================

// cleanupExpiredTokens removes expired auth tokens periodically
func cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		now := time.Now()
		authMutex.Lock()
		count := 0
		for token, expiry := range authTokens {
			if now.After(expiry) {
				delete(authTokens, token)
				count++
			}
		}
		authMutex.Unlock()
		if count > 0 {
			log.Printf("Cleaned up %d expired auth tokens", count)
		}
	}
}

// ==========================================
// RATE LIMITING (Issue 9)
// ==========================================

// checkRateLimit returns true if the IP is rate-limited
func checkRateLimit(ip string) bool {
	loginAttemptsMutex.RLock()
	attempt, exists := loginAttempts[ip]
	loginAttemptsMutex.RUnlock()

	if !exists {
		return false
	}

	// Reset counter if lockout time has passed
	if time.Since(attempt.LastReset) > loginLockoutTime {
		loginAttemptsMutex.Lock()
		delete(loginAttempts, ip)
		loginAttemptsMutex.Unlock()
		return false
	}

	return attempt.Count >= maxLoginAttempts
}

// recordFailedLogin records a failed login attempt for rate limiting
func recordFailedLogin(ip string) {
	loginAttemptsMutex.Lock()
	defer loginAttemptsMutex.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		loginAttempts[ip] = &LoginAttempt{Count: 1, LastReset: time.Now()}
		return
	}

	// Reset if lockout expired
	if time.Since(attempt.LastReset) > loginLockoutTime {
		attempt.Count = 1
		attempt.LastReset = time.Now()
	} else {
		attempt.Count++
	}
}

// clearLoginAttempts clears rate limit for an IP after successful login
func clearLoginAttempts(ip string) {
	loginAttemptsMutex.Lock()
	delete(loginAttempts, ip)
	loginAttemptsMutex.Unlock()
}

// cleanupLoginAttempts periodically removes old login attempt records
func cleanupLoginAttempts() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		now := time.Now()
		loginAttemptsMutex.Lock()
		for ip, attempt := range loginAttempts {
			if now.Sub(attempt.LastReset) > loginLockoutTime*2 {
				delete(loginAttempts, ip)
			}
		}
		loginAttemptsMutex.Unlock()
	}
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

// ==========================================
// JSON ERROR HELPER (Issue 11)
// ==========================================

// jsonError sends a consistent JSON error response
func jsonError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": status,
	})
}

// ==========================================
// MAIN
// ==========================================

// loadEnvFile reads a .env file and sets environment variables
func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {
		// .env file not found, that's okay
		return
	}
	defer file.Close()

	// Read entire file (Issue 10: fix truncation)
	contentBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading .env file: %v", err)
		return
	}
	lines := strings.Split(string(contentBytes), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			// Remove quotes if present
			value = strings.Trim(value, "\"'")
			os.Setenv(key, value)
		}
	}
	log.Println("Loaded .env file")
}

func main() {
	startTime = time.Now()

	// Load .env file if present
	loadEnvFile()

	// Load persistent config (Issue 2)
	loadConfig()

	// Initialize authentication from environment variable
	dashboardPassword = os.Getenv("DASHBOARD_PASSWORD")
	if dashboardPassword != "" {
		authEnabled = true
		// Hash password for secure comparison (Issue 5)
		dashboardPasswordHash = hashPassword(dashboardPassword)
		log.Printf("Authentication ENABLED - password required to access dashboard")
	} else {
		log.Printf("Authentication DISABLED - no DASHBOARD_PASSWORD set")
	}

	// Initialize timezone for time display (Issue 13: configurable)
	initializeTimezone()

	// Start cleanup goroutines (Issues 4, 9)
	go cleanupExpiredTokens()
	go cleanupLoginAttempts()

	frames = []Frame{{Duration: 1000, Clear: true, Elements: []Element{{Type: "text", X: 20, Y: 25, Size: 2, Value: "BOOTING..."}}}}

	go updateLoop()

	// Frame endpoints (ESP32 access - no auth required)
	http.HandleFunc("/frame/current", loggingMiddleware(currentFrame))
	http.HandleFunc("/frame/next", loggingMiddleware(nextFrame))
	http.HandleFunc("/api/gif/full", loggingMiddleware(handleGifFull))

	// Auth endpoints (no auth required to access these)
	http.HandleFunc("/api/auth/login", loggingMiddleware(handleAuthLogin))
	http.HandleFunc("/api/auth/verify", loggingMiddleware(handleAuthVerify))
	http.HandleFunc("/api/auth/logout", loggingMiddleware(handleAuthLogout))

	// Static files
	http.Handle("/", http.FileServer(http.Dir("./static")))

	// Protected API endpoints (require authentication)
	http.HandleFunc("/api/frames", loggingMiddleware(authMiddleware(handleFrames)))
	http.HandleFunc("/api/control/next", loggingMiddleware(authMiddleware(nextFrame)))
	http.HandleFunc("/api/control/prev", loggingMiddleware(authMiddleware(prevFrame)))
	http.HandleFunc("/api/settings", loggingMiddleware(authMiddleware(handleSettings)))
	http.HandleFunc("/api/custom", loggingMiddleware(authMiddleware(handleCustom)))
	http.HandleFunc("/api/custom/text", loggingMiddleware(authMiddleware(handleCustomText)))
	http.HandleFunc("/api/custom/marquee", loggingMiddleware(authMiddleware(handleMarquee)))
	http.HandleFunc("/api/upload", loggingMiddleware(authMiddleware(handleUpload)))
	http.HandleFunc("/api/reset", loggingMiddleware(authMiddleware(handleReset)))
	http.HandleFunc("/api/settings/toggle-headers", loggingMiddleware(authMiddleware(handleToggleHeaders)))
	http.HandleFunc("/api/settings/headers-state", loggingMiddleware(authMiddleware(handleGetHeadersState)))
	http.HandleFunc("/api/weather", loggingMiddleware(authMiddleware(handleWeather)))
	http.HandleFunc("/api/settings/timezone", loggingMiddleware(authMiddleware(handleTimezone)))
	http.HandleFunc("/api/pomodoro", loggingMiddleware(authMiddleware(handlePomodoro)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("ESP Desk Backend v4 running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ==========================================
// IMAGE UPLOAD HANDLER
// ==========================================

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	_, format, err := image.DecodeConfig(file)
	if err != nil {
		http.Error(w, "Unknown image format: "+err.Error(), http.StatusBadRequest)
		return
	}

	file.Seek(0, 0)

	mutex.Lock()
	defer mutex.Unlock()
	isCustomMode = true
	index = 0

	if format == "gif" {
		g, err := gif.DecodeAll(file)
		if err != nil {
			http.Error(w, "Failed to decode GIF", http.StatusInternalServerError)
			return
		}

		frames = []Frame{}
		isGifMode = true // Multi-frame GIF - enable local playback mode

		totalFrames := len(g.Image)

		// Parse maxFrames from form data, with sensible bounds (2-20)
		maxFrames := 10 // default
		if maxFramesStr := r.FormValue("maxFrames"); maxFramesStr != "" {
			if parsed, err := strconv.Atoi(maxFramesStr); err == nil {
				maxFrames = parsed
			}
		}
		if maxFrames < 2 {
			maxFrames = 2
		}
		if maxFrames > 20 {
			maxFrames = 20
		}
		log.Printf("GIF upload: using maxFrames=%d (user setting)", maxFrames)

		// Calculate which frames to sample if we exceed the limit
		var frameIndices []int
		if totalFrames <= maxFrames {
			// Use all frames
			for i := 0; i < totalFrames; i++ {
				frameIndices = append(frameIndices, i)
			}
			log.Printf("GIF has %d frames, using all", totalFrames)
		} else {
			// Sample frames evenly to fit within limit
			step := float64(totalFrames) / float64(maxFrames)
			for i := 0; i < maxFrames; i++ {
				frameIdx := int(float64(i) * step)
				if frameIdx >= totalFrames {
					frameIdx = totalFrames - 1
				}
				frameIndices = append(frameIndices, frameIdx)
			}
			log.Printf("GIF has %d frames, sampling down to %d frames (step: %.2f)", totalFrames, maxFrames, step)
		}

		// Calculate total original duration for timing adjustment
		totalOriginalDuration := 0
		for _, delay := range g.Delay {
			totalOriginalDuration += delay * 10
		}

		// Process selected frames
		for _, frameIdx := range frameIndices {
			srcImg := g.Image[frameIdx]
			bitmap := processImageToBitmap(srcImg, 128, 64)

			var duration int
			if totalFrames > maxFrames {
				// Distribute total animation time evenly across sampled frames
				duration = totalOriginalDuration / len(frameIndices)
			} else {
				duration = g.Delay[frameIdx] * 10
			}

			if duration < 50 {
				duration = 50
			}

			frames = append(frames, Frame{
				Version:  1,
				Duration: duration,
				Clear:    true,
				Elements: []Element{
					{Type: "bitmap", X: 0, Y: 0, Width: 128, Height: 64, Bitmap: bitmap},
				},
			})
		}

	} else {
		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, "Failed to decode image", http.StatusInternalServerError)
			return
		}

		isGifMode = false // Single image - use polling mode
		bitmap := processImageToBitmap(img, 128, 64)
		frames = []Frame{
			{
				Version:  1,
				Duration: 5000,
				Clear:    true,
				Elements: []Element{
					{Type: "bitmap", X: 0, Y: 0, Width: 128, Height: 64, Bitmap: bitmap},
				},
			},
		}
	}

	frameCount := len(frames)
	if isGifMode {
		log.Printf("🎬 GIF uploaded: %s (%d frames, local playback enabled)", header.Filename, frameCount)
	} else {
		log.Printf("🖼️  Image uploaded: %s (format=%s)", header.Filename, format)
	}
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"frameCount": frameCount,
		"autoPlay":   autoPlay,
	}

	// Include bitmap data for single images (not GIF) so frontend can save to display cycle
	if format != "gif" && frameCount == 1 {
		el := frames[0].Elements[0]
		response["bitmap"] = el.Bitmap
		response["width"] = el.Width
		response["height"] = el.Height
		log.Printf("Including bitmap data for save-to-cycle: %dx%d, %d bytes", el.Width, el.Height, len(el.Bitmap))
	}

	json.NewEncoder(w).Encode(response)
}

// ==========================================
// IMAGE PROCESSING
// ==========================================

func processImageToBitmap(src image.Image, width, height int) []int {
	bounds := src.Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()

	bytesPerRow := (width + 7) / 8
	finalBitmap := make([]int, bytesPerRow*height)

	targetW, targetH := width, height
	ratioSrc := float64(dx) / float64(dy)
	ratioDst := float64(width) / float64(height)

	if ratioSrc > ratioDst {
		targetH = int(float64(width) / ratioSrc)
	} else {
		targetW = int(float64(height) * ratioSrc)
	}

	offsetX := (width - targetW) / 2
	offsetY := (height - targetH) / 2

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := int(float64(x) * float64(dx) / float64(targetW))
			srcY := int(float64(y) * float64(dy) / float64(targetH))

			r, g, b, _ := src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA()
			lum := (19595*r + 38470*g + 7471*b + 1<<15) >> 24

			if lum > 128 {
				drawX := x + offsetX
				drawY := y + offsetY

				byteIndex := drawY*bytesPerRow + drawX/8
				if byteIndex < len(finalBitmap) {
					finalBitmap[byteIndex] |= (0x80 >> (drawX % 8))
				}
			}
		}
	}

	return finalBitmap
}
