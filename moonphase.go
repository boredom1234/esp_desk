package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

// ==========================================
// MOON PHASE WIDGET (Astronomy API Integration)
// ==========================================

// AstronomyAPIResponse represents the response from astronomyapi.com
type AstronomyAPIResponse struct {
	Data struct {
		Table struct {
			Rows []struct {
				Cells []struct {
					ExtraInfo struct {
						Phase struct {
							Angle    string `json:"angel"` // Note: API has typo "angel"
							Fraction string `json:"fraction"`
							String   string `json:"string"`
						} `json:"phase"`
					} `json:"extraInfo"`
					Distance struct {
						FromEarth struct {
							KM string `json:"km"`
						} `json:"fromEarth"`
					} `json:"distance"`
					Position struct {
						Constellation struct {
							Name string `json:"name"`
						} `json:"constellation"`
					} `json:"position"`
				} `json:"cells"`
			} `json:"rows"`
		} `json:"table"`
	} `json:"data"`
}

// initMoonPhase initializes the moon phase API key from environment
func initMoonPhase() {
	moonPhaseAPIKey = os.Getenv("ASTRONOMY_API_KEY")
	if moonPhaseAPIKey != "" {
		log.Println("🌙 Astronomy API key configured")
	} else {
		log.Println("🌙 ASTRONOMY_API_KEY not set, using calculated moon phase")
	}
}

// handleMoonPhaseRefresh handles manual refresh requests from the UI
func handleMoonPhaseRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	log.Println("🌙 Manual moon phase refresh triggered")

	// Try to fetch from API
	err := fetchMoonPhaseFromAPI()

	mutex.Lock()
	data := moonPhaseData
	mutex.Unlock()

	if err != nil {
		log.Printf("🌙 Manual refresh failed: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"phaseName": data.PhaseName,
			"source":    "fallback",
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"phaseName":     data.PhaseName,
		"illumination":  data.Illumination,
		"constellation": data.Constellation,
		"source":        "api",
	})
}

// startMoonPhaseFetcher starts the background moon phase fetcher
// Fetches every 6 hours with robust error handling and retry logic
func startMoonPhaseFetcher() {
	go func() {
		// Check if we have cached data that's still fresh (less than 6 hours old)
		mutex.Lock()
		lastFetch := moonPhaseLastFetch
		hasData := moonPhaseData.PhaseName != ""
		mutex.Unlock()

		if hasData && time.Since(lastFetch) < 6*time.Hour {
			log.Printf("🌙 Using cached moon phase data (fetched %s ago)", time.Since(lastFetch).Round(time.Minute))
		} else {
			// Initial fetch with retry
			fetchMoonPhaseWithRetry(3)
		}

		// Schedule periodic fetches every 6 hours
		ticker := time.NewTicker(6 * time.Hour)
		for range ticker.C {
			fetchMoonPhaseWithRetry(3)
		}
	}()
}

// fetchMoonPhaseWithRetry attempts to fetch moon phase with exponential backoff
func fetchMoonPhaseWithRetry(maxRetries int) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fetchMoonPhaseFromAPI()
		if err == nil {
			return // Success
		}

		if attempt < maxRetries {
			backoff := time.Duration(attempt*attempt) * 30 * time.Second
			log.Printf("🌙 Moon phase fetch failed (attempt %d/%d), retrying in %s: %v",
				attempt, maxRetries, backoff, err)
			time.Sleep(backoff)
		} else {
			log.Printf("🌙 Moon phase fetch failed after %d attempts, using calculated fallback: %v",
				maxRetries, err)
			// Use calculated fallback
			useFallbackMoonPhase()
		}
	}
}

