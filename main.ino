#include <WiFi.h>
#include <WiFiClientSecure.h>
#include <HTTPClient.h>
#include <ArduinoJson.h>
#include <Adafruit_GFX.h>
#include <Adafruit_SSD1306.h>

// ===== WIFI =====
const char* ssid = "Wokwi-GUEST";
const char* password = "";

// ===== BACKEND =====
const char* FRAME_CURRENT_URL = "https://vqxh0hd3-3000.inc1.devtunnels.ms/frame/current";
const char* FRAME_NEXT_URL    = "https://vqxh0hd3-3000.inc1.devtunnels.ms/frame/next";
const char* GIF_FULL_URL      = "https://vqxh0hd3-3000.inc1.devtunnels.ms/api/gif/full";

// ===== OLED =====
#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 64
#define OLED_RESET    -1
#define OLED_ADDRESS  0x3C

// ===== STATUS LED =====
#define LED_PIN 2  // Built-in LED on most ESP32 boards (GPIO 2)

// ===== RGB LED BEACON (Satellite Mode) =====
// Connect RGB LED: R->GPIO25, G->GPIO26, B->GPIO27, Common GND->GND
// Use 220Ω resistors on each anode for protection
#define RGB_RED_PIN   25
#define RGB_GREEN_PIN 26
#define RGB_BLUE_PIN  27

// PWM settings for smooth fading (ESP32 LEDC)
#define PWM_FREQ      5000
#define PWM_RESOLUTION 8  // 8-bit = 0-255

// RGB Beacon state
uint8_t ledBrightness = 128;        // 0-255, controlled from web UI
bool ledBeaconEnabled = true;       // Can be toggled from web UI
unsigned long lastBeaconTime = 0;
const unsigned long BEACON_INTERVAL = 2500;  // Flash every 2.5 seconds

// Color constants (R, G, B) - scaled by brightness later
struct RGBColor {
  uint8_t r, g, b;
};
const RGBColor COLOR_IDLE     = {0, 100, 255};   // Blue - standby
const RGBColor COLOR_FETCHING = {255, 165, 0};   // Orange - receiving data
const RGBColor COLOR_SUCCESS  = {0, 255, 50};    // Green - data loaded
const RGBColor COLOR_ERROR    = {255, 0, 0};     // Red - error
const RGBColor COLOR_GIF_MODE = {180, 0, 255};   // Purple - animation playing
const RGBColor COLOR_WIFI     = {0, 255, 255};   // Cyan - WiFi connecting

// Current beacon color (updated by state)
RGBColor currentBeaconColor = COLOR_IDLE;

Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, OLED_RESET);

// ===== GIF LOCAL PLAYBACK =====
// Each frame is stored as bitmap data (1024 bytes for 128x64 1-bit)
#define MAX_GIF_FRAMES 10       // Limit to ~10KB of RAM for bitmaps (reduced for memory safety)
#define BYTES_PER_FRAME 1024    // 128x64 / 8 = 1024 bytes per frame

uint8_t gifFrames[MAX_GIF_FRAMES][BYTES_PER_FRAME];
int gifDurations[MAX_GIF_FRAMES];
int gifFrameCount = 0;
bool isGifMode = false;
int displayRotation = 0;  // 0 = normal, 2 = 180 degrees (for upside-down mounting)
unsigned long lastGifCheck = 0;
const unsigned long GIF_CHECK_INTERVAL = 30000;  // Check for new GIF every 30 seconds

// ===== RGB LED FUNCTIONS =====
// Set RGB LED to a specific color (scaled by brightness)
// Note: ESP32 Arduino Core 3.x uses ledcWrite(pin, duty) instead of channel
void setRGBColor(uint8_t r, uint8_t g, uint8_t b) {
  if (!ledBeaconEnabled) {
    ledcWrite(RGB_RED_PIN, 0);
    ledcWrite(RGB_GREEN_PIN, 0);
    ledcWrite(RGB_BLUE_PIN, 0);
    return;
  }
  // Scale color by brightness (0-255)
  uint8_t scaledR = (r * ledBrightness) / 255;
  uint8_t scaledG = (g * ledBrightness) / 255;
  uint8_t scaledB = (b * ledBrightness) / 255;
  
  ledcWrite(RGB_RED_PIN, scaledR);
  ledcWrite(RGB_GREEN_PIN, scaledG);
  ledcWrite(RGB_BLUE_PIN, scaledB);
}

