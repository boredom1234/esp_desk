package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)



func initMoonPhase() {
	
	log.Println("🌙 Moon phase utilizing web scraping (timeanddate.com)")
}

func handleMoonPhaseRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	log.Println("🌙 Manual moon phase refresh triggered")

	
	err := fetchMoonPhaseFromWeb()

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
		"source":        "web",
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
		err := fetchMoonPhaseFromWeb()
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


func fetchMoonPhaseFromWeb() error {
	mutex.Lock()
	
	url := "https://www.timeanddate.com/moon/phases/usa/new-york"
	mutex.Unlock()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml")

	log.Printf("🌙 Scraping Moon Phase from: %s", url)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %d", resp.StatusCode)
	}

	
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body: %w", err)
	}
	body := string(bodyBytes)

	
	
	reIllum := regexp.MustCompile(`id=cur-moon-percent>([\d\.]+)%</span>`)
	illumMatches := reIllum.FindStringSubmatch(body)
	var illumVal float64
	if len(illumMatches) >= 2 {
		illumStr := illumMatches[1]
		val, err := strconv.ParseFloat(illumStr, 64)
		if err == nil {
			illumVal = val
		} else {
			log.Printf("⚠️ Failed to parse illumination '%s': %v", illumStr, err)
		}
	} else {
		return fmt.Errorf("could not find illumination percentage")
	}

	
	
	
	rePhase := regexp.MustCompile(`(?s)<div id=qlook[^>]*>.*?<a[^>]+>([^<]+)</a>`)
	phaseMatches := rePhase.FindStringSubmatch(body)

	var phaseName string
	if len(phaseMatches) >= 2 {
		phaseName = phaseMatches[1]
	} else {
		
		log.Println("⚠️ Could not parse phase name from HTML")
		phaseName = "Unknown"
	}

	phaseName = strings.TrimSpace(phaseName)

	log.Printf("🌙 Scraped Data - Phase: %s, Illum: %.1f%%", phaseName, illumVal)

	mutex.Lock()
	moonPhaseData = MoonPhaseData{
		PhaseName:     phaseName,
		PhaseAngle:    0, 
		Illumination:  illumVal / 100.0,
		Constellation: "",
		DistanceKM:    "",
		FetchedAt:     time.Now().Format(time.RFC3339),
	}
	moonPhaseLastFetch = time.Now()
	mutex.Unlock()

	go saveConfig()

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