// fetchMoonPhaseFromAPI fetches moon phase data from Astronomy API
func fetchMoonPhaseFromAPI() error {
	if moonPhaseAPIKey == "" {
		useFallbackMoonPhase()
		return nil // Not an error, just use fallback
	}

	// Get current coordinates
	mutex.Lock()
	lat := cityLat
	lng := cityLng
	mutex.Unlock()

	// Get current date/time in configured timezone
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05")

	// Build API URL
	url := fmt.Sprintf(
		"https://api.astronomyapi.com/api/v2/bodies/positions/moon?latitude=%.4f&longitude=%.4f&elevation=0&from_date=%s&to_date=%s&time=%s",
		lat, lng, dateStr, dateStr, timeStr,
	)

	// Create request with Basic Auth
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// The API key from .env should already be base64 encoded (as provided by astronomyapi.com)
	req.Header.Set("Authorization", "Basic "+moonPhaseAPIKey)

	// Execute request with timeout
	client := &http.Client{Timeout: 15 * time.Second}
	log.Printf("🌙 Calling Astronomy API: %s", url)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("🌙 API response status: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp AstronomyAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("🌙 API response decoded successfully")

	// Extract data from response
	if len(apiResp.Data.Table.Rows) == 0 || len(apiResp.Data.Table.Rows[0].Cells) == 0 {
		return fmt.Errorf("empty response from API")
	}

	cell := apiResp.Data.Table.Rows[0].Cells[0]

	// Parse illumination fraction
	illumination, _ := strconv.ParseFloat(cell.ExtraInfo.Phase.Fraction, 64)

	// Parse phase angle
	phaseAngle, _ := strconv.ParseFloat(cell.ExtraInfo.Phase.Angle, 64)

	// Update global state
	mutex.Lock()
	moonPhaseData = MoonPhaseData{
		PhaseName:     cell.ExtraInfo.Phase.String,
		PhaseAngle:    phaseAngle,
		Illumination:  illumination,
		Constellation: cell.Position.Constellation.Name,
		DistanceKM:    cell.Distance.FromEarth.KM,
		FetchedAt:     time.Now().Format(time.RFC3339),
	}
	moonPhaseLastFetch = time.Now()
	mutex.Unlock()

	// Persist to config.json
	go saveConfig()

	log.Printf("🌙 Moon phase fetched: %s (%.0f%% illuminated) in %s",
		cell.ExtraInfo.Phase.String,
		illumination*100,
		cell.Position.Constellation.Name)

	return nil
}

// useFallbackMoonPhase uses mathematical calculation when API is unavailable
func useFallbackMoonPhase() {
	phase := calculateMoonPhase()
	phaseName := getMoonPhaseName(phase)
	illumination := float64(calculateIllumination(phase)) / 100.0

	mutex.Lock()
	moonPhaseData = MoonPhaseData{
		PhaseName:     phaseName,
		PhaseAngle:    float64(phase) * 45.0, // Approximate
		Illumination:  illumination,
		Constellation: "", // Not available without API
		DistanceKM:    "", // Not available without API
		FetchedAt:     time.Now().Format(time.RFC3339),
	}
	moonPhaseLastFetch = time.Now()
	mutex.Unlock()

	log.Printf("🌙 Using calculated moon phase: %s (~%.0f%% illuminated)",
		phaseName, illumination*100)
}

// Synodic month (average lunar cycle) in days
const synodicMonth = 29.53058867

// calculateMoonPhase returns the current moon phase (0-7) based on the current date
func calculateMoonPhase() int {
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}

	// Reference new moon: January 6, 2000 at 18:14 UTC
	referenceNewMoon := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

	// Calculate days since reference new moon
	daysSince := now.Sub(referenceNewMoon).Hours() / 24.0

	// Normalize to current lunar cycle position (0.0 to 1.0)
	cyclePosition := math.Mod(daysSince, synodicMonth) / synodicMonth
	if cyclePosition < 0 {
		cyclePosition += 1.0
	}

	// Convert to phase index (0-7)
	phase := int(cyclePosition * 8.0)
	if phase > 7 {
		phase = 7
	}

	return phase
}

// getMoonPhaseName returns the human-readable name for a moon phase
func getMoonPhaseName(phase int) string {
	names := []string{
		"New Moon",
		"Waxing Crescent",
		"First Quarter",
		"Waxing Gibbous",
		"Full Moon",
		"Waning Gibbous",
		"Last Quarter",
		"Waning Crescent",
	}
	if phase >= 0 && phase < len(names) {
		return names[phase]
	}
	return "Unknown"
}

// calculateIllumination returns the approximate illumination percentage for a phase
func calculateIllumination(phase int) int {
	illuminations := []int{0, 25, 50, 75, 100, 75, 50, 25}
	if phase >= 0 && phase < len(illuminations) {
		return illuminations[phase]
	}
	return 0
}

// getPhaseIndex returns 0-7 phase index from phase name
func getPhaseIndex(phaseName string) int {
	phases := map[string]int{
		"New Moon":        0,
		"Waxing Crescent": 1,
		"First Quarter":   2,
		"Waxing Gibbous":  3,
		"Full Moon":       4,
		"Waning Gibbous":  5,
		"Third Quarter":   6, // API sometimes uses "Third" instead of "Last"
		"Last Quarter":    6,
		"Waning Crescent": 7,
	}
	if idx, ok := phases[phaseName]; ok {
		return idx
	}
	return 0
}

