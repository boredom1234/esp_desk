package main

import (
	"encoding/json"
	"log"
	"net/http"
)

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
	switch req.Style {
	case "centered":
		req.Centered = true
	case "framed":
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
