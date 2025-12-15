# 📺 ESP Desk OS

A smart desk display system powered by **ESP32** and a **128x64 OLED display**, controlled via a modern web dashboard. Display time, weather, custom text, images, and animated GIFs on your desk companion.

![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)
![ESP32](https://img.shields.io/badge/ESP32-E7352C?style=flat&logo=espressif&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

---

## ✨ Features

- **🕐 Time Display** — Real-time clock with configurable styling
- **🌤 Weather Widget** — Live weather data from Open-Meteo API with city selection
- **⏱ Uptime Tracker** — Server uptime monitoring
- **💬 Custom Text** — Display custom messages (normal, centered, or framed styles)
- **📜 Marquee/Scrolling Text** — Animated scrolling text with configurable speed and direction
- **🖼 Image Upload** — Upload PNG, JPG, or GIF files (auto-converted to 1-bit for OLED)
- **🎬 GIF Animations** — Full animated GIF support with frame-by-frame playback
- **🔄 Display Cycle** — Customizable rotation of widgets with drag-and-drop ordering
- **🔐 Password Protection** — Optional dashboard authentication

---

## 🏗 Architecture

```
┌─────────────────────┐      HTTP/JSON      ┌─────────────────────┐
│   Web Dashboard     │ ◄─────────────────► │    Go Backend       │
│   (Browser)         │                     │    (main.go)        │
└─────────────────────┘                     └──────────┬──────────┘
                                                       │
                                                       │ HTTP API
                                                       ▼
                                            ┌─────────────────────┐
                                            │   ESP32 + OLED      │
                                            │   (main.ino)        │
                                            └─────────────────────┘
```

---

## 📁 Project Structure

```
esp_desk/
├── main.go              # Go backend server (API, image processing, frame generation)
├── main.ino             # ESP32 Arduino firmware (OLED display, WiFi, frame fetching)
├── static/
│   ├── index.html       # Web dashboard UI
│   ├── css/style.css    # Dashboard styling
│   └── js/app.js        # Frontend logic
├── .env.example         # Environment configuration template
├── render.yaml          # Render.com deployment configuration
├── go.mod               # Go module definition
└── rules.md             # Project guidelines
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.21+** — Backend server
- **ESP32** with SSD1306 OLED (128x64)
- **Arduino IDE** with ESP32 and Adafruit SSD1306 libraries

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
   ```
3. Upload to your ESP32

---

## 🔌 Hardware Wiring

### ESP32 + SSD1306 OLED (I2C)

```
    ESP32                     SSD1306 OLED (128x64)
  ┌─────────┐                 ┌─────────────────┐
  │     3V3 ├─────────────────┤ VCC             │
  │     GND ├─────────────────┤ GND             │
  │ GPIO 21 ├─────────────────┤ SDA (Data)      │
  │ GPIO 22 ├─────────────────┤ SCL (Clock)     │
  │  GPIO 2 ├──── (Status LED, built-in)        │
  └─────────┘                 └─────────────────┘
```

| ESP32 Pin | OLED Pin | Description                           |
| --------- | -------- | ------------------------------------- |
| 3V3       | VCC      | Power (3.3V)                          |
| GND       | GND      | Ground                                |
| GPIO 21   | SDA      | I2C Data                              |
| GPIO 22   | SCL      | I2C Clock                             |
| GPIO 2    | —        | Status LED (blinks during data fetch) |

> **Note:** Default I2C address is `0x3C`. Update `OLED_ADDRESS` in `main.ino` if different.

---

## 🌐 API Endpoints

### ESP32 Firmware Endpoints

These endpoints are called by the ESP32 firmware to fetch display data:

| Endpoint         | Method | Description                                           |
| ---------------- | ------ | ----------------------------------------------------- |
| `/frame/current` | GET    | Get current display frame (used on initial boot)      |
| `/frame/next`    | GET    | Advance to next frame in cycle (polling mode)         |
| `/api/gif/full`  | GET    | Download all GIF frames at once (local playback mode) |

### Dashboard Endpoints

| Endpoint        | Method   | Description                                  |
| --------------- | -------- | -------------------------------------------- |
| `/api/settings` | GET/POST | Read/update display settings & cycle items   |
| `/api/text`     | POST     | Display styled text (normal/centered/framed) |
| `/api/marquee`  | POST     | Start scrolling text animation               |
| `/api/custom`   | POST     | Display custom bitmap or text                |
| `/api/upload`   | POST     | Upload image/GIF (auto-converts to 1-bit)    |
| `/api/weather`  | GET/POST | Get weather data / change city               |
| `/api/reset`    | POST     | Reset all settings to defaults               |

---

## ⚙️ Configuration

| Variable             | Default | Description                          |
| -------------------- | ------- | ------------------------------------ |
| `PORT`               | `3000`  | Server port                          |
| `DASHBOARD_PASSWORD` | —       | Dashboard access password (optional) |

---

## 🌐 Deployment

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

## 📦 Dependencies

**Backend (Go):**

- Standard library only (no external dependencies)

**ESP32 (Arduino):**

- `WiFi.h`
- `HTTPClient.h`
- `ArduinoJson.h`
- `Adafruit_GFX.h`
- `Adafruit_SSD1306.h`

---

## 📄 License

MIT License — Feel free to use, modify, and distribute.

---
