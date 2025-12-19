package main

import (
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

// GifFullResponse contains all frames for local ESP32 playback
type GifFullResponse struct {
	IsGifMode  bool    `json:"isGifMode"`
	FrameCount int     `json:"frameCount"`
	GifFps     int     `json:"gifFps"`
	Frames     []Frame `json:"frames"`
}
