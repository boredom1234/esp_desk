package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

func initMoonPhase() {
	log.Println("🌙 Moon phase utilizing internal calculation")
}

func handleMoonPhaseRefresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	log.Println("🌙 Manual moon phase refresh triggered")

	updateMoonPhaseData()

	mutex.Lock()
	data := moonPhaseData
	mutex.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"phaseName":     data.PhaseName,
		"illumination":  data.Illumination,
		"constellation": data.Constellation,
		"source":        "calculated",
	})
}

func startMoonPhaseFetcher() {

	updateMoonPhaseData()

	go func() {

		ticker := time.NewTicker(4 * time.Hour)
		for range ticker.C {
			updateMoonPhaseData()
		}
	}()
}

func updateMoonPhaseData() {
	cyclePos := calculateCyclePosition()
	phase := int(math.Round(cyclePos*8.0)) % 8
	phaseName := getMoonPhaseName(phase)
	illumination := calculatePreciseIllumination(cyclePos)

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

	log.Printf("🌙 Updated moon phase: %s (%.0f%% illuminated)", phaseName, illumination*100)

	saveConfig()
}

const synodicMonth = 29.53058867

func calculateCyclePosition() float64 {
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

	return cyclePosition
}

func calculatePreciseIllumination(cyclePos float64) float64 {
	return (1.0 - math.Cos(2.0*math.Pi*cyclePos)) / 2.0
}

func calculateMoonPhase() int {
	cyclePosition := calculateCyclePosition()

	phase := int(math.Round(cyclePosition*8.0)) % 8

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

func generateMoonPhaseFrame(duration int, data MoonPhaseData, headers bool) Frame {

	if data.PhaseName == "" {
		updateMoonPhaseData()
		mutex.Lock()
		data = moonPhaseData
		mutex.Unlock()
	}

	bitmap, width, height := generateMoonBitmap(data.Illumination, data.PhaseName)

	bitmapX := (128 - width) / 2
	bitmapY := 0

	illuminationPct := int(data.Illumination * 100)
	illuminationStr := fmt.Sprintf("%d%%", illuminationPct)

	elements := []Element{
		{Type: "bitmap", X: bitmapX, Y: bitmapY, Width: width, Height: height, Bitmap: bitmap},
	}

	if headers {
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
