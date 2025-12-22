package main

import (
	"sync"
	"time"
)

// ==========================================
// GLOBAL STATE
// ==========================================

const configFile = "config.json"

var (
	frames             []Frame
	index              int
	mutex              sync.Mutex
	startTime          time.Time
	isCustomMode       bool   = false
	isGifMode          bool   = false // True when playing multi-frame GIF animation
	showHeaders        bool   = false
	autoPlay           bool   = true
	frameDuration      int    = 200
	espRefreshDuration int    = 3000      // Duration ESP32 waits before fetching next frame (ms)
	gifFps             int    = 0         // 0 = use original timing, 5-30 = override FPS
	displayRotation    int    = 0         // 0 = normal, 2 = 180 degrees (for upside-down mounting)
	ledBrightness      int    = 100       // 0-100 percentage for RGB LED beacon
	ledBeaconEnabled   bool   = true      // Enable/disable satellite beacon pulse
	ledEffectMode      string = "auto"    // "auto", "static", "flash", "pulse", "rainbow"
	ledCustomColor     string = "#0064FF" // Hex color for static/flash/pulse modes
	ledFlashSpeed      int    = 500       // Flash interval in ms (100-2000)
	ledPulseSpeed      int    = 1000      // Breathing cycle duration in ms (500-3000)
	displayScale       string = "normal"  // "compact", "normal", "large" - global display scale

	// BCD Clock settings
	bcd24HourMode  bool = true // true = 24-hour format, false = 12-hour format
	bcdShowSeconds bool = true // true = show seconds (6 columns), false = hide (4 columns)

	// Display cycle items - flexible list of what to display
	cycleItems = []CycleItem{
		{ID: "time-1", Type: "time", Label: "🕐 Time", Enabled: true, Duration: 3000},
		{ID: "bcd-1", Type: "bcd", Label: "🔢 BCD Clock", Enabled: true, Duration: 3000},
		{ID: "weather-1", Type: "weather", Label: "🌤 Weather", Enabled: true, Duration: 3000},
	}
	cycleItemCounter = 3 // For generating unique IDs

	// Weather state
	currentCity string  = "Bangalore"
	cityLat     float64 = 12.96
	cityLng     float64 = 77.57
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
