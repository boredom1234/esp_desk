package main

import (
	"time"
)











func timeToBCD(h, m, s int, excludeSeconds bool) [][4]bool {
	var digits []int
	if excludeSeconds {
		digits = []int{
			h / 10, h % 10, 
			m / 10, m % 10, 
		}
	} else {
		digits = []int{
			h / 10, h % 10, 
			m / 10, m % 10, 
			s / 10, s % 10, 
		}
	}

	bcd := make([][4]bool, len(digits))
	for i, digit := range digits {
		bcd[i][0] = (digit & 8) != 0 
		bcd[i][1] = (digit & 4) != 0 
		bcd[i][2] = (digit & 2) != 0 
		bcd[i][3] = (digit & 1) != 0 
	}
	return bcd
}




func generateBCDFrame(duration int) Frame {
	
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	h, m, s := now.Hour(), now.Minute(), now.Second()

	
	if !bcd24HourMode {
		if h == 0 {
			h = 12
		} else if h > 12 {
			h = h - 12
		}
	}

	
	bcd := timeToBCD(h, m, s, !bcdShowSeconds)
	numCols := len(bcd)

	
	const (
		screenWidth  = 128
		screenHeight = 64

		
		ledRadius   = 5 
		ledDiameter = ledRadius * 2
		ledSpacing  = 3 

		
		numRows  = 4  
		groupGap = 10 
	)

	
	numGroupGaps := 1 
	if numCols == 6 {
		numGroupGaps = 2 
	}

	
	totalLedWidth := numCols*ledDiameter + (numCols-1)*ledSpacing + numGroupGaps*groupGap
	startX := (screenWidth - totalLedWidth) / 2

	
	totalLedHeight := numRows*ledDiameter + (numRows-1)*ledSpacing
	startY := (screenHeight - totalLedHeight) / 2

	
	if showHeaders {
		headerOffset := 14 
		startY = headerOffset + (screenHeight-headerOffset-totalLedHeight)/2
	}

	elements := []Element{}

	
	if showHeaders {
		headerText := "= BCD CLOCK ="
		headerSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(headerText, headerSize), Y: 2, Size: headerSize, Value: headerText},
			Element{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
		)
	}

	
	colX := make([]int, numCols)
	currentX := startX
	for col := 0; col < numCols; col++ {
		colX[col] = currentX
		currentX += ledDiameter + ledSpacing
		
		if col == 1 || (col == 3 && numCols == 6) {
			currentX += groupGap
		}
	}

	
	for col := 0; col < numCols; col++ {
		for row := 0; row < numRows; row++ {
			cx := colX[col] + ledRadius
			cy := startY + row*(ledDiameter+ledSpacing) + ledRadius

			isOn := bcd[col][row]

			if isOn {
				
				elements = append(elements, drawFilledCircle(cx, cy, ledRadius)...)
			} else {
				
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



func drawFilledCircle(cx, cy, r int) []Element {
	elements := []Element{}

	
	for dy := -r; dy <= r; dy++ {
		
		dySq := dy * dy
		rSq := r * r
		if dySq > rSq {
			continue
		}

		
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



func drawHollowCircle(cx, cy, r int) []Element {
	elements := []Element{}

	
	if r <= 2 {
		
		elements = append(elements,
			Element{Type: "line", X: cx, Y: cy - r, Width: 1, Height: 1},         
			Element{Type: "line", X: cx, Y: cy + r, Width: 1, Height: 1},         
			Element{Type: "line", X: cx - r, Y: cy, Width: 1, Height: 1},         
			Element{Type: "line", X: cx + r, Y: cy, Width: 1, Height: 1},         
			Element{Type: "line", X: cx - 1, Y: cy - r + 1, Width: 1, Height: 1}, 
			Element{Type: "line", X: cx + 1, Y: cy - r + 1, Width: 1, Height: 1}, 
			Element{Type: "line", X: cx - 1, Y: cy + r - 1, Width: 1, Height: 1}, 
			Element{Type: "line", X: cx + 1, Y: cy + r - 1, Width: 1, Height: 1}, 
		)
		return elements
	}

	
	x := r
	y := 0
	err := 1 - r

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
