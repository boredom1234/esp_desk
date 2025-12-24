package main

import (
	"encoding/json"
	"log"
	"net/http"
)





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
		Style    string `json:"style"`    
		Centered bool   `json:"centered"` 
		Framed   bool   `json:"framed"`
		Large    bool   `json:"large"`
		Inverted bool   `json:"inverted"`
		Duration int    `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	
	switch req.Style {
	case "centered":
		req.Centered = true
	case "framed":
		req.Framed = true
	}

	
	size := 2
	if req.Large {
		size = 2
	} else if req.Size > 0 {
		size = req.Size
	} else {
		size = 1 
	}

	
	if req.Large {
		size = 2
	}

	if req.Duration == 0 {
		req.Duration = 5000
	}

	mutex.Lock()
	isCustomMode = true

	var elements []Element

	
	charCount := len([]rune(req.Text))
	textWidth := charCount*5*size + (charCount-1)*size
	if charCount <= 0 {
		textWidth = 0
	}

	
	x := req.X
	y := req.Y

	
	frameInset := 0
	if req.Framed {
		frameInset = 4 
	}

	
	if y == 0 {
		lineHeight := 7 * size
		if req.Framed {
			
			y = (64 - lineHeight) / 2
		} else {
			y = (64 - lineHeight) / 2
		}
	}

	
	if req.Centered {
		availableWidth := 128
		if req.Framed {
			availableWidth = 128 - (frameInset * 2) 
		}
		x = (availableWidth - textWidth) / 2
		if req.Framed {
			x += frameInset 
		}
		if x < frameInset {
			x = frameInset
		}
	} else if x == 0 {
		x = frameInset + 2 
	}

	
	if req.Framed {
		elements = append(elements,
			
			Element{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
			
			Element{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
			
			Element{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
			
			Element{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
		)
	}

	
	elements = append(elements, Element{
		Type:  "text",
		X:     x,
		Y:     y,
		Size:  size,
		Value: req.Text,
	})

	
	
	var finalFrames []Frame
	if req.Inverted {
		
		textFrame := Frame{
			Version:  1,
			Duration: req.Duration,
			Clear:    true,
			Elements: elements,
		}
		bitmapFrame := convertFrameToBitmap(textFrame)
		
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
		Speed     int    `json:"speed"`     
		Direction string `json:"direction"` 
		Loops     int    `json:"loops"`     
		MaxFrames int    `json:"maxFrames"` 
		Framed    bool   `json:"framed"`    
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	
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

	
	charWidth := req.Size * 6
	textWidth := len(req.Text) * charWidth
	totalDistance := 128 + textWidth 

	
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

	
	maxMarqueeFrames := req.MaxFrames
	if maxMarqueeFrames < 2 {
		maxMarqueeFrames = 5 
	}
	if maxMarqueeFrames > 20 {
		maxMarqueeFrames = 20
	}
	var selectedPositions []int
	totalPositions := len(allPositions)

	if totalPositions <= maxMarqueeFrames {
		selectedPositions = allPositions
	} else {
		
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

	
	var marqueeFrames []Frame
	frameTime := 50 

	
	if totalPositions > maxMarqueeFrames {
		frameTime = (totalPositions * 50) / len(selectedPositions)
	}

	for _, x := range selectedPositions {
		
		var frameElements []Element

		
		if req.Framed {
			frameElements = append(frameElements,
				
				Element{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
				
				Element{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
				
				Element{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
				
				Element{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
			)
		}

		
		frameElements = append(frameElements,
			Element{Type: "text", X: x, Y: req.Y, Size: req.Size, Value: req.Text},
		)

		
		textFrame := Frame{
			Version:  1,
			Duration: frameTime,
			Clear:    true,
			Elements: frameElements,
		}

		
		bitmapFrame := convertFrameToBitmap(textFrame)
		marqueeFrames = append(marqueeFrames, bitmapFrame)
	}

	mutex.Lock()
	isCustomMode = true
	isGifMode = true 
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
