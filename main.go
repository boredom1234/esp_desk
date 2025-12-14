package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
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
	AutoPlay      bool `json:"autoPlay"`
	FrameDuration int  `json:"frameDuration"`
	ShowHeaders   bool `json:"showHeaders"`
	FrameCount    int  `json:"frameCount"`
	CurrentIndex  int  `json:"currentIndex"`
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

type WeatherData struct {
	City        string `json:"city"`
	Temperature string `json:"temperature"`
	Condition   string `json:"condition"`
	Icon        string `json:"icon"`
	Windspeed   string `json:"windspeed"`
	IsDay       bool   `json:"isDay"`
}

// ==========================================
// GLOBAL STATE
// ==========================================

var (
	frames        []Frame
	index         int
	mutex         sync.Mutex
	startTime     time.Time
	isCustomMode  bool = false
	showHeaders   bool = true
	autoPlay      bool = true
	frameDuration int  = 200

	// Weather state
	currentCity string  = "Kolkata"
	cityLat     float64 = 22.57
	cityLng     float64 = 88.36
	weatherData WeatherData
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
			AutoPlay:      autoPlay,
			FrameDuration: frameDuration,
			ShowHeaders:   showHeaders,
			FrameCount:    len(frames),
			CurrentIndex:  index,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(settings)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			AutoPlay      *bool `json:"autoPlay,omitempty"`
			FrameDuration *int  `json:"frameDuration,omitempty"`
			ShowHeaders   *bool `json:"showHeaders,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mutex.Lock()
		if req.AutoPlay != nil {
			autoPlay = *req.AutoPlay
		}
		if req.FrameDuration != nil {
			frameDuration = *req.FrameDuration
			if frameDuration < 50 {
				frameDuration = 50
			}
			if frameDuration > 5000 {
				frameDuration = 5000
			}
		}
		if req.ShowHeaders != nil {
			showHeaders = *req.ShowHeaders
		}
		settings := Settings{
			AutoPlay:      autoPlay,
			FrameDuration: frameDuration,
			ShowHeaders:   showHeaders,
			FrameCount:    len(frames),
			CurrentIndex:  index,
		}
		mutex.Unlock()

		json.NewEncoder(w).Encode(settings)
		return
	}
}

func handleToggleHeaders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	showHeaders = !showHeaders
	currentState := showHeaders
	mutex.Unlock()

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
		Style    string `json:"style"` // "normal", "centered", "framed"
		Duration int    `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Defaults
	if req.Size == 0 {
		req.Size = 2
	}
	if req.Duration == 0 {
		req.Duration = 5000
	}

	mutex.Lock()
	isCustomMode = true

	var elements []Element

	switch req.Style {
	case "centered":
		// Approximate centering for OLED
		charWidth := req.Size * 6
		textWidth := len(req.Text) * charWidth
		x := (128 - textWidth) / 2
		if x < 0 {
			x = 0
		}
		y := 28 // Vertically centered-ish
		elements = []Element{
			{Type: "text", X: x, Y: y, Size: req.Size, Value: req.Text},
		}

	case "framed":
		// Decorative frame with text
		elements = []Element{
			// Top border line
			{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
			// Bottom border line
			{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
			// Left border
			{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
			// Right border
			{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
			// Main text centered
			{Type: "text", X: 8, Y: 28, Size: req.Size, Value: req.Text},
		}

	default: // "normal"
		elements = []Element{
			{Type: "text", X: req.X, Y: req.Y, Size: req.Size, Value: req.Text},
		}
	}

	frames = []Frame{
		{Version: 1, Duration: req.Duration, Clear: true, Elements: elements},
	}
	index = 0
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "frameCount": 1})
}

func handleMarquee(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Text      string `json:"text"`
		Y         int    `json:"y"`
		Size      int    `json:"size"`
		Speed     int    `json:"speed"`     // pixels per frame
		Direction string `json:"direction"` // "left" or "right"
		Loops     int    `json:"loops"`     // number of complete scrolls
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	// Generate frames for each position
	var marqueeFrames []Frame
	frameTime := 50 // ms per frame

	for loop := 0; loop < req.Loops; loop++ {
		for offset := 0; offset < totalDistance; offset += req.Speed {
			var x int
			if req.Direction == "left" {
				x = 128 - offset
			} else {
				x = offset - textWidth
			}

			marqueeFrames = append(marqueeFrames, Frame{
				Version:  1,
				Duration: frameTime,
				Clear:    true,
				Elements: []Element{
					{Type: "text", X: x, Y: req.Y, Size: req.Size, Value: req.Text},
				},
			})
		}
	}

	mutex.Lock()
	isCustomMode = true
	frames = marqueeFrames
	index = 0
	mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"frameCount": len(marqueeFrames),
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
	showHeaders = true
	autoPlay = true
	frameDuration = 200
	currentCity = "Kolkata"
	cityLat = 22.57
	cityLng = 88.36
	index = 0
	mutex.Unlock()

	// Refresh weather for default city
	go fetchWeather()

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

func fetchWeather() {
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.2f&longitude=%.2f&current_weather=true", cityLat, cityLng)
	resp, err := http.Get(url)
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

	isDay := w.CurrentWeather.IsDay == 1
	weatherData = WeatherData{
		City:        currentCity,
		Temperature: fmt.Sprintf("%.1fC", w.CurrentWeather.Temperature),
		Condition:   getWeatherCondition(w.CurrentWeather.WeatherCode),
		Icon:        getWeatherIcon(w.CurrentWeather.WeatherCode, isDay),
		Windspeed:   fmt.Sprintf("%.0f km/h", w.CurrentWeather.Windspeed),
		IsDay:       isDay,
	}
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
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		mutex.Lock()
		currentCity = req.City
		cityLat = req.Latitude
		cityLng = req.Longitude
		mutex.Unlock()

		// Fetch weather for new location
		fetchWeather()

		mutex.Lock()
		data := weatherData
		mutex.Unlock()

		json.NewEncoder(w).Encode(data)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func updateLoop() {
	go func() {
		fetchWeather()
		ticker := time.NewTicker(15 * time.Minute)
		for range ticker.C {
			fetchWeather()
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mutex.Lock()

		if !isCustomMode {
			currentTime := time.Now().Format("15:04:05")
			uptime := time.Since(startTime).Round(time.Second).String()

			// More attractive frame layouts
			timeElements := []Element{
				{Type: "text", X: 20, Y: 22, Size: 2, Value: currentTime},
			}
			if showHeaders {
				timeElements = append([]Element{
					{Type: "text", X: 40, Y: 2, Size: 1, Value: "= TIME ="},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, timeElements...)
				timeElements = append(timeElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				timeElements = append(timeElements, Element{Type: "text", X: 45, Y: 55, Size: 1, Value: "IST"})
			}

			weatherElements := []Element{
				{Type: "text", X: 30, Y: 18, Size: 2, Value: weatherData.Temperature},
			}
			if showHeaders {
				weatherElements = append([]Element{
					{Type: "text", X: 28, Y: 2, Size: 1, Value: "= WEATHER ="},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, weatherElements...)
				weatherElements = append(weatherElements, Element{Type: "text", X: 25, Y: 38, Size: 1, Value: weatherData.Condition})
				weatherElements = append(weatherElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				weatherElements = append(weatherElements, Element{Type: "text", X: 40, Y: 55, Size: 1, Value: weatherData.City})
			}

			uptimeElements := []Element{
				{Type: "text", X: 10, Y: 28, Size: 1, Value: uptime},
			}
			if showHeaders {
				uptimeElements = append([]Element{
					{Type: "text", X: 32, Y: 2, Size: 1, Value: "= UPTIME ="},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, uptimeElements...)
			}

			frames = []Frame{
				{Version: 1, Duration: 3000, Clear: true, Elements: timeElements},
				{Version: 1, Duration: 3000, Clear: true, Elements: weatherElements},
				{Version: 1, Duration: 3000, Clear: true, Elements: uptimeElements},
			}
		}

		mutex.Unlock()
	}
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
// MAIN
// ==========================================

func main() {
	startTime = time.Now()

	frames = []Frame{{Duration: 1000, Clear: true, Elements: []Element{{Type: "text", X: 20, Y: 25, Size: 2, Value: "BOOTING..."}}}}

	go updateLoop()

	// Frame endpoints
	http.HandleFunc("/frame/current", loggingMiddleware(currentFrame))
	http.HandleFunc("/frame/next", loggingMiddleware(nextFrame))

	// Static files
	http.Handle("/", http.FileServer(http.Dir("./static")))

	// API endpoints
	http.HandleFunc("/api/frames", loggingMiddleware(handleFrames))
	http.HandleFunc("/api/control/next", loggingMiddleware(nextFrame))
	http.HandleFunc("/api/settings", loggingMiddleware(handleSettings))
	http.HandleFunc("/api/custom", loggingMiddleware(handleCustom))
	http.HandleFunc("/api/custom/text", loggingMiddleware(handleCustomText))
	http.HandleFunc("/api/custom/marquee", loggingMiddleware(handleMarquee))
	http.HandleFunc("/api/upload", loggingMiddleware(handleUpload))
	http.HandleFunc("/api/reset", loggingMiddleware(handleReset))
	http.HandleFunc("/api/settings/toggle-headers", loggingMiddleware(handleToggleHeaders))
	http.HandleFunc("/api/settings/headers-state", loggingMiddleware(handleGetHeadersState))
	http.HandleFunc("/api/weather", loggingMiddleware(handleWeather))

	log.Println("ESP Desk Backend v4 running on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
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

		for i, srcImg := range g.Image {
			if i >= 100 { // Limit frames
				break
			}

			bitmap := processImageToBitmap(srcImg, 128, 64)

			duration := g.Delay[i] * 10
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
	log.Printf("Uploaded %s. Generated %d frames.", header.Filename, frameCount)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"frameCount": frameCount,
		"autoPlay":   autoPlay,
	})
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