// Quick beacon flash (satellite pulse effect)
void beaconFlash(RGBColor color, int durationMs = 80) {
  setRGBColor(color.r, color.g, color.b);
  delay(durationMs);
  setRGBColor(0, 0, 0);  // Turn off
}

// Non-blocking beacon update (call from loop)
void updateBeacon() {
  if (!ledBeaconEnabled) return;
  
  unsigned long now = millis();
  if (now - lastBeaconTime >= BEACON_INTERVAL) {
    lastBeaconTime = now;
    
    // Quick flash with current state color
    setRGBColor(currentBeaconColor.r, currentBeaconColor.g, currentBeaconColor.b);
    delay(80);  // Brief flash
    setRGBColor(0, 0, 0);  // Back to off
  }
}

// ===== FUNCTION: DRAW BITMAP FROM BUFFER =====
void drawBitmapFromBuffer(const uint8_t* bitmap) {
  display.clearDisplay();
  display.drawBitmap(0, 0, bitmap, 128, 64, SSD1306_WHITE);
  display.display();
}

// ===== FUNCTION: DRAW FRAME FROM JSON =====
void drawFrame(JsonDocument& doc) {
  if (doc["clear"] == true) {
    display.clearDisplay();
  }

  JsonArray elements = doc["elements"].as<JsonArray>();

  for (JsonObject el : elements) {
    const char* type = el["type"];

    if (strcmp(type, "text") == 0) {
      int x = el["x"] | 0;
      int y = el["y"] | 0;
      int size = el["size"] | 1;
      const char* value = el["value"];

      // Screen boundary check (Rule #8)
      if (x < 0) x = 0;
      if (x > 127) x = 127;
      if (y < 0) y = 0;
      if (y > 63) y = 63;

      display.setTextSize(size);
      display.setTextColor(SSD1306_WHITE);
      display.setCursor(x, y);
      display.print(value);
    }
    else if (strcmp(type, "bitmap") == 0) {
      int x = el["x"] | 0;
      int y = el["y"] | 0;
      int w = el["width"] | 0;
      int h = el["height"] | 0;
      
      // Screen boundary check (Rule #8)
      if (x < 0) x = 0;
      if (y < 0) y = 0;
      // Clamp dimensions to screen bounds
      if (x + w > 128) w = 128 - x;
      if (y + h > 64) h = 64 - y;
      if (w <= 0 || h <= 0) continue;
      
      // Copy data from JSON array to byte buffer
      JsonArray data = el["bitmap"];
      int len = data.size();
      
      // Safety check: max 1KB buffer for bitmaps
      if (len > 0 && len <= 1024) {
        uint8_t bmp[1024]; 
        for(int i=0; i<len; i++) {
          bmp[i] = (uint8_t)data[i].as<int>();
        }
        display.drawBitmap(x, y, bmp, w, h, SSD1306_WHITE);
      }
    }
    else if (strcmp(type, "line") == 0) {
      // Decorative lines for frames/borders
      int x = el["x"] | 0;
      int y = el["y"] | 0;
      int w = el["width"] | 1;
      int h = el["height"] | 1;
      
      // Screen boundary check
      if (x < 0) x = 0;
      if (y < 0) y = 0;
      if (x + w > 128) w = 128 - x;
      if (y + h > 64) h = 64 - y;
      
      // Draw filled rectangle for line
      display.fillRect(x, y, w, h, SSD1306_WHITE);
    }
  }

  display.display();
}

