package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// ==========================================
// WEATHER HANDLERS
// ==========================================

func getWeatherIcon(code int, isDay bool) string {
	// WMO Weather interpretation codes
	switch {
	case code == 0:
		if isDay {
			return "☀"
		}
		return "☽"
	case code == 1, code == 2, code == 3:
		return "⛅"
	case code >= 45 && code <= 48:
		return "🌫"
	case code >= 51 && code <= 57:
		return "🌧"
	case code >= 61 && code <= 67:
		return "🌧"
	case code >= 71 && code <= 77:
		return "❄"
	case code >= 80 && code <= 82:
		return "🌧"
	case code >= 95 && code <= 99:
		return "⛈"
	default:
		return "🌡"
	}
}

func getWeatherCondition(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code == 1:
		return "Mostly Clear"
	case code == 2:
		return "Partly Cloudy"
	case code == 3:
		return "Overcast"
	case code >= 45 && code <= 48:
		return "Foggy"
	case code >= 51 && code <= 55:
		return "Drizzle"
	case code >= 56 && code <= 57:
		return "Freezing Drizzle"
	case code >= 61 && code <= 65:
		return "Rain"
	case code >= 66 && code <= 67:
		return "Freezing Rain"
	case code >= 71 && code <= 75:
		return "Snowfall"
	case code == 77:
		return "Snow Grains"
	case code >= 80 && code <= 82:
		return "Rain Showers"
	case code >= 85 && code <= 86:
		return "Snow Showers"
	case code == 95:
		return "Thunderstorm"
	case code >= 96 && code <= 99:
		return "Thunderstorm + Hail"
	default:
		return "Unknown"
	}
}

// getAQILevel returns a human-readable AQI level based on US AQI scale
func getAQILevel(aqi int) string {
	switch {
	case aqi <= 50:
		return "Good"
	case aqi <= 100:
		return "Moderate"
	case aqi <= 150:
		return "Unhealthy (SG)"
	case aqi <= 200:
		return "Unhealthy"
	case aqi <= 300:
		return "Very Unhealthy"
	default:
		return "Hazardous"
	}
}

func fetchWeather() {
	// Read coordinates with mutex to avoid race conditions
	mutex.Lock()
	lat := cityLat
	lng := cityLng
	city := currentCity
	mutex.Unlock()

	// Fetch weather data from Open-Meteo
	weatherURL := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.2f&longitude=%.2f&current_weather=true", lat, lng)
	weatherResp, err := http.Get(weatherURL)
	if err != nil {
		log.Println("Error fetching weather:", err)
		return
	}
	defer weatherResp.Body.Close()

	var w WeatherResponse
	if err := json.NewDecoder(weatherResp.Body).Decode(&w); err != nil {
		log.Println("Error decoding weather:", err)
		return
	}

	isDay := w.CurrentWeather.IsDay == 1
	newData := WeatherData{
		City:        city,
		Temperature: fmt.Sprintf("%.1fC", w.CurrentWeather.Temperature),
		Condition:   getWeatherCondition(w.CurrentWeather.WeatherCode),
		Icon:        getWeatherIcon(w.CurrentWeather.WeatherCode, isDay),
		Windspeed:   fmt.Sprintf("%.0f km/h", w.CurrentWeather.Windspeed),
		IsDay:       isDay,
		AQI:         0,
		AQILevel:    "N/A",
		PM25:        "N/A",
		PM10:        "N/A",
	}

	// Fetch air quality data from Open-Meteo Air Quality API
	aqiURL := fmt.Sprintf("https://air-quality-api.open-meteo.com/v1/air-quality?latitude=%.2f&longitude=%.2f&current=pm2_5,pm10,european_aqi,us_aqi,european_aqi_pm2_5,european_aqi_pm10", lat, lng)
	aqiResp, err := http.Get(aqiURL)
	if err != nil {
		log.Println("Error fetching AQI (continuing with weather only):", err)
	} else {
		defer aqiResp.Body.Close()
		var aq AirQualityResponse
		if err := json.NewDecoder(aqiResp.Body).Decode(&aq); err != nil {
			log.Println("Error decoding AQI:", err)
		} else {
			newData.AQI = aq.Current.USAQI
			newData.AQILevel = getAQILevel(aq.Current.USAQI)
			newData.PM25 = fmt.Sprintf("%.1f", aq.Current.PM25)
			newData.PM10 = fmt.Sprintf("%.1f", aq.Current.PM10)
			log.Printf("AQI fetched: US AQI=%d, PM2.5=%.1f, PM10=%.1f", aq.Current.USAQI, aq.Current.PM25, aq.Current.PM10)
		}
	}

	// Write weather data with mutex protection
	mutex.Lock()
	weatherData = newData
	mutex.Unlock()
}

func handleWeather(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		data := weatherData
		mutex.Unlock()
		json.NewEncoder(w).Encode(data)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			City      string  `json:"city"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Validate coordinates (Issue 7)
		if req.Latitude < -90 || req.Latitude > 90 {
			jsonError(w, "Invalid latitude: must be between -90 and 90", http.StatusBadRequest)
			return
		}
		if req.Longitude < -180 || req.Longitude > 180 {
			jsonError(w, "Invalid longitude: must be between -180 and 180", http.StatusBadRequest)
			return
		}
		if req.City == "" {
			jsonError(w, "City name is required", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		currentCity = req.City
		cityLat = req.Latitude
		cityLng = req.Longitude
		mutex.Unlock()

		// Persist settings (Issue 2)
		go saveConfig()

		// Fetch weather for new location
		fetchWeather()

		log.Printf("🌤️  Weather city changed: %s (%.2f, %.2f)", req.City, req.Latitude, req.Longitude)

		mutex.Lock()
		data := weatherData
		mutex.Unlock()

		json.NewEncoder(w).Encode(data)
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}
