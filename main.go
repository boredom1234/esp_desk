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

type Element struct {
	Type   string `json:"type"`
	X      int    `json:"x,omitempty"`
	Y      int    `json:"y,omitempty"`
	Size   int    `json:"size,omitempty"`
	Value  string `json:"value,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Bitmap []int  `json:"bitmap,omitempty"`
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
	showHeaders  bool = true
)

// Sample 16x16 Heart Icon Bitmap (row-major bytes)
var heartBitmap = []int{
	0x00, 0x00, 0x18, 0x18, 0x3C, 0x3C, 0x7E, 0x7E,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x7E, 0x7E,
	0x3C, 0x3C, 0x18, 0x18, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

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

func handleToggleHeaders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	showHeaders = !showHeaders
	currentState := showHeaders
	mutex.Unlock()

	// Return the new state
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"headersVisible": currentState})
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
		// Image Mode
		el = Element{
			Type:   "bitmap",
			X:      0,
			Y:      0,
			Width:  req.Width,
			Height: req.Height,
			Bitmap: req.Bitmap,
		}
	} else {
		// Text Mode
		el = Element{
			Type:  "text",
			X:     0,
			Y:     30,
			Size:  2,
			Value: req.Text,
		}
	}

	// Construct elements slice
	var elements []Element
	if len(req.Bitmap) > 0 {
		// Image Mode - Only show the image
		elements = []Element{el}
	} else {
		// Text Mode
		elements = []Element{}
		if showHeaders {
			elements = append(elements, Element{Type: "text", X: 0, Y: 0, Size: 1, Value: "CUSTOM MSG:"})
		}
		elements = append(elements, el)
	}

	frames = []Frame{
		{
			Duration: 5000,
			Clear:    true,
			Elements: elements,
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

			// Helpers for clean frame construction
			timeElements := []Element{{Type: "text", X: 0, Y: 20, Size: 2, Value: currentTime}}
			if showHeaders {
				timeElements = append([]Element{{Type: "text", X: 0, Y: 0, Size: 1, Value: "TIME (IST)"}}, timeElements...)
			}

			weatherElements := []Element{{Type: "text", X: 0, Y: 25, Size: 2, Value: lastWeather}}
			if showHeaders {
				weatherElements = append([]Element{{Type: "text", X: 0, Y: 0, Size: 1, Value: "WEATHER"}}, weatherElements...)
			}

			uptimeElements := []Element{{Type: "text", X: 0, Y: 25, Size: 1, Value: uptime}}
			if showHeaders {
				uptimeElements = append([]Element{{Type: "text", X: 0, Y: 0, Size: 1, Value: "SYSTEM UPTIME"}}, uptimeElements...)
			}

			bitmapElements := []Element{{Type: "bitmap", X: 56, Y: 10, Width: 16, Height: 16, Bitmap: heartBitmap}}
			if showHeaders {
				bitmapElements = append([]Element{{Type: "text", X: 64, Y: 50, Size: 1, Value: "BITMAP TEST"}}, bitmapElements...)
			}

			// Rebuild frames dynamically
			frames = []Frame{
				{Duration: 3000, Clear: true, Elements: timeElements},
				{Duration: 3000, Clear: true, Elements: weatherElements},
				{Duration: 3000, Clear: true, Elements: uptimeElements},
				{Duration: 3000, Clear: true, Elements: bitmapElements},
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
	http.HandleFunc("/api/upload", loggingMiddleware(handleUpload)) // New upload handler
	http.HandleFunc("/api/reset", loggingMiddleware(handleReset))
	http.HandleFunc("/api/settings/toggle-headers", loggingMiddleware(handleToggleHeaders))

	log.Println("Display backend v3 running on :3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

// ==========================================
// IMAGE PROCESSING HELPERS
// ==========================================

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 10MB limit
	r.ParseMultipartForm(10 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Check mime type or try to decode as GIF first
	// We'll decode config first
	_, format, err := image.DecodeConfig(file)
	if err != nil {
		http.Error(w, "Unknown image format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Reset file pointer
	file.Seek(0, 0)

	mutex.Lock()
	defer mutex.Unlock()
	isCustomMode = true
	index = 0

	// Handle GIF
	if format == "gif" {
		g, err := gif.DecodeAll(file)
		if err != nil {
			http.Error(w, "Failed to decode GIF", http.StatusInternalServerError)
			return
		}

		frames = []Frame{} // Reset frames

		// Process each frame
		for i, srcImg := range g.Image {
			// Limit frame count to avoid memory issues
			if i >= 50 {
				break
			}

			// GIF frames might be partial updates (Disposal methods),
			// but for simplicity we treat them as individual frames or we rely on the backend to just send the bits.
			// Ideally we should composite over previous frame if not "restore background" etc.
			// But simple resize often "just works" for simple GIFs.

			// NOTE: real GIF handling is complex (transparency/disposal).
			// We will try to just maximize the bounds.

			// We need a helper to resize/dither
			bitmap := processImageToBitmap(srcImg, 128, 64)

			// Delay is in 100ths of a second
			duration := g.Delay[i] * 10
			if duration < 100 {
				duration = 100
			} // Min 100ms

			frames = append(frames, Frame{
				Duration: duration,
				Clear:    true,
				Elements: []Element{
					{Type: "bitmap", X: 0, Y: 0, Width: 128, Height: 64, Bitmap: bitmap},
				},
			})
		}

	} else {
		// Single Image (PNG/JPG)
		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, "Failed to decode image", http.StatusInternalServerError)
			return
		}

		bitmap := processImageToBitmap(img, 128, 64)
		frames = []Frame{
			{
				Duration: 5000,
				Clear:    true,
				Elements: []Element{
					{Type: "bitmap", X: 0, Y: 0, Width: 128, Height: 64, Bitmap: bitmap},
				},
			},
		}
	}

	log.Printf("Uploaded %s. Generated %d frames.", header.Filename, len(frames))
	w.WriteHeader(http.StatusOK)
}

func processImageToBitmap(src image.Image, width, height int) []int {
	bounds := src.Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()

	// Actually exact size: ceil(width/8) * height

	// Real basic nearest neighbor with centering
	bytesPerRow := (width + 7) / 8
	finalBitmap := make([]int, bytesPerRow*height)

	// Calculate target dimensions to keep aspect ratio
	targetW, targetH := width, height
	ratioSrc := float64(dx) / float64(dy)
	ratioDst := float64(width) / float64(height)

	if ratioSrc > ratioDst {
		// Source is wider, fit to width
		targetH = int(float64(width) / ratioSrc)
	} else {
		// Source is taller, fit to height
		targetW = int(float64(height) * ratioSrc)
	}

	offsetX := (width - targetW) / 2
	offsetY := (height - targetH) / 2

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			// Source coordinates
			srcX := int(float64(x) * float64(dx) / float64(targetW))
			srcY := int(float64(y) * float64(dy) / float64(targetH))

			r, g, b, _ := src.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA() // uint32 0-65535
			// Luminance
			lum := (19595*r + 38470*g + 7471*b + 1<<15) >> 24 // 0-255

			// Threshold
			if lum > 128 {
				// Set bit
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
