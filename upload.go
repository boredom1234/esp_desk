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
	"strconv"
)







func handleGifFull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")

	
	if !isGifMode || len(frames) == 0 {
		log.Printf("📡 ESP32 check: isGifMode=false (polling mode)")
		json.NewEncoder(w).Encode(GifFullResponse{
			IsGifMode:        false,
			FrameCount:       len(frames),
			GifFps:           gifFps,
			Frames:           nil,
			LedBrightness:    ledBrightness,
			LedBeaconEnabled: ledBeaconEnabled,
			LedEffectMode:    ledEffectMode,
			LedCustomColor:   ledCustomColor,
			LedFlashSpeed:    ledFlashSpeed,
			LedPulseSpeed:    ledPulseSpeed,
		})
		return
	}

	
	maxFrames := 10
	framesToSend := make([]Frame, 0, maxFrames)

	
	fpsOverrideDuration := 0
	if gifFps > 0 {
		fpsOverrideDuration = 1000 / gifFps 
	}

	for i, frame := range frames {
		if i >= maxFrames {
			log.Printf("Warning: Limiting GIF to %d frames for ESP32 memory", maxFrames)
			break
		}

		
		frameCopy := frame
		if fpsOverrideDuration > 0 {
			frameCopy.Duration = fpsOverrideDuration
		}
		framesToSend = append(framesToSend, frameCopy)
	}

	log.Printf("📡 ESP32 check: isGifMode=true (%d frames sent for local playback)", len(framesToSend))

	resp := GifFullResponse{
		IsGifMode:        true,
		FrameCount:       len(framesToSend),
		GifFps:           gifFps,
		Frames:           framesToSend,
		LedBrightness:    ledBrightness,
		LedBeaconEnabled: ledBeaconEnabled,
		LedEffectMode:    ledEffectMode,
		LedCustomColor:   ledCustomColor,
		LedFlashSpeed:    ledFlashSpeed,
		LedPulseSpeed:    ledPulseSpeed,
	}

	
	
	jsonData, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling GIF JSON: %v", err)
		http.Error(w, "JSON marshal error", http.StatusInternalServerError)
		return
	}

	log.Printf("📡 Sending GIF payload: %d bytes", len(jsonData))

	
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}





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
		isGifMode = true 

		totalFrames := len(g.Image)

		
		maxFrames := 10 
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

		
		var frameIndices []int
		if totalFrames <= maxFrames {
			
			for i := 0; i < totalFrames; i++ {
				frameIndices = append(frameIndices, i)
			}
			log.Printf("GIF has %d frames, using all", totalFrames)
		} else {
			
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

		
		totalOriginalDuration := 0
		for _, delay := range g.Delay {
			totalOriginalDuration += delay * 10
		}

		
		for _, frameIdx := range frameIndices {
			srcImg := g.Image[frameIdx]
			bitmap := processImageToBitmap(srcImg, 128, 64)

			var duration int
			if totalFrames > maxFrames {
				
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

		isGifMode = false 
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

	
	if format != "gif" && frameCount == 1 {
		el := frames[0].Elements[0]
		response["bitmap"] = el.Bitmap
		response["width"] = el.Width
		response["height"] = el.Height
		log.Printf("Including bitmap data for save-to-cycle: %dx%d, %d bytes", el.Width, el.Height, len(el.Bitmap))
	}

	json.NewEncoder(w).Encode(response)
}
