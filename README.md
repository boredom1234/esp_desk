# ESP Desk OS

A smart desk display system powered by **ESP32** and a **128x64 OLED display**, controlled via a modern web dashboard. Display time, weather (with AQI), custom text, images, animated GIFs, and scrolling marquees on your desk companion.

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)
![ESP32](https://img.shields.io/badge/ESP32-E7352C?style=flat&logo=espressif&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

---

## Features

- **Time Display** — Real-time clock with configurable timezone
- **Spotify Integration** — Display currently playing song and artist with album art support
- **Pomodoro Timer** — Productivity timer with work/break intervals
- **QR Codes** — Generate and display QR codes for any text/URL
- **Advanced Clocks** — Binary (BCD) and Analog clock faces
- **Moon Phase** — Real-time moon phase tracking
- **Weather Widget** — Live weather data from Open-Meteo API with Air Quality Index (AQI), PM2.5, and PM10 readings
- **Uptime Tracker** — Server uptime monitoring
- **Custom Text** — Display custom messages (normal, centered, or framed styles)
- **Marquee/Scrolling Text** — Animated scrolling text with configurable speed, direction, and local ESP32 playback
- **Image Upload** — Upload PNG, JPG, or GIF files (auto-converted to 1-bit for OLED)
- **GIF Animations** — Full animated GIF support with local ESP32 playback (no network lag)
- **Display Cycle** — Customizable rotation of widgets with drag-and-drop ordering
- **RGB LED Beacon** — Satellite-style status indicator with configurable brightness
- **Password Protection** — Optional dashboard authentication with rate limiting
- **Responsive UI** — Modern tabbed interface optimized for desktop and mobile

---

## Architecture

The system follows a **"Backend as Brain, ESP32 as GPU"** philosophy — the ESP32 never decides _what_ to show, only _how_ to render the instructions it receives.

```
┌─────────────────────┐      HTTP/JSON      ┌─────────────────────┐
│   Web Dashboard     │ ◄─────────────────► │    Go Backend       │
│   (Browser)         │                     │    (main.go)        │
└─────────────────────┘                     └──────────┬──────────┘
                                                       │
                                                       │ HTTP API
                                                       │ • Polling mode (frame/next)
                                                       │ • Local playback mode (gif/full)
                                                       ▼
                                            ┌─────────────────────┐
                                            │   ESP32 + OLED      │
                                            │   (main.ino)        │
                                            │   + RGB LED Beacon  │
                                            └─────────────────────┘
```

### Playback Modes

| Mode                    | Description                                                      | Use Case                                |
| ----------------------- | ---------------------------------------------------------------- | --------------------------------------- |
| **Polling Mode**        | ESP32 fetches `/frame/next` repeatedly                           | Time, weather, uptime, static content   |
| **Local Playback Mode** | ESP32 downloads all frames via `/api/gif/full` and plays locally | GIFs, marquees — eliminates network lag |

The ESP32 automatically switches between modes based on server hints (`isGifMode` field).

---

## Project Structure

```
esp_desk/
├── main.go                  # Go backend (API, image processing, frame generation)
├── main.ino                 # ESP32 Arduino firmware (display, WiFi, local playback)
├── spotify.go               # Spotify integration
├── pomodoro.go              # Pomodoro timer logic
├── qrcode.go                # QR code generation
├── bcd.go                   # Binary Clock Display logic
├── analog.go                # Analog clock logic
├── moonphase.go             # Moon phase calculation
├── weather.go               # Weather API handling
├── background.go            # Background tasks and polling
├── config.json              # Persisted settings (auto-generated)
├── static/
│   ├── index.html           # Web dashboard UI (tabbed layout)
│   ├── css/
│   │   └── style.css        # Dashboard styling (modern minimal theme)
│   └── js/
│       ├── app.js           # Main app initialization
│       ├── api.js           # Backend API communication
│       ├── auth.js          # Authentication handling
│       ├── autoplay.js      # Auto-play cycle control
│       ├── controls.js      # UI control handlers
│       ├── cycle.js         # Display cycle management
│       ├── render.js        # OLED preview rendering
│       ├── texthelper.js    # Text input utilities
│       ├── upload.js        # Image/GIF upload handling
│       ├── utils.js         # Shared utilities
│       └── weather.js       # Weather display logic
├── .env.example             # Environment configuration template
├── render.yaml              # Render.com deployment configuration
├── rules.md                 # Development guidelines
└── go.mod                   # Go module definition
```

---

## Quick Start

### Prerequisites

- **Go 1.21+** — Backend server
- **ESP32** with SSD1306 OLED (128x64) and optional RGB LED
- **Arduino IDE** with ESP32, ArduinoJson, and Adafruit SSD1306 libraries

### 1. Backend Setup

```bash
# Clone the repository
git clone https://github.com/boredom1234/esp_desk.git
cd esp_desk

# Configure environment
cp .env.example .env
# Edit .env and set DASHBOARD_PASSWORD

# Run the server
go run main.go
```

The dashboard will be available at `http://localhost:3000`

### 2. ESP32 Setup

1. Open `main.ino` in Arduino IDE
2. Update the WiFi credentials and backend URL:
   ```cpp
   const char* ssid = "Your-WiFi-SSID";
   const char* password = "Your-WiFi-Password";
   const char* FRAME_CURRENT_URL = "https://your-server.com/frame/current";
   const char* FRAME_NEXT_URL    = "https://your-server.com/frame/next";
   const char* GIF_FULL_URL      = "https://your-server.com/api/gif/full";
   ```
3. Upload to your ESP32

---

## Hardware Wiring

### ESP32 + SSD1306 OLED (I2C) + RGB LED Beacon

```
    ESP32                     SSD1306 OLED (128x64)
  ┌─────────┐                 ┌─────────────────┐
  │     3V3 ├─────────────────┤ VCC             │
  │     GND ├─────────────────┤ GND             │
  │ GPIO 21 ├─────────────────┤ SDA (Data)      │
  │ GPIO 22 ├─────────────────┤ SCL (Clock)     │
  └─────────┘                 └─────────────────┘

    ESP32                     RGB LED (Common Cathode)
  ┌─────────┐                 ┌─────────────────┐
  │ GPIO 25 ├──[220Ω]─────────┤ Red             │
  │ GPIO 26 ├──[220Ω]─────────┤ Green           │
  │ GPIO 27 ├──[220Ω]─────────┤ Blue            │
  │     GND ├─────────────────┤ Common GND      │
  └─────────┘                 └─────────────────┘
```

| ESP32 Pin | Component         | Description                            |
| --------- | ----------------- | -------------------------------------- |
| 3V3       | OLED VCC          | Power (3.3V)                           |
| GND       | OLED GND, LED GND | Ground                                 |
| GPIO 21   | OLED SDA          | I2C Data                               |
| GPIO 22   | OLED SCL          | I2C Clock                              |
| GPIO 2    | Built-in LED      | Status indicator (blinks during fetch) |
| GPIO 25   | RGB Red           | RGB beacon (through 220Ω resistor)     |
| GPIO 26   | RGB Green         | RGB beacon (through 220Ω resistor)     |
| GPIO 27   | RGB Blue          | RGB beacon (through 220Ω resistor)     |

> **Note:** Default I2C address is `0x3C`. Update `OLED_ADDRESS` in `main.ino` if different.

### Wokwi Project for Reference

[LINK](https://wokwi.com/projects/451211743294143489)

### RGB LED Beacon Colors

| Color  | State                    |
| ------ | ------------------------ |
| Blue   | Idle/standby             |
| Orange | Fetching data            |
| Green  | Data loaded successfully |
| Red    | Error                    |
| Purple | Animation playing        |
| Cyan   | WiFi connecting          |

---

## API Endpoints

### ESP32 Firmware Endpoints

| Endpoint         | Method | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| `/frame/current` | GET    | Get current display frame (initial boot)              |
| `/frame/next`    | GET    | Advance to next frame in cycle (polling mode)         |
| `/api/gif/full`  | GET    | Download all GIF/marquee frames (local playback mode) |

### Dashboard Endpoints

| Endpoint        | Method   | Description                                           |
| --------------- | -------- | ----------------------------------------------------- |
| `/api/settings` | GET/POST | Read/update display settings, cycle items, LED beacon |
| `/api/text`     | POST     | Display styled text (normal/centered/framed)          |
| `/api/marquee`  | POST     | Start scrolling text animation (local playback)       |
| `/api/custom`   | POST     | Display custom bitmap or text                         |
| `/api/upload`   | POST     | Upload image/GIF (auto-converts to 1-bit)             |
| `/api/weather`  | GET/POST | Get weather data / change city                        |
| `/api/timezone` | POST     | Set display timezone                                  |
| `/api/reset`    | POST     | Reset all settings to defaults                        |

### Authentication Endpoints

| Endpoint           | Method | Description                          |
| ------------------ | ------ | ------------------------------------ |
| `/api/auth/status` | GET    | Check if authentication is required  |
| `/api/auth/login`  | POST   | Authenticate with dashboard password |
| `/api/auth/logout` | POST   | Invalidate session token             |

---

## Configuration

### Environment Variables

| Variable                | Default | Description                                      |
| ----------------------- | ------- | ------------------------------------------------ |
| `PORT`                  | `3000`  | Server port                                      |
| `DASHBOARD_PASSWORD`    | —       | Dashboard access password (optional)             |
| `SPOTIFY_CLIENT_ID`     | —       | Spotify Client ID (for music integration)        |
| `SPOTIFY_CLIENT_SECRET` | —       | Spotify Client Secret                            |
| `ASTRONOMY_API_KEY`     | —       | Astronomy API ID/Secret (for accurate moon data) |

### Persisted Settings (config.json)

The following settings are automatically saved and restored:

- Display cycle items and order
- Auto-play state and frame duration
- ESP32 refresh interval
- Weather city and coordinates
- Timezone
- Display rotation
- LED beacon settings (brightness, enabled)
- Header visibility

---

## Deployment

Deploy to [Render.com](https://render.com) using the included `render.yaml`:

```yaml
services:
  - type: web
    name: esp-desk
    env: go
    buildCommand: go build -tags netgo -ldflags '-s -w' -o app .
    startCommand: ./app
```

---

## Dependencies

**Backend (Go):**

- Standard library only (no external dependencies)
- `github.com/skip2/go-qrcode`

**ESP32 (Arduino):**

- `WiFi.h` / `WiFiClientSecure.h`
- `HTTPClient.h`
- `ArduinoJson.h`
- `Adafruit_GFX.h`
- `Adafruit_SSD1306.h`

---

## Development Guidelines

See [rules.md](rules.md) for the complete development philosophy. Key principles:

1. **Backend = Brain, ESP32 = GPU** — ESP32 never decides _what_ to show
2. **Stateless ESP32** — Can reboot anytime, backend stays in control
3. **Stable JSON Contract** — Add fields, never rename/remove
4. **Safe Defaults** — ESP32 provides fallbacks for all values
5. **One Feature at a Time** — Backend → cURL test → ESP32 → UI

---

## License

MIT License — Feel free to use, modify, and distribute.

---
