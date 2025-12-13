package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Element struct {
	Type  string `json:"type"`
	X     int    `json:"x,omitempty"`
	Y     int    `json:"y,omitempty"`
	Size  int    `json:"size,omitempty"`
	Value string `json:"value,omitempty"`
	Width int    `json:"width,omitempty"`
}

type Frame struct {
	Duration int       `json:"duration"`
	Clear    bool      `json:"clear"`
	Elements []Element `json:"elements"`
}

// OpenMeteo Response Struct
type WeatherResponse struct {
	CurrentWeather struct {
		Temperature float64 `json:"temperature"`
		WeatherCode int     `json:"weathercode"`
	} `json:"current_weather"`
}

var (
	frames      []Frame
	index       int
	mutex       sync.Mutex
	startTime   time.Time
	lastWeather string = "Loading..."
	// Mode control
	isCustomMode bool = false
)

func currentFrame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	// Safe guard against empty frames
	if len(frames) == 0 {
		http.Error(w, "No frames available", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(frames[index])
}

func nextFrame(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(frames) == 0 {
		return
	}

	index = (index + 1) % len(frames)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(frames[index])
}

func handleFrames(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	defer mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(frames)
		return
	}

	// Simple manual override capability
	if r.Method == http.MethodPost {
		nextFrame(w, r)
		return
	}
}

func handleCustom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mutex.Lock()
	isCustomMode = true
	frames = []Frame{
		{
			Duration: 5000, // Stay longer on custom text
			Clear:    true,
			Elements: []Element{
				{Type: "text", X: 0, Y: 10, Size: 1, Value: "MESSAGE:"},
				{Type: "text", X: 0, Y: 30, Size: 2, Value: req.Text},
			},
		},
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
	isCustomMode = false
	mutex.Unlock()

	// Force immediate update loop trigger logic if needed,
	// but the ticker will pick it up in <1s.
	w.WriteHeader(http.StatusOK)
}

func fetchWeather() {
	// Kolkata coordinates: 22.57, 88.36
	resp, err := http.Get("https://api.open-meteo.com/v1/forecast?latitude=22.57&longitude=88.36&current_weather=true")
	if err != nil {
		log.Println("Error fetching weather:", err)
		return
	}
	defer resp.Body.Close()

	var w WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		log.Println("Error decoding weather:", err)
		return
	}

	lastWeather = fmt.Sprintf("%.1fC", w.CurrentWeather.Temperature)
}

func updateLoop() {
	// Update weather every 15 minutes
	go func() {
		fetchWeather() // Initial fetch
		ticker := time.NewTicker(15 * time.Minute)
		for range ticker.C {
			fetchWeather()
		}
	}()

	// Update frames every second (for clock)
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mutex.Lock()

		if !isCustomMode {
			currentTime := time.Now().Format("15:04:05")
			uptime := time.Since(startTime).Round(time.Second).String()

			// Rebuild frames dynamically
			frames = []Frame{
				{
					Duration: 3000,
					Clear:    true,
					Elements: []Element{
						{Type: "text", X: 0, Y: 0, Size: 1, Value: "TIME (IST)"},
						{Type: "text", X: 0, Y: 20, Size: 2, Value: currentTime},
					},
				},
				{
					Duration: 3000,
					Clear:    true,
					Elements: []Element{
						{Type: "text", X: 0, Y: 0, Size: 1, Value: "WEATHER"},
						{Type: "text", X: 0, Y: 25, Size: 2, Value: lastWeather},
					},
				},
				{
					Duration: 3000,
					Clear:    true,
					Elements: []Element{
						{Type: "text", X: 0, Y: 0, Size: 1, Value: "SYSTEM UPTIME"},
						{Type: "text", X: 0, Y: 25, Size: 1, Value: uptime},
					},
				},
			}
		}

		mutex.Unlock()
	}
}

// Middleware for logging
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next(w, r)
		log.Printf("Completed %s %s in %v", r.Method, r.URL.Path, time.Since(start))
	}
}

func main() {
	startTime = time.Now()

	// Initialize with placeholder so we don't crash on immediate start
	frames = []Frame{{Duration: 1000, Clear: true, Elements: []Element{{Type: "text", Value: "BOOTING..."}}}}

	// Start dynamic updates
	go updateLoop()

	// Register handlers with logging
	http.HandleFunc("/frame/current", loggingMiddleware(currentFrame))
	http.HandleFunc("/frame/next", loggingMiddleware(nextFrame))

	// Dashboard API
	http.Handle("/", http.FileServer(http.Dir("./static")))
	http.HandleFunc("/api/frames", loggingMiddleware(handleFrames))
	http.HandleFunc("/api/control/next", loggingMiddleware(nextFrame))
	// New Endpoints
	http.HandleFunc("/api/custom", loggingMiddleware(handleCustom))
	http.HandleFunc("/api/reset", loggingMiddleware(handleReset))

	log.Println("Display backend running on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
