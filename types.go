package main

import (
	"time"
)





type Element struct {
	Type      string `json:"type"`
	X         int    `json:"x,omitempty"`
	Y         int    `json:"y,omitempty"`
	Size      int    `json:"size,omitempty"`
	Value     string `json:"value,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Bitmap    []int  `json:"bitmap,omitempty"`
	Speed     int    `json:"speed,omitempty"`     
	Direction string `json:"direction,omitempty"` 
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
	DisplayRotation    int         `json:"displayRotation"` 
	FrameCount         int         `json:"frameCount"`
	CurrentIndex       int         `json:"currentIndex"`
	CycleItems         []CycleItem `json:"cycleItems"`
	LedBrightness      int         `json:"ledBrightness"`    
	LedBeaconEnabled   bool        `json:"ledBeaconEnabled"` 
	LedEffectMode      string      `json:"ledEffectMode"`    
	LedCustomColor     string      `json:"ledCustomColor"`   
	LedFlashSpeed      int         `json:"ledFlashSpeed"`    
	LedPulseSpeed      int         `json:"ledPulseSpeed"`    
	DisplayScale       string      `json:"displayScale"`     
}



type CycleItem struct {
	ID          string `json:"id"`                    
	Type        string `json:"type"`                  
	Label       string `json:"label"`                 
	Text        string `json:"text,omitempty"`        
	Style       string `json:"style,omitempty"`       
	Size        int    `json:"size,omitempty"`        
	Duration    int    `json:"duration,omitempty"`    
	Bitmap      []int  `json:"bitmap,omitempty"`      
	Width       int    `json:"width,omitempty"`       
	Height      int    `json:"height,omitempty"`      
	Enabled     bool   `json:"enabled"`               
	TargetDate  string `json:"targetDate,omitempty"`  
	TargetLabel string `json:"targetLabel,omitempty"` 
	QRData      string `json:"qrData,omitempty"`      
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
	AQI         int    `json:"aqi"`      
	AQILevel    string `json:"aqiLevel"` 
	PM25        string `json:"pm25"`     
	PM10        string `json:"pm10"`     
}


type MoonPhaseData struct {
	PhaseName     string  `json:"phaseName"`     
	PhaseAngle    float64 `json:"phaseAngle"`    
	Illumination  float64 `json:"illumination"`  
	Constellation string  `json:"constellation"` 
	DistanceKM    string  `json:"distanceKm"`    
	FetchedAt     string  `json:"fetchedAt"`     
}


type PersistentConfig struct {
	ShowHeaders        bool        `json:"showHeaders"`
	AutoPlay           bool        `json:"autoPlay"`
	FrameDuration      int         `json:"frameDuration"`
	EspRefreshDuration int         `json:"espRefreshDuration"`
	GifFps             int         `json:"gifFps"`
	DisplayRotation    int         `json:"displayRotation"` 
	CycleItems         []CycleItem `json:"cycleItems"`
	CycleItemCounter   int         `json:"cycleItemCounter"`
	CurrentCity        string      `json:"currentCity"`
	CityLat            float64     `json:"cityLat"`
	CityLng            float64     `json:"cityLng"`
	TimezoneName       string      `json:"timezoneName"`     
	LedBrightness      int         `json:"ledBrightness"`    
	LedBeaconEnabled   bool        `json:"ledBeaconEnabled"` 
	LedEffectMode      string      `json:"ledEffectMode"`    
	LedCustomColor     string      `json:"ledCustomColor"`   
	LedFlashSpeed      int         `json:"ledFlashSpeed"`    
	LedPulseSpeed      int         `json:"ledPulseSpeed"`    
	DisplayScale       string      `json:"displayScale"`     
	
	BCD24HourMode  bool `json:"bcd24HourMode"`  
	BCDShowSeconds bool `json:"bcdShowSeconds"` 
	
	AnalogShowSeconds bool `json:"analogShowSeconds"` 
	AnalogShowRoman   bool `json:"analogShowRoman"`   
	
	PomodoroWorkDuration  int  `json:"pomodoroWorkDuration"`  
	PomodoroBreakDuration int  `json:"pomodoroBreakDuration"` 
	PomodoroLongBreak     int  `json:"pomodoroLongBreak"`     
	PomodoroCyclesUntil   int  `json:"pomodoroCyclesUntil"`   
	PomodoroShowInCycle   bool `json:"pomodoroShowInCycle"`   
	
	SpotifyClientID     string `json:"spotifyClientId"`
	SpotifyClientSecret string `json:"spotifyClientSecret"`
	SpotifyRefreshToken string `json:"spotifyRefreshToken"`
	
	MoonPhaseData MoonPhaseData `json:"moonPhaseData"`
}


type LoginAttempt struct {
	Count     int
	LastReset time.Time
}


type PomodoroSession struct {
	Active          bool      `json:"active"`
	Mode            string    `json:"mode"`          
	TimeRemaining   int       `json:"timeRemaining"` 
	StartedAt       time.Time `json:"startedAt"`
	IsPaused        bool      `json:"isPaused"`
	PausedRemaining int       `json:"pausedRemaining"` 
	CyclesCompleted int       `json:"cyclesCompleted"`
}


type PomodoroSettings struct {
	WorkDuration    int  `json:"workDuration"`    
	BreakDuration   int  `json:"breakDuration"`   
	LongBreak       int  `json:"longBreak"`       
	CyclesUntilLong int  `json:"cyclesUntilLong"` 
	ShowInCycle     bool `json:"showInCycle"`     
}


type GifFullResponse struct {
	IsGifMode  bool    `json:"isGifMode"`
	FrameCount int     `json:"frameCount"`
	GifFps     int     `json:"gifFps"`
	Frames     []Frame `json:"frames"`
	
	LedBrightness    int    `json:"ledBrightness"`
	LedBeaconEnabled bool   `json:"ledBeaconEnabled"`
	LedEffectMode    string `json:"ledEffectMode"`
	LedCustomColor   string `json:"ledCustomColor"`
	LedFlashSpeed    int    `json:"ledFlashSpeed"`
	LedPulseSpeed    int    `json:"ledPulseSpeed"`
}
