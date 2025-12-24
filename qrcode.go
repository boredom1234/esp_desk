package main

import (
	"encoding/json"
	"log"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
)







func generateQRBitmap(data string) ([]int, int, int, error) {
	if data == "" {
		return nil, 0, 0, nil
	}

	
	
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		return nil, 0, 0, err
	}

	
	qrBitmap := qr.Bitmap()
	qrSize := len(qrBitmap)

	
	baseTargetSize := 64

	
	scaledWidth, scaledHeight := getScaledBitmapSize(baseTargetSize, baseTargetSize)
	targetSize := scaledWidth
	if scaledHeight < targetSize {
		targetSize = scaledHeight
	}

	
	scale := targetSize / qrSize
	if scale < 1 {
		scale = 1
	}

	
	bitmapWidth := qrSize * scale
	bitmapHeight := qrSize * scale

	
	if bitmapWidth > 64 {
		bitmapWidth = 64
	}
	if bitmapHeight > 64 {
		bitmapHeight = 64
	}

	
	bytesPerRow := (bitmapWidth + 7) / 8
	totalBytes := bytesPerRow * bitmapHeight

	bitmap := make([]int, totalBytes)

	
	
	
	
	for y := 0; y < bitmapHeight; y++ {
		srcY := y / scale
		if srcY >= qrSize {
			srcY = qrSize - 1
		}

		for x := 0; x < bitmapWidth; x++ {
			srcX := x / scale
			if srcX >= qrSize {
				srcX = qrSize - 1
			}

			
			if qrBitmap[srcY][srcX] {
				byteIndex := y*bytesPerRow + x/8
				bitIndex := 7 - (x % 8) 
				bitmap[byteIndex] |= 1 << bitIndex
			}
		}
	}

	return bitmap, bitmapWidth, bitmapHeight, nil
}


func generateQRFrame(data string, duration int) (Frame, error) {
	bitmap, width, height, err := generateQRBitmap(data)
	if err != nil {
		return Frame{}, err
	}

	if duration <= 0 {
		duration = 5000 
	}

	
	
	offsetX := (128 - width) / 2
	offsetY := (64 - height) / 2

	elements := []Element{
		{Type: "bitmap", X: offsetX, Y: offsetY, Width: width, Height: height, Bitmap: bitmap},
	}

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}, nil
}


func handleQRCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodPost {
		var req struct {
			Data     string `json:"data"`
			Duration int    `json:"duration,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		if req.Data == "" {
			jsonError(w, "Data field is required", http.StatusBadRequest)
			return
		}

		
		frame, err := generateQRFrame(req.Data, req.Duration)
		if err != nil {
			jsonError(w, "Failed to generate QR code: "+err.Error(), http.StatusInternalServerError)
			return
		}

		
		mutex.Lock()
		isCustomMode = true
		isGifMode = false
		frames = []Frame{frame}
		index = 0
		mutex.Unlock()

		log.Printf("📱 QR code displayed: %d chars of data", len(req.Data))

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"message": "QR code displayed",
		})
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}