// ===== FUNCTION: FETCH FULL GIF =====
// Downloads all GIF frames at once and stores them in RAM for local playback
// Returns: 1 = GIF loaded successfully, 0 = no GIF available (server says isGifMode=false), -1 = network/parse error
int fetchFullGifWithStatus() {
  digitalWrite(LED_PIN, HIGH);
  
  // Log available heap before allocation
  Serial.printf("Free heap before GIF fetch: %d bytes\n", ESP.getFreeHeap());
  
  // Use WiFiClientSecure for HTTPS connections
  WiFiClientSecure client;
  client.setInsecure();  // Skip certificate verification (required for dev tunnels)
  client.setTimeout(30);  // 30 second timeout
  
  HTTPClient http;
  http.setFollowRedirects(HTTPC_STRICT_FOLLOW_REDIRECTS);  // Follow redirects
  http.setReuse(false);  // Don't reuse connection
  
  Serial.printf("Connecting to: %s\n", GIF_FULL_URL);
  
  if (!http.begin(client, GIF_FULL_URL)) {
    Serial.println("ERROR: http.begin() failed");
    digitalWrite(LED_PIN, LOW);
    return -1;
  }
  
  http.setTimeout(30000);  // 30 second timeout for large GIFs
  
  // Add header to request limited frames for ESP32 memory constraints
  http.addHeader("X-ESP32-Max-Frames", String(MAX_GIF_FRAMES));
  
  Serial.println("Sending HTTP GET request...");
  int code = http.GET();
  Serial.printf("HTTP response code: %d\n", code);

  if (code != 200) {
    String errorMsg = http.errorToString(code);
    Serial.printf("GIF fetch failed with code: %d (%s)\n", code, errorMsg.c_str());
    http.end();
    digitalWrite(LED_PIN, LOW);
    return -1;  // Network error - don't change GIF mode state
  }

  // Get response size
  int contentLength = http.getSize();
  Serial.printf("GIF response size: %d bytes\n", contentLength);
  Serial.printf("Free heap after connect: %d bytes\n", ESP.getFreeHeap());
  
  // Read entire response as String (more reliable than streaming for HTTPS)
  String payload = http.getString();
  http.end();
  
  int payloadLen = payload.length();
  Serial.printf("Received payload length: %d chars\n", payloadLen);
  
  if (payloadLen == 0) {
    Serial.println("ERROR: Empty response received");
    digitalWrite(LED_PIN, LOW);
    return -1;
  }
  
  // Debug: Print first 200 chars of response
  Serial.println("Response preview:");
  Serial.println(payload.substring(0, min(200, payloadLen)));
  
  // Calculate JSON buffer size - need extra space for JSON overhead
  size_t jsonBufferSize = min((size_t)payloadLen * 2, (size_t)122880);
  Serial.printf("Allocating JSON buffer: %d bytes\n", jsonBufferSize);
  Serial.printf("Free heap before JSON alloc: %d bytes\n", ESP.getFreeHeap());
  
  // Try to allocate dynamic JSON document
  DynamicJsonDocument* doc = new (std::nothrow) DynamicJsonDocument(jsonBufferSize);
  if (doc == nullptr) {
    Serial.println("ERROR: Failed to allocate JSON buffer - out of memory");
    digitalWrite(LED_PIN, LOW);
    return -1;
  }
  
  // Parse JSON from String
  DeserializationError error = deserializeJson(*doc, payload);
  
  // Free the payload string memory immediately
  payload = "";
  
  if (error) {
    Serial.printf("JSON parse error: %s\n", error.c_str());
    Serial.printf("JSON buffer size was: %d bytes\n", jsonBufferSize);
    delete doc;
    digitalWrite(LED_PIN, LOW);
    return -1;  // Parse error - don't change GIF mode state
  }

  bool gifMode = (*doc)["isGifMode"] | false;
  int frameCount = (*doc)["frameCount"] | 0;

  if (!gifMode || frameCount == 0) {
    Serial.println("Server says: Not in GIF mode or no frames");
    delete doc;
    digitalWrite(LED_PIN, LOW);
    isGifMode = false;
    gifFrameCount = 0;
    return 0;  // Server explicitly says no GIF mode
  }

  Serial.printf("Server sent %d frames, processing...\n", frameCount);

  // Limit frames to our buffer size
  if (frameCount > MAX_GIF_FRAMES) {
    frameCount = MAX_GIF_FRAMES;
    Serial.printf("Limiting to %d frames\n", MAX_GIF_FRAMES);
  }

  // Extract frames
  JsonArray framesArray = (*doc)["frames"].as<JsonArray>();
  gifFrameCount = 0;

  for (JsonObject frame : framesArray) {
    if (gifFrameCount >= MAX_GIF_FRAMES) break;

    // Get duration
    gifDurations[gifFrameCount] = frame["duration"] | 100;

    // Get bitmap from first element
    JsonArray elements = frame["elements"].as<JsonArray>();
    if (elements.size() > 0) {
      JsonObject el = elements[0];
      if (strcmp(el["type"], "bitmap") == 0) {
        JsonArray bitmap = el["bitmap"].as<JsonArray>();
        int len = bitmap.size();
        if (len > 0 && len <= BYTES_PER_FRAME) {
          for (int i = 0; i < len; i++) {
            gifFrames[gifFrameCount][i] = (uint8_t)bitmap[i].as<int>();
          }
          // Zero out remaining bytes if bitmap is smaller
          for (int i = len; i < BYTES_PER_FRAME; i++) {
            gifFrames[gifFrameCount][i] = 0;
          }
          gifFrameCount++;
        }
      }
    }
  }

  // Clean up JSON document
  delete doc;

  Serial.printf("Loaded %d GIF frames for local playback\n", gifFrameCount);
  Serial.printf("Free heap after GIF load: %d bytes\n", ESP.getFreeHeap());
  digitalWrite(LED_PIN, LOW);
  
  isGifMode = (gifFrameCount > 0);
  return isGifMode ? 1 : 0;
}

