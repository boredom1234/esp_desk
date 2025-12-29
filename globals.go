package main

import (
	"sync"
	"time"
)

const configFile = "config.json"

var (
	frames             []Frame
	index              int
	mutex              sync.Mutex
	startTime          time.Time
	isCustomMode       bool   = false
	isGifMode          bool   = false
	showHeaders        bool   = false
	autoPlay           bool   = true
	frameDuration      int    = 200
	espRefreshDuration int    = 3000
	gifFps             int    = 0
	displayRotation    int    = 0
	ledBrightness      int    = 100
	ledBeaconEnabled   bool   = true
	ledEffectMode      string = "auto"
	ledCustomColor     string = "#0064FF"
	ledFlashSpeed      int    = 500
	ledPulseSpeed      int    = 1000
	displayScale       string = "normal"

	bcd24HourMode  bool = true
	bcdShowSeconds bool = true

	timeShowSeconds bool = true

	analogShowSeconds bool = false
	analogShowRoman   bool = false

	cycleItems = []CycleItem{
		{ID: "time-1", Type: "time", Label: "🕐 Time", Enabled: true, Duration: 3000},
		{ID: "bcd-1", Type: "bcd", Label: "🔢 BCD Clock", Enabled: true, Duration: 3000},
		{ID: "analog-1", Type: "analog", Label: "🧮 Analog Clock", Enabled: true, Duration: 3000},
		{ID: "weather-1", Type: "weather", Label: "🌤 Weather", Enabled: true, Duration: 3000},
	}
	cycleItemCounter = 4

	currentCity string  = "Bangalore"
	cityLat     float64 = 12.96
	cityLng     float64 = 77.57
	weatherData WeatherData

	moonPhaseData      MoonPhaseData
	moonPhaseLastFetch time.Time

	dashboardPassword     string
	dashboardPasswordHash string
	authTokens            = make(map[string]time.Time)
	authMutex             sync.RWMutex
	authEnabled           bool = false

	loginAttempts      = make(map[string]*LoginAttempt)
	loginAttemptsMutex sync.RWMutex
	maxLoginAttempts   = 5
	loginLockoutTime   = 1 * time.Minute

	timezoneName    string = "Asia/Kolkata"
	displayLocation *time.Location

	pomodoroSession = PomodoroSession{
		Active:          false,
		Mode:            "work",
		TimeRemaining:   25 * 60,
		CyclesCompleted: 0,
	}
	pomodoroSettings = PomodoroSettings{
		WorkDuration:    25 * 60,
		BreakDuration:   5 * 60,
		LongBreak:       15 * 60,
		CyclesUntilLong: 4,
		ShowInCycle:     false,
	}
)
