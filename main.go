package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func loadEnvFile() {
	file, err := os.Open(".env")
	if err != nil {

		return
	}
	defer file.Close()

	contentBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading .env file: %v", err)
		return
	}
	lines := strings.Split(string(contentBytes), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			value = strings.Trim(value, "\"'")
			os.Setenv(key, value)
		}
	}
	log.Println("Loaded .env file")
}

func main() {
	startTime = time.Now()

	loadEnvFile()

	loadConfig()

	dashboardPassword = os.Getenv("DASHBOARD_PASSWORD")
	if dashboardPassword != "" {
		authEnabled = true

		dashboardPasswordHash = hashPassword(dashboardPassword)
		log.Printf("Authentication ENABLED - password required to access dashboard")
	} else {
		log.Printf("Authentication DISABLED - no DASHBOARD_PASSWORD set")
	}

	initializeTimezone()

	go cleanupExpiredTokens()
	go cleanupLoginAttempts()

	startSpotifyPoller()

	initMoonPhase()
	startMoonPhaseFetcher()

	frames = []Frame{{Duration: 1000, Clear: true, Elements: []Element{{Type: "text", X: 20, Y: 25, Size: 2, Value: "BOOTING..."}}}}

	go updateLoop()

	http.HandleFunc("/frame/current", loggingMiddleware(currentFrame))
	http.HandleFunc("/frame/next", loggingMiddleware(nextFrame))
	http.HandleFunc("/api/gif/full", loggingMiddleware(handleGifFull))

	http.HandleFunc("/api/auth/login", loggingMiddleware(handleAuthLogin))
	http.HandleFunc("/api/auth/verify", loggingMiddleware(handleAuthVerify))
	http.HandleFunc("/api/auth/logout", loggingMiddleware(handleAuthLogout))

	http.Handle("/", http.FileServer(http.Dir("./static")))

	http.HandleFunc("/api/frames", loggingMiddleware(authMiddleware(handleFrames)))
	http.HandleFunc("/api/control/next", loggingMiddleware(authMiddleware(nextFrame)))
	http.HandleFunc("/api/control/prev", loggingMiddleware(authMiddleware(prevFrame)))
	http.HandleFunc("/api/settings", loggingMiddleware(authMiddleware(handleSettings)))
	http.HandleFunc("/api/custom", loggingMiddleware(authMiddleware(handleCustom)))
	http.HandleFunc("/api/custom/text", loggingMiddleware(authMiddleware(handleCustomText)))
	http.HandleFunc("/api/custom/marquee", loggingMiddleware(authMiddleware(handleMarquee)))
	http.HandleFunc("/api/upload", loggingMiddleware(authMiddleware(handleUpload)))
	http.HandleFunc("/api/reset", loggingMiddleware(authMiddleware(handleReset)))
	http.HandleFunc("/api/settings/toggle-headers", loggingMiddleware(authMiddleware(handleToggleHeaders)))
	http.HandleFunc("/api/settings/headers-state", loggingMiddleware(authMiddleware(handleGetHeadersState)))
	http.HandleFunc("/api/weather", loggingMiddleware(authMiddleware(handleWeather)))
	http.HandleFunc("/api/settings/timezone", loggingMiddleware(authMiddleware(handleTimezone)))
	http.HandleFunc("/api/pomodoro", loggingMiddleware(authMiddleware(handlePomodoro)))
	http.HandleFunc("/api/qrcode", loggingMiddleware(authMiddleware(handleQRCode)))
	http.HandleFunc("/api/settings/bcd", loggingMiddleware(authMiddleware(handleBCDSettings)))
	http.HandleFunc("/api/settings/time", loggingMiddleware(authMiddleware(handleTimeSettings)))
	http.HandleFunc("/api/settings/analog", loggingMiddleware(authMiddleware(handleAnalogSettings)))
	http.HandleFunc("/api/settings/spotify", loggingMiddleware(authMiddleware(handleSpotifySettings)))
	http.HandleFunc("/api/spotify/auth", loggingMiddleware(authMiddleware(handleSpotifyAuth)))
	http.HandleFunc("/api/spotify/callback", loggingMiddleware(handleSpotifyCallback))
	http.HandleFunc("/api/moonphase/refresh", loggingMiddleware(authMiddleware(handleMoonPhaseRefresh)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("ESP Desk Backend v4 running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