// Wrapper for backward compatibility
bool fetchFullGif() {
  return fetchFullGifWithStatus() == 1;
}

// ===== FUNCTION: PLAY GIF LOCALLY =====
// Plays all stored GIF frames without any network calls
void playGifLocally() {
  for (int i = 0; i < gifFrameCount; i++) {
    // Check WiFi status - if lost, exit GIF playback to attempt reconnection
    if (WiFi.status() != WL_CONNECTED) {
      Serial.println("WiFi lost during GIF playback, exiting for reconnection");
      isGifMode = false; // Force exit GIF mode so main loop handles reconnection
      return;
    }
    
    // Draw frame from buffer
    drawBitmapFromBuffer(gifFrames[i]);
    
    // Wait for frame duration
    delay(gifDurations[i]);
  }
}

// ===== FUNCTION: FETCH SINGLE FRAME (LEGACY/FALLBACK) =====
int fetchFrame(const char* url) {
  digitalWrite(LED_PIN, HIGH);
  beaconFlash(COLOR_FETCHING, 50);  // Orange flash on data fetch
  
  // Use WiFiClientSecure for HTTPS connections
  WiFiClientSecure client;
  client.setInsecure();  // Skip certificate verification (required for dev tunnels)
  client.setTimeout(10000);
  
  HTTPClient http;
  http.setFollowRedirects(HTTPC_STRICT_FOLLOW_REDIRECTS);  // Follow redirects
  http.setReuse(false);
  
  if (!http.begin(client, url)) {
    Serial.println("ERROR: fetchFrame http.begin() failed");
    digitalWrite(LED_PIN, LOW);
    return 1000;
  }
  
  http.setTimeout(10000);  // 10 second timeout for single frames
  int code = http.GET();

  if (code != 200) {
    String errorMsg = http.errorToString(code);
    Serial.printf("fetchFrame failed with HTTP code: %d (%s)\n", code, errorMsg.c_str());
    http.end();
    digitalWrite(LED_PIN, LOW);
    return 1000;  // Retry after 1 second
  }

  // Increased buffer size to 8192 for large animations
  StaticJsonDocument<8192> doc;
  String payload = http.getString();
  http.end();
  
  DeserializationError error = deserializeJson(doc, payload);
  
  if (error) {
    Serial.printf("JSON parse error in fetchFrame: %s\n", error.c_str());
    digitalWrite(LED_PIN, LOW);
    return 1000;  // Retry after 1 second
  }

  // ===== CHECK FOR DISPLAY ROTATION =====
  // Server can send displayRotation (0 = normal, 2 = 180 degrees)
  int serverRotation = doc["displayRotation"] | 0;
  if (serverRotation != displayRotation) {
    displayRotation = serverRotation;
    display.setRotation(displayRotation);
    Serial.printf("Display rotation changed to: %d\n", displayRotation);
  }

  // ===== CHECK FOR LED BEACON SETTINGS =====
  // Server sends LED brightness (0-100%) and enabled state
  int serverBrightness = doc["ledBrightness"] | -1;
  if (serverBrightness >= 0 && serverBrightness <= 100) {
    // Convert 0-100% to 0-255 for PWM
    uint8_t newBrightness = (serverBrightness * 255) / 100;
    if (newBrightness != ledBrightness) {
      ledBrightness = newBrightness;
      Serial.printf("LED brightness changed to: %d%%\n", serverBrightness);
    }
  }
  bool serverBeaconEnabled = doc["ledBeaconEnabled"] | true;
  if (serverBeaconEnabled != ledBeaconEnabled) {
    ledBeaconEnabled = serverBeaconEnabled;
    Serial.printf("LED beacon %s\n", ledBeaconEnabled ? "ENABLED" : "DISABLED");
    if (!ledBeaconEnabled) {
      setRGBColor(0, 0, 0);  // Turn off LED when disabled
    }
  }

  // ===== CHECK FOR GIF MODE HINT =====
  // Server indicates if GIF/Marquee mode is active via isGifMode field
  // This allows immediate detection without waiting for 30s poll interval
  bool serverGifMode = doc["isGifMode"] | false;
  
  if (serverGifMode) {
    // Server says GIF mode is active
    if (!isGifMode) {
      // We're not in GIF mode yet - need to fetch full GIF
      Serial.println(">>> Server signals GIF mode - fetching full GIF NOW");
      digitalWrite(LED_PIN, LOW);
      
      int result = fetchFullGifWithStatus();
      if (result == 1) {
        // Successfully loaded GIF
        Serial.println(">>> Switched to local GIF playback mode");
        return 50;  // Minimal delay before GIF playback starts
      } else if (result == 0) {
        // Server changed its mind - no GIF after all
        Serial.println(">>> Server returned no GIF data despite hint");
        // Fall through to display the current frame
      } else {
        // Network error - try again next frame
        Serial.println(">>> GIF fetch failed, will retry next frame");
        return 500;  // Retry soon
      }
    }
    // Already in GIF mode - shouldn't reach here normally
    // (loop() should be using playGifLocally)
  }

  // Draw the frame normally (polling mode)
  drawFrame(doc);
  
  digitalWrite(LED_PIN, LOW);

  return doc["duration"] | 3000;
}

