package main

import (
	"math"
	"time"
)

// ==========================================
// ANALOG CLOCK
// ==========================================
// Classic analog clock face with hour, minute, and optional second hands.
// Supports Roman numerals (XII, III, VI, IX) or simple markers.

// generateAnalogFrame creates a frame displaying an analog clock face
func generateAnalogFrame(duration int) Frame {
	// Get current time with timezone
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	h, m, s := now.Hour(), now.Minute(), now.Second()

	// Clock dimensions (128x64 OLED)
	const (
		screenWidth  = 128
		screenHeight = 64
	)

	// Center the clock on the display
	clockRadius := 28           // Slightly smaller than half of 64
	centerX := screenWidth / 2  // Center horizontally
	centerY := screenHeight / 2 // Center vertically

	// Adjust for headers if enabled
	if showHeaders {
		centerY = 12 + (screenHeight-12)/2
	}

	elements := []Element{}

	// Add header if enabled
	if showHeaders {
		headerText := "= CLOCK ="
		headerSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(headerText, headerSize), Y: 2, Size: headerSize, Value: headerText},
			Element{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
		)
	}

	// Draw clock face (circle outline)
	elements = append(elements, drawClockCircle(centerX, centerY, clockRadius)...)

	// Draw hour markers or Roman numerals
	if analogShowRoman {
		elements = append(elements, drawRomanNumerals(centerX, centerY, clockRadius)...)
	} else {
		elements = append(elements, drawHourMarkers(centerX, centerY, clockRadius)...)
	}

	// Draw center dot
	elements = append(elements, Element{Type: "line", X: centerX - 1, Y: centerY - 1, Width: 3, Height: 3})

	// Calculate hand angles (clock math: 12 o'clock = -90 degrees, clockwise positive)
	// Hour hand: 360/12 = 30 degrees per hour, plus minute contribution
	hourAngle := float64(h%12)*30 + float64(m)*0.5 - 90
	// Minute hand: 360/60 = 6 degrees per minute
	minuteAngle := float64(m)*6 + float64(s)*0.1 - 90
	// Second hand: 360/60 = 6 degrees per second
	secondAngle := float64(s)*6 - 90

	// Draw hour hand (shorter, thicker)
	hourLength := int(float64(clockRadius) * 0.5)
	elements = append(elements, drawClockHand(centerX, centerY, hourAngle, hourLength, 3)...)

	// Draw minute hand (longer, medium thickness)
	minuteLength := int(float64(clockRadius) * 0.75)
	elements = append(elements, drawClockHand(centerX, centerY, minuteAngle, minuteLength, 2)...)

	// Draw second hand if enabled (longest, thin)
	if analogShowSeconds {
		secondLength := int(float64(clockRadius) * 0.85)
		elements = append(elements, drawClockHand(centerX, centerY, secondAngle, secondLength, 1)...)
	}

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}
}

// drawClockCircle draws the outline of the clock face
func drawClockCircle(cx, cy, radius int) []Element {
	elements := []Element{}

	// Use midpoint circle algorithm for accurate circle
	x := radius
	y := 0
	err := 1 - radius

	for x >= y {
		// Draw 8 symmetric points
		points := [][2]int{
			{cx + x, cy + y}, {cx - x, cy + y},
			{cx + x, cy - y}, {cx - x, cy - y},
			{cx + y, cy + x}, {cx - y, cy + x},
			{cx + y, cy - x}, {cx - y, cy - x},
		}
		for _, p := range points {
			elements = append(elements, Element{
				Type:   "line",
				X:      p[0],
				Y:      p[1],
				Width:  1,
				Height: 1,
			})
		}

		y++
		if err < 0 {
			err += 2*y + 1
		} else {
			x--
			err += 2*(y-x) + 1
		}
	}

	return elements
}

