package main

import (
	"time"
)



var wordClockGrid = []string{
	"ITTISITWENTY", 
	"QUARTERHALFM", 
	"TENFIVEEPAST", 
	"TOATWELVEONE", 
	"TWOTHREEFOUR", 
	"FIVESIXSEVEN", 
	"EIGHTNINETEN", 
	"ELEVENNDDATE", 
	"TO'CLOCKIMEA", 
}



type WordPosition struct {
	Row      int
	StartCol int
	EndCol   int
}


var wordMapping = map[string][]WordPosition{
	
	"IT": {{0, 0, 1}},
	"IS": {{0, 3, 4}},

	
	"M_TWENTY":  {{0, 6, 11}},
	"M_QUARTER": {{1, 0, 6}},
	"M_HALF":    {{1, 7, 10}},
	"M_TEN":     {{2, 0, 2}},
	"M_FIVE":    {{2, 3, 6}},

	
	"PAST": {{2, 8, 11}},
	"TO":   {{3, 0, 1}},
	"A":    {{3, 2, 2}},

	
	"H_TWELVE": {{3, 3, 8}},
	"H_ONE":    {{3, 9, 11}},
	"H_TWO":    {{4, 0, 2}},
	"H_THREE":  {{4, 3, 7}},
	"H_FOUR":   {{4, 8, 11}},
	"H_FIVE":   {{5, 0, 3}},
	"H_SIX":    {{5, 4, 6}},
	"H_SEVEN":  {{5, 7, 11}},
	"H_EIGHT":  {{6, 0, 4}},
	"H_NINE":   {{6, 5, 8}},
	"H_TEN":    {{6, 9, 11}},
	"H_ELEVEN": {{7, 0, 5}},

	
	"OCLOCK": {{8, 1, 7}},
}


var hourNames = []string{
	"", "H_ONE", "H_TWO", "H_THREE", "H_FOUR", "H_FIVE", "H_SIX",
	"H_SEVEN", "H_EIGHT", "H_NINE", "H_TEN", "H_ELEVEN", "H_TWELVE",
}


func getActiveWords(hours, minutes int) []string {
	activeWords := []string{"IT", "IS"}

	
	displayHour := hours % 12
	if displayHour == 0 {
		displayHour = 12
	}

	
	if minutes >= 5 && minutes < 35 {
		activeWords = append(activeWords, "PAST")
	}
	if minutes >= 35 {
		activeWords = append(activeWords, "TO")
		displayHour = (displayHour % 12) + 1 
	}

	
	switch {
	case minutes >= 5 && minutes < 10:
		activeWords = append(activeWords, "M_FIVE")
	case minutes >= 10 && minutes < 15:
		activeWords = append(activeWords, "M_TEN")
	case minutes >= 15 && minutes < 20:
		activeWords = append(activeWords, "A", "M_QUARTER")
	case minutes >= 20 && minutes < 25:
		activeWords = append(activeWords, "M_TWENTY")
	case minutes >= 25 && minutes < 30:
		activeWords = append(activeWords, "M_TWENTY", "M_FIVE")
	case minutes >= 30 && minutes < 35:
		activeWords = append(activeWords, "M_HALF")
	case minutes >= 35 && minutes < 40:
		activeWords = append(activeWords, "M_TWENTY", "M_FIVE")
	case minutes >= 40 && minutes < 45:
		activeWords = append(activeWords, "M_TWENTY")
	case minutes >= 45 && minutes < 50:
		activeWords = append(activeWords, "A", "M_QUARTER")
	case minutes >= 50 && minutes < 55:
		activeWords = append(activeWords, "M_TEN")
	case minutes >= 55:
		activeWords = append(activeWords, "M_FIVE")
	default:
		
		activeWords = append(activeWords, "OCLOCK")
	}

	
	activeWords = append(activeWords, hourNames[displayHour])

	return activeWords
}


func generateWordClockFrame(duration int) Frame {
	
	now := time.Now()
	if displayLocation != nil {
		now = now.In(displayLocation)
	}
	hours := now.Hour()
	minutes := now.Minute()

	
	hours12 := hours % 12
	if hours12 == 0 {
		hours12 = 12
	}

	
	activeWords := getActiveWords(hours12, minutes)

	
	activePositions := make(map[int]map[int]bool)
	for row := 0; row < len(wordClockGrid); row++ {
		activePositions[row] = make(map[int]bool)
	}

	
	for _, word := range activeWords {
		if positions, ok := wordMapping[word]; ok {
			for _, pos := range positions {
				for col := pos.StartCol; col <= pos.EndCol; col++ {
					activePositions[pos.Row][col] = true
				}
			}
		}
	}

	
	const (
		screenWidth  = 128
		screenHeight = 64
		gridRows     = 9
		gridCols     = 12
	)

	
	
	charWidth := 6  
	rowSpacing := 7 
	textSize := 1

	
	totalGridWidth := gridCols * charWidth * textSize
	totalGridHeight := gridRows * rowSpacing

	
	startX := (screenWidth - totalGridWidth) / 2
	startY := (screenHeight - totalGridHeight) / 2

	
	if showHeaders {
		
		rowSpacing = 6
		totalGridHeight = gridRows * rowSpacing
		startY = 14 + (screenHeight-14-totalGridHeight)/2
	}

	elements := []Element{}

	
	if showHeaders {
		headerText := "= WORD CLOCK ="
		headerSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(headerText, headerSize), Y: 2, Size: headerSize, Value: headerText},
			Element{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
		)
	}

	
	
	
	for row := 0; row < gridRows; row++ {
		if row >= len(wordClockGrid) {
			break
		}
		rowStr := wordClockGrid[row]
		for col := 0; col < gridCols && col < len(rowStr); col++ {
			if activePositions[row][col] {
				x := startX + col*charWidth*textSize
				y := startY + row*rowSpacing
				letter := string(rowStr[col])
				elements = append(elements, Element{
					Type:  "text",
					X:     x,
					Y:     y,
					Size:  textSize,
					Value: letter,
				})
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
