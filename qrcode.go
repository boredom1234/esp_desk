package main

import (
	"encoding/json"
	"log"
	"net/http"

	qrcode "github.com/skip2/go-qrcode"
)

// ==========================================
// QR CODE GENERATION
// ==========================================

// generateQRBitmap creates a 1-bit monochrome bitmap from text/URL
// Returns a 64x64 QR code centered on the 128x64 OLED display
func generateQRBitmap(data string) ([]int, int, int, error) {
	if data == "" {
		return nil, 0, 0, nil
	}

	// Generate QR code as PNG image
	// Size 64 fits nicely on 128x64 OLED, leaving room for label
	qr, err := qrcode.New(data, qrcode.Medium)
	if err != nil {
		return nil, 0, 0, err
	}

	// Get the QR as a 2D bit matrix
	qrBitmap := qr.Bitmap()
	qrSize := len(qrBitmap)

	// Base target size for OLED (64x64 for the QR, leaving width for label)
	baseTargetSize := 64

	// Apply global display scale
	scaledWidth, scaledHeight := getScaledBitmapSize(baseTargetSize, baseTargetSize)
	targetSize := scaledWidth
	if scaledHeight < targetSize {
		targetSize = scaledHeight
	}

	// Calculate scale factor
	scale := targetSize / qrSize
	if scale < 1 {
		scale = 1
	}

	// Final bitmap dimensions
	bitmapWidth := qrSize * scale
	bitmapHeight := qrSize * scale

	// Clamp to maximum 64x64
	if bitmapWidth > 64 {
		bitmapWidth = 64
	}
	if bitmapHeight > 64 {
		bitmapHeight = 64
	}

	// Calculate bytes per row (each byte = 8 horizontal pixels)
	bytesPerRow := (bitmapWidth + 7) / 8
	totalBytes := bytesPerRow * bitmapHeight

	bitmap := make([]int, totalBytes)

	// Convert QR matrix to 1-bit bitmap
	// QR code: true = black module, false = white module
	// OLED: 1 = white pixel (lit), 0 = black pixel (off)
	// We want QR black modules to be white on OLED for visibility
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

			// QR module is black (true) = set bit to 1 (white on OLED)
			if qrBitmap[srcY][srcX] {
				byteIndex := y*bytesPerRow + x/8
				bitIndex := 7 - (x % 8) // MSB first
				bitmap[byteIndex] |= 1 << bitIndex
			}
		}
	}

	return bitmap, bitmapWidth, bitmapHeight, nil
}

// generateQRFrame creates a complete Frame with QR code and optional label
func generateQRFrame(data string, duration int) (Frame, error) {
	bitmap, width, height, err := generateQRBitmap(data)
	if err != nil {
		return Frame{}, err
	}

	if duration <= 0 {
		duration = 5000 // 5 seconds default for QR codes (give time to scan)
	}

	// Center the QR on the display
	// OLED is 128x64, QR is 64x64, so offset by 32 to center horizontally
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

// handleQRCode handles QR code generation and immediate display
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

		// Generate QR frame
		frame, err := generateQRFrame(req.Data, req.Duration)
		if err != nil {
			jsonError(w, "Failed to generate QR code: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Set as current display
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