// drawHourMarkers draws simple tick marks at each hour position
func drawHourMarkers(cx, cy, radius int) []Element {
	elements := []Element{}

	for hour := 0; hour < 12; hour++ {
		angle := float64(hour)*30 - 90 // 30 degrees per hour, starting at 12 o'clock
		radians := angle * math.Pi / 180

		// Calculate marker positions (inner and outer)
		innerRadius := float64(radius) - 4
		outerRadius := float64(radius) - 1

		x1 := cx + int(innerRadius*math.Cos(radians))
		y1 := cy + int(innerRadius*math.Sin(radians))
		x2 := cx + int(outerRadius*math.Cos(radians))
		y2 := cy + int(outerRadius*math.Sin(radians))

		// Draw marker as a line
		elements = append(elements, drawLine(x1, y1, x2, y2)...)
	}

	return elements
}

// drawRomanNumerals draws XII, III, VI, IX at cardinal positions
func drawRomanNumerals(cx, cy, radius int) []Element {
	elements := []Element{}

	// Roman numerals at 12, 3, 6, 9 o'clock positions
	// Positioned inside the clock face
	textRadius := float64(radius) - 8

	// 12 o'clock (XII) - top
	elements = append(elements, Element{
		Type:  "text",
		X:     cx - 9, // Approximate centering for "XII"
		Y:     cy - int(textRadius) - 2,
		Size:  1,
		Value: "12",
	})

	// 3 o'clock (III) - right
	elements = append(elements, Element{
		Type:  "text",
		X:     cx + int(textRadius) - 2,
		Y:     cy - 3,
		Size:  1,
		Value: "3",
	})

	// 6 o'clock (VI) - bottom
	elements = append(elements, Element{
		Type:  "text",
		X:     cx - 3,
		Y:     cy + int(textRadius) - 6,
		Size:  1,
		Value: "6",
	})

	// 9 o'clock (IX) - left
	elements = append(elements, Element{
		Type:  "text",
		X:     cx - int(textRadius) - 2,
		Y:     cy - 3,
		Size:  1,
		Value: "9",
	})

	// Add small tick marks for other hours
	for hour := 0; hour < 12; hour++ {
		// Skip cardinal positions (12, 3, 6, 9)
		if hour == 0 || hour == 3 || hour == 6 || hour == 9 {
			continue
		}

		angle := float64(hour)*30 - 90
		radians := angle * math.Pi / 180

		innerRadius := float64(radius) - 3
		outerRadius := float64(radius) - 1

		x1 := cx + int(innerRadius*math.Cos(radians))
		y1 := cy + int(innerRadius*math.Sin(radians))
		x2 := cx + int(outerRadius*math.Cos(radians))
		y2 := cy + int(outerRadius*math.Sin(radians))

		elements = append(elements, drawLine(x1, y1, x2, y2)...)
	}

	return elements
}

// drawClockHand draws a clock hand from center to the specified angle
func drawClockHand(cx, cy int, angleDegrees float64, length, thickness int) []Element {
	radians := angleDegrees * math.Pi / 180

	// Calculate end point
	endX := cx + int(float64(length)*math.Cos(radians))
	endY := cy + int(float64(length)*math.Sin(radians))

	// Draw the hand as a line with thickness
	elements := []Element{}

	if thickness <= 1 {
		// Thin hand - just draw a line
		elements = append(elements, drawLine(cx, cy, endX, endY)...)
	} else {
		// Thicker hand - draw multiple parallel lines
		perpAngle := radians + math.Pi/2
		halfThickness := float64(thickness-1) / 2

		for t := -halfThickness; t <= halfThickness; t++ {
			offsetX := int(t * math.Cos(perpAngle))
			offsetY := int(t * math.Sin(perpAngle))

			x1 := cx + offsetX
			y1 := cy + offsetY
			x2 := endX + offsetX
			y2 := endY + offsetY

			elements = append(elements, drawLine(x1, y1, x2, y2)...)
		}
	}

	return elements
}

// drawLine draws a line between two points using Bresenham's algorithm
func drawLine(x1, y1, x2, y2 int) []Element {
	elements := []Element{}

	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := 1
	sy := 1

	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}

	err := dx - dy

	for {
		elements = append(elements, Element{
			Type:   "line",
			X:      x1,
			Y:      y1,
			Width:  1,
			Height: 1,
		})

		if x1 == x2 && y1 == y2 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}

	return elements
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
