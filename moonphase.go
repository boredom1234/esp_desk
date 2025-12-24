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






type AstronomyAPIResponse struct {
	Data struct {
		Table struct {
			Rows []struct {
				Cells []struct {
					ExtraInfo struct {
						Phase struct {
							Angle    string `json:"angel"` 
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


func initMoonPhase() {
	moonPhaseAPIKey = os.Getenv("ASTRONOMY_API_KEY")
	if moonPhaseAPIKey != "" {
		log.Println("🌙 Astronomy API key configured")
	} else {
		log.Println("🌙 ASTRONOMY_API_KEY not set, using calculated moon phase")
	}
}


func handleMoonPhaseRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	log.Println("🌙 Manual moon phase refresh triggered")

	
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



func startMoonPhaseFetcher() {
	go func() {
		
		mutex.Lock()
		lastFetch := moonPhaseLastFetch
		hasData := moonPhaseData.PhaseName != ""
		mutex.Unlock()

		if hasData && time.Since(lastFetch) < 6*time.Hour {
			log.Printf("🌙 Using cached moon phase data (fetched %s ago)", time.Since(lastFetch).Round(time.Minute))
		} else {
			
			fetchMoonPhaseWithRetry(3)
		}

		
		ticker := time.NewTicker(6 * time.Hour)
		for range ticker.C {
			fetchMoonPhaseWithRetry(3)
		}
	}()
}


func fetchMoonPhaseWithRetry(maxRetries int) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := fetchMoonPhaseFromAPI()
		if err == nil {
			return 
		}

		if attempt < maxRetries {
			backoff := time.Duration(attempt*attempt) * 30 * time.Second
			log.Printf("🌙 Moon phase fetch failed (attempt %d/%d), retrying in %s: %v",
				attempt, maxRetries, backoff, err)
			time.Sleep(backoff)
		} else {
			log.Printf("🌙 Moon phase fetch failed after %d attempts, using calculated fallback: %v",
				maxRetries, err)
			
			useFallbackMoonPhase()
		}
	}
}


func fetchMoonPhaseFromAPI() error {
	if moonPhaseAPIKey == "" {
		useFallbackMoonPhase()
		return nil 
	}

	
	mutex.Lock()
	lat := cityLat
	lng := cityLng
	mutex.Unlock()

	
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	dateStr := now.Format("2006-01-02")
	timeStr := now.Format("15:04:05")

	
	url := fmt.Sprintf(
		"https://api.astronomyapi.com/api/v2/bodies/positions/moon?latitude=%.4f&longitude=%.4f&elevation=0&from_date=%s&to_date=%s&time=%s",
		lat, lng, dateStr, dateStr, timeStr,
	)

	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	
	req.Header.Set("Authorization", "Basic "+moonPhaseAPIKey)

	
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

	
	var apiResp AstronomyAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("🌙 API response decoded successfully")

	
	if len(apiResp.Data.Table.Rows) == 0 || len(apiResp.Data.Table.Rows[0].Cells) == 0 {
		return fmt.Errorf("empty response from API")
	}

	cell := apiResp.Data.Table.Rows[0].Cells[0]

	
	illumination, _ := strconv.ParseFloat(cell.ExtraInfo.Phase.Fraction, 64)

	
	phaseAngle, _ := strconv.ParseFloat(cell.ExtraInfo.Phase.Angle, 64)

	
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

	
	go saveConfig()

	log.Printf("🌙 Moon phase fetched: %s (%.0f%% illuminated) in %s",
		cell.ExtraInfo.Phase.String,
		illumination*100,
		cell.Position.Constellation.Name)

	return nil
}


func useFallbackMoonPhase() {
	phase := calculateMoonPhase()
	phaseName := getMoonPhaseName(phase)
	illumination := float64(calculateIllumination(phase)) / 100.0

	mutex.Lock()
	moonPhaseData = MoonPhaseData{
		PhaseName:     phaseName,
		PhaseAngle:    float64(phase) * 45.0, 
		Illumination:  illumination,
		Constellation: "", 
		DistanceKM:    "", 
		FetchedAt:     time.Now().Format(time.RFC3339),
	}
	moonPhaseLastFetch = time.Now()
	mutex.Unlock()

	log.Printf("🌙 Using calculated moon phase: %s (~%.0f%% illuminated)",
		phaseName, illumination*100)
}


const synodicMonth = 29.53058867


func calculateMoonPhase() int {
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}

	
	referenceNewMoon := time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

	
	daysSince := now.Sub(referenceNewMoon).Hours() / 24.0

	
	cyclePosition := math.Mod(daysSince, synodicMonth) / synodicMonth
	if cyclePosition < 0 {
		cyclePosition += 1.0
	}

	
	phase := int(cyclePosition * 8.0)
	if phase > 7 {
		phase = 7
	}

	return phase
}


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


func calculateIllumination(phase int) int {
	illuminations := []int{0, 25, 50, 75, 100, 75, 50, 25}
	if phase >= 0 && phase < len(illuminations) {
		return illuminations[phase]
	}
	return 0
}


func getPhaseIndex(phaseName string) int {
	phases := map[string]int{
		"New Moon":        0,
		"Waxing Crescent": 1,
		"First Quarter":   2,
		"Waxing Gibbous":  3,
		"Full Moon":       4,
		"Waning Gibbous":  5,
		"Third Quarter":   6, 
		"Last Quarter":    6,
		"Waning Crescent": 7,
	}
	if idx, ok := phases[phaseName]; ok {
		return idx
	}
	return 0
}


func generateMoonBitmap(illumination float64, phaseName string) ([]int, int, int) {
	const size = 48
	const radius = 22
	const centerX = size / 2
	const centerY = size / 2

	bytesPerRow := (size + 7) / 8
	bitmap := make([]int, bytesPerRow*size)

	
	setPixel := func(x, y int) {
		if x < 0 || x >= size || y < 0 || y >= size {
			return
		}
		byteIndex := y*bytesPerRow + x/8
		if byteIndex < len(bitmap) {
			bitmap[byteIndex] |= (0x80 >> (x % 8))
		}
	}

	
	isWaxing := phaseName == "Waxing Crescent" || phaseName == "Waxing Gibbous" ||
		phaseName == "First Quarter" || phaseName == "New Moon"

	
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x - centerX)
			dy := float64(y - centerY)
			dist := math.Sqrt(dx*dx + dy*dy)

			
			if dist > float64(radius) {
				continue
			}

			
			normalizedX := dx / float64(radius)

			
			var shouldLight bool

			if illumination <= 0.02 {
				
				shouldLight = dist >= float64(radius-1)
			} else if illumination >= 0.98 {
				
				shouldLight = true
			} else {
				
				
				terminator := (illumination - 0.5) * 2.0 

				if isWaxing {
					
					shouldLight = normalizedX >= -terminator
				} else {
					
					shouldLight = normalizedX <= terminator
				}
			}

			if shouldLight {
				setPixel(x, y)
			}
		}
	}

	
	for angle := 0.0; angle < 360.0; angle += 1.0 {
		rad := angle * math.Pi / 180.0
		x := int(float64(centerX) + float64(radius)*math.Cos(rad))
		y := int(float64(centerY) + float64(radius)*math.Sin(rad))
		setPixel(x, y)
	}

	return bitmap, size, size
}



func generateMoonPhaseFrame(duration int) Frame {
	
	data := moonPhaseData

	
	
	if data.PhaseName == "" {
		phase := calculateMoonPhase()
		phaseName := getMoonPhaseName(phase)
		illumination := float64(calculateIllumination(phase)) / 100.0

		moonPhaseData = MoonPhaseData{
			PhaseName:     phaseName,
			PhaseAngle:    float64(phase) * 45.0, 
			Illumination:  illumination,
			Constellation: "", 
			DistanceKM:    "", 
			FetchedAt:     time.Now().Format(time.RFC3339),
		}
		data = moonPhaseData 
	}

	
	bitmap, width, height := generateMoonBitmap(data.Illumination, data.PhaseName)

	
	bitmapX := (128 - width) / 2
	bitmapY := 0 

	
	illuminationPct := int(data.Illumination * 100)
	illuminationStr := fmt.Sprintf("%d%%", illuminationPct)

	elements := []Element{
		
		{Type: "bitmap", X: bitmapX, Y: bitmapY, Width: width, Height: height, Bitmap: bitmap},
	}

	if showHeaders {
		
		displayStr := fmt.Sprintf("%s %s", data.PhaseName, illuminationStr)
		elements = append(elements, Element{
			Type:  "text",
			X:     calcCenteredX(displayStr, 1),
			Y:     50,
			Size:  1,
			Value: displayStr,
		})
	}
	

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}
}
