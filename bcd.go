package main

import (
	"time"
)

// ==========================================
// BCD (Binary-Coded Decimal) CLOCK
// ==========================================
// Displays time as a visual binary matrix where each digit
// is represented by 4 LEDs showing its binary value (8-4-2-1)

// timeToBCD converts hour, minute, second to BCD representation.
// Returns [6][4]bool: each digit has 4 bits (MSB first: 8, 4, 2, 1)
// Order: [H1, H0, M1, M0, S1, S0] (e.g., 12:34:56 → [1,2,3,4,5,6])
// If excludeSeconds is true, only returns first 4 digits (HH:MM)
func timeToBCD(h, m, s int, excludeSeconds bool) [][4]bool {
	var digits []int
	if excludeSeconds {
		digits = []int{
			h / 10, h % 10, // Hours tens, ones
			m / 10, m % 10, // Minutes tens, ones
		}
	} else {
		digits = []int{
			h / 10, h % 10, // Hours tens, ones
			m / 10, m % 10, // Minutes tens, ones
			s / 10, s % 10, // Seconds tens, ones
		}
	}

	bcd := make([][4]bool, len(digits))
	for i, digit := range digits {
		bcd[i][0] = (digit & 8) != 0 // Bit 3 (weight 8)
		bcd[i][1] = (digit & 4) != 0 // Bit 2 (weight 4)
		bcd[i][2] = (digit & 2) != 0 // Bit 1 (weight 2)
		bcd[i][3] = (digit & 1) != 0 // Bit 0 (weight 1)
	}
	return bcd
}

// generateBCDFrame creates a frame displaying the current time as a BCD clock.
// Layout: 4 or 6 columns (HH:MM or HH:MM:SS) × 4 rows (bit weights 8, 4, 2, 1)
// LED ON = filled circle, LED OFF = hollow circle (small dot)
func generateBCDFrame(duration int) Frame {
	// Get current time with timezone
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	h, m, s := now.Hour(), now.Minute(), now.Second()

	// Apply 12-hour format if configured
	if !bcd24HourMode {
		if h == 0 {
			h = 12
		} else if h > 12 {
			h = h - 12
		}
	}

	// Get BCD representation (with or without seconds)
	bcd := timeToBCD(h, m, s, !bcdShowSeconds)
	numCols := len(bcd)

	// Display layout constants (128x64 OLED)
	const (
		screenWidth  = 128
		screenHeight = 64

		// LED dot dimensions - adjusted for OLED visibility
		ledRadius   = 5 // Radius of LED circle
		ledDiameter = ledRadius * 2
		ledSpacing  = 3 // Space between LEDs

		// Grid dimensions
		numRows  = 4  // Bit weights: 8, 4, 2, 1
		groupGap = 10 // Extra gap between digit pairs (HH : MM : SS)
	)

	// Calculate number of group gaps (between pairs: after col 1, and optionally after col 3)
	numGroupGaps := 1 // Always gap after hours
	if numCols == 6 {
		numGroupGaps = 2 // Also gap after minutes if showing seconds
	}

	// Calculate total width of the BCD grid
	totalLedWidth := numCols*ledDiameter + (numCols-1)*ledSpacing + numGroupGaps*groupGap
	startX := (screenWidth - totalLedWidth) / 2

	// Calculate total height
	totalLedHeight := numRows*ledDiameter + (numRows-1)*ledSpacing
	startY := (screenHeight - totalLedHeight) / 2

	// Adjust for header if enabled
	if showHeaders {
		headerOffset := 14 // Space for header text + line
		startY = headerOffset + (screenHeight-headerOffset-totalLedHeight)/2
	}

	elements := []Element{}

	// Add header if enabled
	if showHeaders {
		headerText := "= BCD CLOCK ="
		headerSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(headerText, headerSize), Y: 2, Size: headerSize, Value: headerText},
			Element{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
		)
	}

	// Column positions: account for group gaps after columns 1 and 3
	colX := make([]int, numCols)
	currentX := startX
	for col := 0; col < numCols; col++ {
		colX[col] = currentX
		currentX += ledDiameter + ledSpacing
		// Add group gap after H0 (col 1) and M0 (col 3 if seconds shown)
		if col == 1 || (col == 3 && numCols == 6) {
			currentX += groupGap
		}
	}

	// Draw each LED
	for col := 0; col < numCols; col++ {
		for row := 0; row < numRows; row++ {
			cx := colX[col] + ledRadius
			cy := startY + row*(ledDiameter+ledSpacing) + ledRadius

			isOn := bcd[col][row]

			if isOn {
				// LED ON: Draw filled circle using multiple rectangles
				elements = append(elements, drawFilledCircle(cx, cy, ledRadius)...)
			} else {
				// LED OFF: Draw hollow circle (outline only)
				elements = append(elements, drawHollowCircle(cx, cy, ledRadius)...)
			}
		}
	}

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}
}

// drawFilledCircle creates elements that approximate a filled circle.
// Uses overlapping horizontal lines for a pixelated circle effect on OLED.
func drawFilledCircle(cx, cy, r int) []Element {
	elements := []Element{}

	// Approximate circle with horizontal slices
	for dy := -r; dy <= r; dy++ {
		// Calculate width at this y-level using circle equation: x² + y² = r²
		dySq := dy * dy
		rSq := r * r
		if dySq > rSq {
			continue
		}

		// Integer approximation of sqrt(r² - y²)
		dx := 0
		for (dx+1)*(dx+1) <= rSq-dySq {
			dx++
		}

		if dx >= 0 {
			width := dx*2 + 1
			x := cx - dx
			y := cy + dy
			elements = append(elements, Element{
				Type:   "line",
				X:      x,
				Y:      y,
				Width:  width,
				Height: 1,
			})
		}
	}

	return elements
}

// drawHollowCircle creates elements that draw a circle outline.
// Uses the midpoint circle algorithm for accurate pixel placement.
func drawHollowCircle(cx, cy, r int) []Element {
	elements := []Element{}

	// For very small circles, draw a minimal outline
	if r <= 2 {
		// Draw a simple cross/diamond pattern
		elements = append(elements,
			Element{Type: "line", X: cx, Y: cy - r, Width: 1, Height: 1},         // Top
			Element{Type: "line", X: cx, Y: cy + r, Width: 1, Height: 1},         // Bottom
			Element{Type: "line", X: cx - r, Y: cy, Width: 1, Height: 1},         // Left
			Element{Type: "line", X: cx + r, Y: cy, Width: 1, Height: 1},         // Right
			Element{Type: "line", X: cx - 1, Y: cy - r + 1, Width: 1, Height: 1}, // Near top-left
			Element{Type: "line", X: cx + 1, Y: cy - r + 1, Width: 1, Height: 1}, // Near top-right
			Element{Type: "line", X: cx - 1, Y: cy + r - 1, Width: 1, Height: 1}, // Near bottom-left
			Element{Type: "line", X: cx + 1, Y: cy + r - 1, Width: 1, Height: 1}, // Near bottom-right
		)
		return elements
	}

	// Midpoint circle algorithm for larger circles
	x := r
	y := 0
	err := 1 - r

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