// ===== FUNCTION: CHECK GIF MODE =====
// Periodically check if server has a new GIF to download
void checkForGifUpdate() {
  if (millis() - lastGifCheck < GIF_CHECK_INTERVAL) {
    return;
  }
  lastGifCheck = millis();
  
  bool wasGifMode = isGifMode;
  int previousFrameCount = gifFrameCount;
  
  // Try to fetch full GIF - returns: 1=success, 0=no GIF on server, -1=error
  int result = fetchFullGifWithStatus();
  
  if (result == 1) {
    // New GIF loaded successfully
    if (!wasGifMode || gifFrameCount != previousFrameCount) {
      Serial.println("New GIF/Marquee loaded, switching to local playback");
    }
  } else if (result == 0) {
    // Server explicitly says no GIF mode
    if (wasGifMode) {
      Serial.println("Exited GIF/Marquee mode, switching to polling");
      
      // ===== BUFFER CLEANUP =====
      // Zero out frame buffers - note: memory is statically allocated,
      // so this just clears the data but doesn't reduce RAM usage
      for (int i = 0; i < previousFrameCount; i++) {
        memset(gifFrames[i], 0, BYTES_PER_FRAME);
      }
      
      // isGifMode and gifFrameCount already set to false/0 by fetchFullGifWithStatus
      Serial.println("Buffer cleanup complete");
    }
  } else {
    // result == -1: Network or parsing error
    // IMPORTANT: Keep current GIF mode state - don't interrupt playback due to transient errors
    if (wasGifMode) {
      Serial.println("GIF update check failed (network/parse error), continuing local playback");
    }
  }
}