// generateMoonBitmap creates a 48x48 bitmap of the moon showing the current phase
func generateMoonBitmap(illumination float64, phaseName string) ([]int, int, int) {
	const size = 48
	const radius = 22
	const centerX = size / 2
	const centerY = size / 2

	bytesPerRow := (size + 7) / 8
	bitmap := make([]int, bytesPerRow*size)

	// Helper function to set a pixel
	setPixel := func(x, y int) {
		if x < 0 || x >= size || y < 0 || y >= size {
			return
		}
		byteIndex := y*bytesPerRow + x/8
		if byteIndex < len(bitmap) {
			bitmap[byteIndex] |= (0x80 >> (x % 8))
		}
	}

	// Determine if waxing (right side lit) or waning (left side lit)
	isWaxing := phaseName == "Waxing Crescent" || phaseName == "Waxing Gibbous" ||
		phaseName == "First Quarter" || phaseName == "New Moon"

	// Draw the moon
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)

			// Check if inside circle
			if dist > float64(radius) {
				continue
			}

			// Normalize x position (-1 to 1)
			normalizedX := dx / float64(radius)

			// Determine if this pixel should be illuminated
			var shouldLight bool

			if illumination <= 0.02 {
				// New Moon - just outline
				shouldLight = dist >= float64(radius-1)
			} else if illumination >= 0.98 {
				// Full Moon - fill entire circle
				shouldLight = true
			} else {
				// Partial illumination
				// Calculate terminator position based on illumination
				terminator := (illumination - 0.5) * 2.0 // -1 to 1

				if isWaxing {
					// Right side lights first
					shouldLight = normalizedX >= -terminator
				} else {
					// Left side remains lit
					shouldLight = normalizedX <= terminator
				}
			}

			if shouldLight {
				setPixel(x, y)
			}
		}
	}

	// Draw outline for visibility
	for angle := 0.0; angle < 360.0; angle += 1.0 {
		rad := angle * math.Pi / 180.0
		x := int(float64(centerX) + float64(radius)*math.Cos(rad))
		y := int(float64(centerY) + float64(radius)*math.Sin(rad))
		setPixel(x, y)
	}

	return bitmap, size, size
}

// generateMoonPhaseFrame generates a complete frame for the moon phase display
// NOTE: This function assumes the global mutex is ALREADY HELD by the caller (updateLoop)
func generateMoonPhaseFrame(duration int) Frame {
	// Access global data directly (mutex is held)
	data := moonPhaseData

	// If no data, calculate fallback immediately
	// We update the global variable directly since we hold the lock
	if data.PhaseName == "" {
		phase := calculateMoonPhase()
		phaseName := getMoonPhaseName(phase)
		illumination := float64(calculateIllumination(phase)) / 100.0

		moonPhaseData = MoonPhaseData{
			PhaseName:     phaseName,
			PhaseAngle:    float64(phase) * 45.0, // Approximate
			Illumination:  illumination,
			Constellation: "", // Not available without API
			DistanceKM:    "", // Not available without API
			FetchedAt:     time.Now().Format(time.RFC3339),
		}
		data = moonPhaseData // Use the new data for this frame
	}

	// Generate bitmap
	bitmap, width, height := generateMoonBitmap(data.Illumination, data.PhaseName)

	// Calculate positions
	bitmapX := (128 - width) / 2
	bitmapY := 2

	// Format illumination percentage
	illuminationPct := int(data.Illumination * 100)
	illuminationStr := fmt.Sprintf("%d%%", illuminationPct)

	elements := []Element{
		// Moon bitmap
		{Type: "bitmap", X: bitmapX, Y: bitmapY, Width: width, Height: height, Bitmap: bitmap},
	}

	if showHeaders {
		// Combine phase name and illumination on one line (small text, size 1)
		displayStr := fmt.Sprintf("%s %s", data.PhaseName, illuminationStr)
		elements = append(elements, Element{
			Type:  "text",
			X:     calcCenteredX(displayStr, 1),
			Y:     52,
			Size:  1,
			Value: displayStr,
		})
		// Constellation if available (small text)
		if data.Constellation != "" {
			constStr := "in " + data.Constellation
			elements = append(elements, Element{
				Type:  "text",
				X:     calcCenteredX(constStr, 1),
				Y:     60,
				Size:  1,
				Value: constStr,
			})
		}
	}
	// When headers are off, show only the moon bitmap (no text)

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}
}
