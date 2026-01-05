package main

import (
	"math"
	"time"
)

func generateAnalogFrame(duration int) Frame {

	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	h, m, s := now.Hour(), now.Minute(), now.Second()

	const (
		screenWidth  = 128
		screenHeight = 64
	)

	clockRadius := 28
	centerX := screenWidth / 2
	centerY := screenHeight / 2

	if showHeaders {
		centerY = 12 + (screenHeight-12)/2
	}

	elements := []Element{}

	if showHeaders {
		headerText := "= CLOCK ="
		headerSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(headerText, headerSize), Y: 2, Size: headerSize, Value: headerText},
			Element{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
		)
	}

	elements = append(elements, drawClockCircle(centerX, centerY, clockRadius)...)

	if analogShowRoman {
		elements = append(elements, drawRomanNumerals(centerX, centerY, clockRadius)...)
	} else {
		elements = append(elements, drawHourMarkers(centerX, centerY, clockRadius)...)
	}

	elements = append(elements, Element{Type: "line", X: centerX - 1, Y: centerY - 1, Width: 3, Height: 3})

	hourAngle := float64(h%12)*30 + float64(m)*0.5 - 90

	minuteAngle := float64(m)*6 + float64(s)*0.1 - 90

	secondAngle := float64(s)*6 - 90

	hourLength := int(float64(clockRadius) * 0.5)
	elements = append(elements, drawClockHand(centerX, centerY, hourAngle, hourLength, 3)...)

	minuteLength := int(float64(clockRadius) * 0.75)
	elements = append(elements, drawClockHand(centerX, centerY, minuteAngle, minuteLength, 2)...)

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

func drawClockCircle(cx, cy, radius int) []Element {
	elements := []Element{}

	x := radius
	y := 0
	err := 1 - radius

	for x >= y {

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

func drawHourMarkers(cx, cy, radius int) []Element {
	elements := []Element{}

	for hour := 0; hour < 12; hour++ {
		angle := float64(hour)*30 - 90
		radians := angle * math.Pi / 180

		innerRadius := float64(radius) - 4
		outerRadius := float64(radius) - 1

		x1 := cx + int(innerRadius*math.Cos(radians))
		y1 := cy + int(innerRadius*math.Sin(radians))
		x2 := cx + int(outerRadius*math.Cos(radians))
		y2 := cy + int(outerRadius*math.Sin(radians))

		elements = append(elements, drawLine(x1, y1, x2, y2)...)
	}

	return elements
}

func drawRomanNumerals(cx, cy, radius int) []Element {
	elements := []Element{}

	textRadius := float64(radius) - 8

	// 12
	elements = append(elements, Element{
		Type:  "text",
		X:     cx - 6,
		Y:     cy - int(textRadius) - 5,
		Size:  1,
		Value: "12",
	})

	// 3
	elements = append(elements, Element{
		Type:  "text",
		X:     cx + int(textRadius) - 3,
		Y:     cy - 5,
		Size:  1,
		Value: "3",
	})

	// 6
	elements = append(elements, Element{
		Type:  "text",
		X:     cx - 3,
		Y:     cy + int(textRadius) - 5,
		Size:  1,
		Value: "6",
	})

	// 9
	elements = append(elements, Element{
		Type:  "text",
		X:     cx - int(textRadius) - 3,
		Y:     cy - 5,
		Size:  1,
		Value: "9",
	})

	for hour := 0; hour < 12; hour++ {

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

func drawClockHand(cx, cy int, angleDegrees float64, length, thickness int) []Element {
	radians := angleDegrees * math.Pi / 180

	endX := cx + int(float64(length)*math.Cos(radians))
	endY := cy + int(float64(length)*math.Sin(radians))

	elements := []Element{}

	if thickness <= 1 {

		elements = append(elements, drawLine(cx, cy, endX, endY)...)
	} else {

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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
