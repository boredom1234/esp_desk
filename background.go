package main

import (
	"fmt"
	"log"
	"time"
)

// ==========================================
// BACKGROUND TASKS
// ==========================================

func updateLoop() {
	go func() {
		fetchWeather()
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			fetchWeather()
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		mutex.Lock()

		// Pomodoro timer countdown (runs every second)
		if pomodoroSession.Active && !pomodoroSession.IsPaused {
			if pomodoroSession.TimeRemaining > 0 {
				pomodoroSession.TimeRemaining--
			} else {
				// Timer reached 0 - auto-transition to next phase
				if pomodoroSession.Mode == "work" {
					pomodoroSession.CyclesCompleted++
					if pomodoroSession.CyclesCompleted >= pomodoroSettings.CyclesUntilLong {
						pomodoroSession.Mode = "longBreak"
						pomodoroSession.TimeRemaining = pomodoroSettings.LongBreak
						pomodoroSession.CyclesCompleted = 0
						log.Printf("🍅 Pomodoro: Auto-started long break (%d min)", pomodoroSettings.LongBreak/60)
					} else {
						pomodoroSession.Mode = "break"
						pomodoroSession.TimeRemaining = pomodoroSettings.BreakDuration
						log.Printf("🍅 Pomodoro: Auto-started break (%d min)", pomodoroSettings.BreakDuration/60)
					}
				} else {
					// Break ended - start new work session
					pomodoroSession.Mode = "work"
					pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
					log.Printf("🍅 Pomodoro: Auto-started work session (%d min)", pomodoroSettings.WorkDuration/60)
				}
				pomodoroSession.StartedAt = time.Now()
			}
		}

		if !isCustomMode {
			// Use configurable timezone for time display
			now := time.Now()
			if displayLocation != nil {
				now = now.In(displayLocation)
			}
			currentTime := now.Format("15:04:05")
			uptime := time.Since(startTime).Round(time.Second).String()

			// Build frame map for each type
			frameMap := make(map[string]Frame)

			// Time frame
			// Get timezone abbreviation from current time in selected location
			tzAbbrev, _ := now.Zone()
			timeMainSize := getScaledTextSize(2) // Main time text
			headerSize := getScaledTextSize(1)   // Headers/labels
			timeElements := []Element{
				{Type: "text", X: calcCenteredX(currentTime, timeMainSize), Y: 22, Size: timeMainSize, Value: currentTime},
			}
			if showHeaders {
				timeHeaderText := "= TIME ="
				timeElements = append([]Element{
					{Type: "text", X: calcCenteredX(timeHeaderText, headerSize), Y: 2, Size: headerSize, Value: timeHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, timeElements...)
				timeElements = append(timeElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				timeElements = append(timeElements, Element{Type: "text", X: calcCenteredX(tzAbbrev, headerSize), Y: 55, Size: headerSize, Value: tzAbbrev})
			}
			frameMap["time"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: timeElements}

			// Weather frame - now includes AQI
			// Build compact AQI display (just number, fits OLED width)
			aqiDisplay := ""
			if weatherData.AQI > 0 {
				aqiDisplay = fmt.Sprintf("AQI:%d", weatherData.AQI)
			}

			weatherMainSize := getScaledTextSize(2)
			weatherLabelSize := getScaledTextSize(1)
			weatherElements := []Element{
				{Type: "text", X: calcCenteredX(weatherData.Temperature, weatherMainSize), Y: 20, Size: weatherMainSize, Value: weatherData.Temperature},
			}
			if showHeaders {
				weatherHeaderText := "= WEATHER ="
				weatherElements = append([]Element{
					{Type: "text", X: calcCenteredX(weatherHeaderText, weatherLabelSize), Y: 2, Size: weatherLabelSize, Value: weatherHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, weatherElements...)
				// Condition and AQI on same line if both fit
				if aqiDisplay != "" {
					// Show condition + AQI side by side
					weatherElements = append(weatherElements, Element{Type: "text", X: 5, Y: 42, Size: weatherLabelSize, Value: weatherData.Condition})
					weatherElements = append(weatherElements, Element{Type: "text", X: 75, Y: 42, Size: weatherLabelSize, Value: aqiDisplay})
				} else {
					weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(weatherData.Condition, weatherLabelSize), Y: 42, Size: weatherLabelSize, Value: weatherData.Condition})
				}
				weatherElements = append(weatherElements, Element{Type: "line", X: 0, Y: 53, Width: 128, Height: 1})
				weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(weatherData.City, weatherLabelSize), Y: 56, Size: weatherLabelSize, Value: weatherData.City})
			} else {
				// Without headers, show compact info
				weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(weatherData.Condition, weatherLabelSize), Y: 42, Size: weatherLabelSize, Value: weatherData.Condition})
				if aqiDisplay != "" {
					weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(aqiDisplay, weatherLabelSize), Y: 52, Size: weatherLabelSize, Value: aqiDisplay})
				}
			}
			frameMap["weather"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: weatherElements}

			// Uptime frame
			uptimeSize := getScaledTextSize(1)
			uptimeElements := []Element{
				{Type: "text", X: calcCenteredX(uptime, uptimeSize), Y: 28, Size: uptimeSize, Value: uptime},
			}
			if showHeaders {
				uptimeHeaderText := "= UPTIME ="
				uptimeElements = append([]Element{
					{Type: "text", X: calcCenteredX(uptimeHeaderText, headerSize), Y: 2, Size: headerSize, Value: uptimeHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, uptimeElements...)
			}
			frameMap["uptime"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: uptimeElements}

			// Pomodoro frame - clean, centered countdown display
			pomodoroMinutes := pomodoroSession.TimeRemaining / 60
			pomodoroSeconds := pomodoroSession.TimeRemaining % 60
			pomodoroTimeStr := fmt.Sprintf("%02d:%02d", pomodoroMinutes, pomodoroSeconds)

			// Mode display text
			var modeText string
			switch pomodoroSession.Mode {
			case "work":
				modeText = "FOCUS"
			case "break":
				modeText = "BREAK"
			case "longBreak":
				modeText = "LONG BREAK"
			default:
				modeText = "READY"
			}

			// Status indicator
			statusText := ""
			if pomodoroSession.IsPaused {
				statusText = "PAUSED"
			} else if !pomodoroSession.Active {
				statusText = "READY"
				modeText = "POMODORO"
			}

			// Cycle progress
			cycleText := fmt.Sprintf("%d/%d", pomodoroSession.CyclesCompleted, pomodoroSettings.CyclesUntilLong)

			// Build clean Pomodoro display elements
			// Large centered countdown timer for visibility
			timeX := calcCenteredX(pomodoroTimeStr, 2)
			pomodoroElements := []Element{
				// Large time display in center
				{Type: "text", X: timeX, Y: 22, Size: 2, Value: pomodoroTimeStr},
			}

			if showHeaders {
				// Header with mode
				headerText := fmt.Sprintf("= %s =", modeText)
				headerX := calcCenteredX(headerText, 1)
				pomodoroElements = append([]Element{
					{Type: "text", X: headerX, Y: 2, Size: 1, Value: headerText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, pomodoroElements...)

				// Footer with status and cycles
				pomodoroElements = append(pomodoroElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				if statusText != "" {
					pomodoroElements = append(pomodoroElements, Element{Type: "text", X: 8, Y: 55, Size: 1, Value: statusText})
				}
				pomodoroElements = append(pomodoroElements, Element{Type: "text", X: 90, Y: 55, Size: 1, Value: cycleText})
			} else {
				// Without headers, add mode text below timer (centered)
				modeX := calcCenteredX(modeText, 1)
				pomodoroElements = append(pomodoroElements, Element{Type: "text", X: modeX, Y: 48, Size: 1, Value: modeText})
			}

			frameMap["pomodoro"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: pomodoroElements}

			// Build frames array based on cycleItems
			frames = []Frame{}
			for _, item := range cycleItems {
				if !item.Enabled {
					continue
				}

				duration := item.Duration
				if duration <= 0 {
					duration = 3000 // Default duration
				}

				switch item.Type {
				case "time":
					frame := frameMap["time"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "weather":
					frame := frameMap["weather"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "uptime":
					frame := frameMap["uptime"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "text":
					// Custom text message
					var elements []Element
					textSize := item.Size
					if textSize <= 0 {
						textSize = 2
					}

					switch item.Style {
					case "centered":
						charWidth := textSize * 6
						textWidth := len(item.Text) * charWidth
						x := (128 - textWidth) / 2
						if x < 0 {
							x = 0
						}
						elements = []Element{
							{Type: "text", X: x, Y: 28, Size: textSize, Value: item.Text},
						}
					case "framed":
						elements = []Element{
							{Type: "line", X: 0, Y: 0, Width: 128, Height: 1},
							{Type: "line", X: 0, Y: 63, Width: 128, Height: 1},
							{Type: "line", X: 0, Y: 0, Width: 1, Height: 64},
							{Type: "line", X: 127, Y: 0, Width: 1, Height: 64},
							{Type: "text", X: 8, Y: 28, Size: textSize, Value: item.Text},
						}
					default: // "normal"
						elements = []Element{
							{Type: "text", X: 4, Y: 28, Size: textSize, Value: item.Text},
						}
					}

					if showHeaders && item.Label != "" {
						elements = append([]Element{
							{Type: "text", X: 32, Y: 2, Size: 1, Value: "= MESSAGE ="},
							{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
						}, elements...)
					}

					frames = append(frames, Frame{Version: 1, Duration: duration, Clear: true, Elements: elements})

				case "image":
					// Custom image
					if len(item.Bitmap) > 0 {
						elements := []Element{
							{Type: "bitmap", X: 0, Y: 0, Width: item.Width, Height: item.Height, Bitmap: item.Bitmap},
						}
						frames = append(frames, Frame{Version: 1, Duration: duration, Clear: true, Elements: elements})
					}

				case "pomodoro":
					// Pomodoro timer display
					frame := frameMap["pomodoro"]
					frame.Duration = duration
					frames = append(frames, frame)

				case "countdown":
					// Countdown timer to target date
					if item.TargetDate != "" {
						targetTime, err := time.Parse("2006-01-02", item.TargetDate)
						if err == nil {
							// Calculate time remaining
							remaining := time.Until(targetTime)

							var countdownStr string
							if remaining <= 0 {
								countdownStr = "Done!"
							} else if remaining.Hours() >= 24 {
								days := int(remaining.Hours() / 24)
								hours := int(remaining.Hours()) % 24
								countdownStr = fmt.Sprintf("%dd %dh", days, hours)
							} else if remaining.Hours() >= 1 {
								hours := int(remaining.Hours())
								mins := int(remaining.Minutes()) % 60
								countdownStr = fmt.Sprintf("%dh %dm", hours, mins)
							} else {
								mins := int(remaining.Minutes())
								secs := int(remaining.Seconds()) % 60
								countdownStr = fmt.Sprintf("%dm %ds", mins, secs)
							}

							// Build frame elements
							label := item.TargetLabel
							if label == "" {
								label = "Countdown"
							}

							countdownElements := []Element{
								{Type: "text", X: calcCenteredX(countdownStr, 2), Y: 24, Size: 2, Value: countdownStr},
							}
							if showHeaders {
								headerText := fmt.Sprintf("= %s =", label)
								countdownElements = append([]Element{
									{Type: "text", X: calcCenteredX(headerText, 1), Y: 2, Size: 1, Value: headerText},
									{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
								}, countdownElements...)
								countdownElements = append(countdownElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
								// Show target date at bottom
								dateStr := targetTime.Format("Jan 2, 2006")
								countdownElements = append(countdownElements, Element{Type: "text", X: calcCenteredX(dateStr, 1), Y: 55, Size: 1, Value: dateStr})
							}

							frames = append(frames, Frame{Version: 1, Duration: duration, Clear: true, Elements: countdownElements})
						}
					}

				case "qr":
					// QR code display
					if item.QRData != "" {
						qrFrame, err := generateQRFrame(item.QRData, duration)
						if err == nil {
							frames = append(frames, qrFrame)
						}
					}

				case "bcd":
					// BCD (Binary-Coded Decimal) clock display
					bcdFrame := generateBCDFrame(duration)
					frames = append(frames, bcdFrame)
				}
			}

			// Auto-include Pomodoro in cycle if enabled in settings and not already in cycle
			if pomodoroSettings.ShowInCycle {
				hasPomodoroInCycle := false
				for _, item := range cycleItems {
					if item.Type == "pomodoro" && item.Enabled {
						hasPomodoroInCycle = true
						break
					}
				}
				if !hasPomodoroInCycle {
					frame := frameMap["pomodoro"]
					frame.Duration = 3000
					frames = append(frames, frame)
				}
			}

			// Fallback: if no frames, show at least time
			if len(frames) == 0 {
				frames = append(frames, frameMap["time"])
			}
		}

		mutex.Unlock()
	}
}
