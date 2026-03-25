package main

import (
	"fmt"
	"log"
	"time"
)

func updateLoop() {
	go func() {
		fetchWeather()
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			fetchWeather()
		}
	}()

	
	ticker := time.NewTicker(100 * time.Millisecond)
	var lastPomodoroTick time.Time
	var pomodoroAccumulator time.Duration
	for range ticker.C {
		mutex.Lock()

		nowTick := time.Now()
		if lastPomodoroTick.IsZero() {
			lastPomodoroTick = nowTick
		}
		delta := nowTick.Sub(lastPomodoroTick)
		lastPomodoroTick = nowTick

		if pomodoroSession.Active && !pomodoroSession.IsPaused {
			pomodoroAccumulator += delta
			for pomodoroAccumulator >= time.Second {
				pomodoroAccumulator -= time.Second
				if pomodoroSession.TimeRemaining > 0 {
					pomodoroSession.TimeRemaining--
					continue
				}
				if pomodoroSession.Mode == "work" {
					pomodoroSession.CyclesCompleted++
					if pomodoroSession.CyclesCompleted >= pomodoroSettings.CyclesUntilLong {
						pomodoroSession.Mode = "longBreak"
						pomodoroSession.TimeRemaining = pomodoroSettings.LongBreak
						pomodoroSession.CyclesCompleted = 0
						log.Printf("Pomodoro: Auto-started long break (%d min)", pomodoroSettings.LongBreak/60)
					} else {
						pomodoroSession.Mode = "break"
						pomodoroSession.TimeRemaining = pomodoroSettings.BreakDuration
						log.Printf("Pomodoro: Auto-started break (%d min)", pomodoroSettings.BreakDuration/60)
					}
				} else {
					pomodoroSession.Mode = "work"
					pomodoroSession.TimeRemaining = pomodoroSettings.WorkDuration
					log.Printf("Pomodoro: Auto-started work session (%d min)", pomodoroSettings.WorkDuration/60)
				}
				pomodoroSession.StartedAt = nowTick
			}
		} else {
			pomodoroAccumulator = 0
		}

		localIsCustomMode := isCustomMode
		localStartTime := startTime
		localDisplayLocation := displayLocation
		localTimeShowSeconds := timeShowSeconds
		localShowHeaders := showHeaders
		localBCD24HourMode := bcd24HourMode
		localBCDShowSeconds := bcdShowSeconds
		localAnalogShowSeconds := analogShowSeconds
		localAnalogShowRoman := analogShowRoman
		localWeatherData := weatherData
		localPomodoroSession := pomodoroSession
		localPomodoroSettings := pomodoroSettings
		localCycleItems := make([]CycleItem, len(cycleItems))
		copy(localCycleItems, cycleItems)

		localSpotifyTrack := spotifyLastTrack
		localSpotifyEnabled := spotifyEnabled
		localMoonPhaseData := moonPhaseData

		mutex.Unlock()

		var newFrames []Frame

		if !localIsCustomMode {
			now := time.Now()
			if localDisplayLocation != nil {
				now = now.In(localDisplayLocation)
			}
			timeFormat := "15:04"
			if localTimeShowSeconds {
				timeFormat = "15:04:05"
			}
			currentTime := now.Format(timeFormat)
			uptime := time.Since(localStartTime).Round(time.Second).String()

			frameMap := make(map[string]Frame)

			tzAbbrev, _ := now.Zone()
			timeMainSize := getScaledTextSize(2)
			headerSize := getScaledTextSize(1)

			timeElements := []Element{
				{Type: "text", X: calcCenteredX(currentTime, timeMainSize), Y: 22, Size: timeMainSize, Value: currentTime},
			}
			if localShowHeaders {
				timeHeaderText := "= TIME ="
				timeElements = append([]Element{
					{Type: "text", X: calcCenteredX(timeHeaderText, headerSize), Y: 2, Size: headerSize, Value: timeHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, timeElements...)
				timeElements = append(timeElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				timeElements = append(timeElements, Element{Type: "text", X: calcCenteredX(tzAbbrev, headerSize), Y: 55, Size: headerSize, Value: tzAbbrev})
			}
			frameMap["time"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: timeElements}

			aqiDisplay := ""
			if localWeatherData.AQI > 0 {
				aqiDisplay = fmt.Sprintf("AQI:%d", localWeatherData.AQI)
			}

			weatherMainSize := getScaledTextSize(2)
			weatherLabelSize := getScaledTextSize(1)
			weatherElements := []Element{
				{Type: "text", X: calcCenteredX(localWeatherData.Temperature, weatherMainSize), Y: 20, Size: weatherMainSize, Value: localWeatherData.Temperature},
			}
			if localShowHeaders {
				weatherHeaderText := "= WEATHER ="
				weatherElements = append([]Element{
					{Type: "text", X: calcCenteredX(weatherHeaderText, weatherLabelSize), Y: 2, Size: weatherLabelSize, Value: weatherHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, weatherElements...)

				if aqiDisplay != "" {
					weatherElements = append(weatherElements, Element{Type: "text", X: 5, Y: 42, Size: weatherLabelSize, Value: localWeatherData.Condition})
					weatherElements = append(weatherElements, Element{Type: "text", X: 75, Y: 42, Size: weatherLabelSize, Value: aqiDisplay})
				} else {
					weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(localWeatherData.Condition, weatherLabelSize), Y: 42, Size: weatherLabelSize, Value: localWeatherData.Condition})
				}
				weatherElements = append(weatherElements, Element{Type: "line", X: 0, Y: 53, Width: 128, Height: 1})
				weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(localWeatherData.City, weatherLabelSize), Y: 56, Size: weatherLabelSize, Value: localWeatherData.City})
			} else {
				weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(localWeatherData.Condition, weatherLabelSize), Y: 42, Size: weatherLabelSize, Value: localWeatherData.Condition})
				if aqiDisplay != "" {
					weatherElements = append(weatherElements, Element{Type: "text", X: calcCenteredX(aqiDisplay, weatherLabelSize), Y: 52, Size: weatherLabelSize, Value: aqiDisplay})
				}
			}
			frameMap["weather"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: weatherElements}

			uptimeSize := getScaledTextSize(1)
			uptimeElements := []Element{
				{Type: "text", X: calcCenteredX(uptime, uptimeSize), Y: 28, Size: uptimeSize, Value: uptime},
			}
			if localShowHeaders {
				uptimeHeaderText := "= UPTIME ="
				uptimeElements = append([]Element{
					{Type: "text", X: calcCenteredX(uptimeHeaderText, headerSize), Y: 2, Size: headerSize, Value: uptimeHeaderText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, uptimeElements...)
			}
			frameMap["uptime"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: uptimeElements}

			pomodoroMinutes := localPomodoroSession.TimeRemaining / 60
			pomodoroSeconds := localPomodoroSession.TimeRemaining % 60
			pomodoroTimeStr := fmt.Sprintf("%02d:%02d", pomodoroMinutes, pomodoroSeconds)

			var modeText string
			switch localPomodoroSession.Mode {
			case "work":
				modeText = "FOCUS"
			case "break":
				modeText = "BREAK"
			case "longBreak":
				modeText = "LONG BREAK"
			default:
				modeText = "READY"
			}

			statusText := ""
			if localPomodoroSession.IsPaused {
				statusText = "PAUSED"
			} else if !localPomodoroSession.Active {
				statusText = "READY"
				modeText = "POMODORO"
			}

			cycleText := fmt.Sprintf("%d/%d", localPomodoroSession.CyclesCompleted, localPomodoroSettings.CyclesUntilLong)

			timeX := calcCenteredX(pomodoroTimeStr, 2)
			pomodoroElements := []Element{
				{Type: "text", X: timeX, Y: 22, Size: 2, Value: pomodoroTimeStr},
			}

			if localShowHeaders {
				headerText := fmt.Sprintf("= %s =", modeText)
				headerX := calcCenteredX(headerText, 1)
				pomodoroElements = append([]Element{
					{Type: "text", X: headerX, Y: 2, Size: 1, Value: headerText},
					{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
				}, pomodoroElements...)

				pomodoroElements = append(pomodoroElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})
				if statusText != "" {
					pomodoroElements = append(pomodoroElements, Element{Type: "text", X: 8, Y: 55, Size: 1, Value: statusText})
				}
				pomodoroElements = append(pomodoroElements, Element{Type: "text", X: 90, Y: 55, Size: 1, Value: cycleText})
			} else {
				modeX := calcCenteredX(modeText, 1)
				pomodoroElements = append(pomodoroElements, Element{Type: "text", X: modeX, Y: 48, Size: 1, Value: modeText})
			}
			frameMap["pomodoro"] = Frame{Version: 1, Duration: 3000, Clear: true, Elements: pomodoroElements}

			for _, item := range localCycleItems {
				if !item.Enabled {
					continue
				}

				duration := item.Duration
				if duration <= 0 {
					duration = 3000
				}

				switch item.Type {
				case "time":
					frame := frameMap["time"]
					frame.Duration = duration
					newFrames = append(newFrames, frame)

				case "weather":
					frame := frameMap["weather"]
					frame.Duration = duration
					newFrames = append(newFrames, frame)

				case "uptime":
					frame := frameMap["uptime"]
					frame.Duration = duration
					newFrames = append(newFrames, frame)

				case "text":
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
					default:
						elements = []Element{
							{Type: "text", X: 4, Y: 28, Size: textSize, Value: item.Text},
						}
					}

					if localShowHeaders && item.Label != "" {
						elements = append([]Element{
							{Type: "text", X: 32, Y: 2, Size: 1, Value: "= MESSAGE ="},
							{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
						}, elements...)
					}

					newFrames = append(newFrames, Frame{Version: 1, Duration: duration, Clear: true, Elements: elements})

				case "image":
					if len(item.Bitmap) > 0 {
						elements := []Element{
							{Type: "bitmap", X: 0, Y: 0, Width: item.Width, Height: item.Height, Bitmap: item.Bitmap},
						}
						newFrames = append(newFrames, Frame{Version: 1, Duration: duration, Clear: true, Elements: elements})
					}

				case "pomodoro":
					frame := frameMap["pomodoro"]
					frame.Duration = duration
					newFrames = append(newFrames, frame)

				case "countdown":
					if item.TargetDate != "" {
						targetTime, err := time.Parse("2006-01-02", item.TargetDate)
						if err == nil {
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

							label := item.TargetLabel
							if label == "" {
								label = "Countdown"
							}

							countdownElements := []Element{
								{Type: "text", X: calcCenteredX(countdownStr, 2), Y: 24, Size: 2, Value: countdownStr},
							}
							if localShowHeaders {
								headerText := fmt.Sprintf("= %s =", label)
								countdownElements = append([]Element{
									{Type: "text", X: calcCenteredX(headerText, 1), Y: 2, Size: 1, Value: headerText},
									{Type: "line", X: 0, Y: 12, Width: 128, Height: 1},
								}, countdownElements...)
								countdownElements = append(countdownElements, Element{Type: "line", X: 0, Y: 52, Width: 128, Height: 1})

								dateStr := targetTime.Format("Jan 2, 2006")
								countdownElements = append(countdownElements, Element{Type: "text", X: calcCenteredX(dateStr, 1), Y: 55, Size: 1, Value: dateStr})
							}

							newFrames = append(newFrames, Frame{Version: 1, Duration: duration, Clear: true, Elements: countdownElements})
						}
					}

				case "qr":
					if item.QRData != "" {
						qrFrame, err := generateQRFrame(item.QRData, duration)
						if err == nil {
							newFrames = append(newFrames, qrFrame)
						}
					}

				case "bcd":
					bcdFrame := generateBCDFrame(duration, localDisplayLocation, localShowHeaders, localBCD24HourMode, localBCDShowSeconds)
					newFrames = append(newFrames, bcdFrame)

				case "analog":
					analogFrame := generateAnalogFrame(duration, localDisplayLocation, localShowHeaders, localAnalogShowSeconds, localAnalogShowRoman)
					newFrames = append(newFrames, analogFrame)

				case "spotify":

					spotifyFrame := generateSpotifyFrame(duration, localSpotifyTrack, localSpotifyEnabled)
					newFrames = append(newFrames, spotifyFrame)

				case "moonphase":

					moonFrame := generateMoonPhaseFrame(duration, localMoonPhaseData, localShowHeaders)
					newFrames = append(newFrames, moonFrame)

				case "wordclock":

					wordClockFrame := generateWordClockFrame(duration, localDisplayLocation, localShowHeaders)
					newFrames = append(newFrames, wordClockFrame)

				case "snake":

					snakeFrame := generateSnakeFrame(duration, localShowHeaders)
					newFrames = append(newFrames, snakeFrame)
				}
			}

			if localPomodoroSettings.ShowInCycle {
				hasPomodoroInCycle := false
				for _, item := range localCycleItems {
					if item.Type == "pomodoro" && item.Enabled {
						hasPomodoroInCycle = true
						break
					}
				}
				if !hasPomodoroInCycle {
					frame := frameMap["pomodoro"]
					frame.Duration = 3000
					newFrames = append(newFrames, frame)
				}
			}

			if len(newFrames) == 0 {
				newFrames = append(newFrames, frameMap["time"])
			}
		}

		mutex.Lock()
		frames = newFrames
		if len(frames) == 0 || index < 0 || index >= len(frames) {
			index = 0
		}
		mutex.Unlock()
	}
}