void setup() {
  Serial.begin(115200);
  Wire.begin();
  
  // Status LED init
  pinMode(LED_PIN, OUTPUT);
  digitalWrite(LED_PIN, LOW);

  // ===== RGB LED PWM Init (ESP32 Arduino Core 3.x API) =====
  ledcAttach(RGB_RED_PIN, PWM_FREQ, PWM_RESOLUTION);
  ledcAttach(RGB_GREEN_PIN, PWM_FREQ, PWM_RESOLUTION);
  ledcAttach(RGB_BLUE_PIN, PWM_FREQ, PWM_RESOLUTION);
  
  // Initial state: off
  setRGBColor(0, 0, 0);
  currentBeaconColor = COLOR_WIFI;  // Cyan during WiFi connect

  // OLED init
  if (!display.begin(SSD1306_SWITCHCAPVCC, OLED_ADDRESS)) {
    Serial.println("SSD1306 allocation failed - restarting in 5 seconds");
    delay(5000);
    ESP.restart();
  }

  display.clearDisplay();
  display.setTextSize(1);
  display.setTextColor(SSD1306_WHITE);
  display.setCursor(20, 28);
  display.print("Connecting WiFi...");
  display.display();

  // WiFi
  WiFi.begin(ssid, password);
  int retries = 0;
  while (WiFi.status() != WL_CONNECTED && retries < 40) {
    delay(500);
    retries++;
  }
  
  if (WiFi.status() != WL_CONNECTED) {
    display.clearDisplay();
    display.setCursor(10, 24);
    display.print("WiFi Failed!");
    display.setCursor(10, 36);
    display.print("Restarting...");
    display.display();
    delay(3000);
    ESP.restart();
  }

  Serial.println("WiFi connected");
  Serial.print("IP: ");
  Serial.println(WiFi.localIP());

  // Try to fetch full GIF/Marquee first
  if (fetchFullGif()) {
    Serial.println("Animation mode active - local playback enabled");
  } else {
    // Fallback to polling mode
    Serial.println("No animation active - using polling mode");
    fetchFrame(FRAME_CURRENT_URL);
  }
  
  lastGifCheck = millis();
}

void loop() {
  // ===== RGB BEACON UPDATE (satellite pulse) =====
  updateBeacon();
  
  // WiFi reconnection check
  if (WiFi.status() != WL_CONNECTED) {
    currentBeaconColor = COLOR_WIFI;  // Cyan during reconnect
    Serial.println("WiFi lost, reconnecting...");
    WiFi.disconnect();
    WiFi.begin(ssid, password);
    int retries = 0;
    while (WiFi.status() != WL_CONNECTED && retries < 20) {
      delay(500);
      retries++;
    }
    if (WiFi.status() != WL_CONNECTED) {
      currentBeaconColor = COLOR_ERROR;  // Red on failure
      delay(5000);
      return;
    }
    Serial.println("WiFi reconnected");
    beaconFlash(COLOR_SUCCESS, 150);  // Green flash on reconnect
  }

  if (isGifMode && gifFrameCount > 0) {
    // ===== LOCAL ANIMATION PLAYBACK (GIF/MARQUEE) =====
    currentBeaconColor = COLOR_GIF_MODE;  // Purple during animation
    
    // Play all frames from RAM without API calls
    playGifLocally();
    
    // Only check for updates AFTER a complete playback cycle
    // This prevents blocking HTTP calls from interrupting smooth animation
    checkForGifUpdate();
  } else {
    // ===== LEGACY POLLING MODE =====
    currentBeaconColor = COLOR_IDLE;  // Blue in normal mode
    
    // Check for GIF updates periodically when not in GIF mode
    checkForGifUpdate();
    
    // Fetch next frame from server
    int duration = fetchFrame(FRAME_NEXT_URL);
    delay(duration);
  }
}
