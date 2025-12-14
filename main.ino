#include <WiFi.h>
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

// ===== OLED =====
#define SCREEN_WIDTH 128
#define SCREEN_HEIGHT 64
#define OLED_RESET    -1
#define OLED_ADDRESS  0x3C

// ===== STATUS LED =====
#define LED_PIN 2  // Built-in LED on most ESP32 boards (GPIO 2)

Adafruit_SSD1306 display(SCREEN_WIDTH, SCREEN_HEIGHT, &Wire, OLED_RESET);

// ===== FRAME STRUCT =====
struct Frame {
  int duration;
};

// ===== FUNCTION: DRAW FRAME =====
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

// ===== FUNCTION: FETCH FRAME =====
int fetchFrame(const char* url) {
  // Turn LED ON - fetching data
  digitalWrite(LED_PIN, HIGH);
  
  HTTPClient http;
  http.begin(url);
  int code = http.GET();

  if (code != 200) {
    http.end();
    digitalWrite(LED_PIN, LOW);  // Turn LED OFF
    return 1000;
  }

  // Increased buffer size to 8192 for large GIF animations
  StaticJsonDocument<8192> doc;
  deserializeJson(doc, http.getString());
  http.end();

  drawFrame(doc);
  
  // Turn LED OFF - fetch complete
  digitalWrite(LED_PIN, LOW);

  return doc["duration"] | 3000;
}

void setup() {
  Serial.begin(115200);
  Wire.begin();
  
  // Status LED init
  pinMode(LED_PIN, OUTPUT);
  digitalWrite(LED_PIN, LOW);

  // OLED init
  if (!display.begin(SSD1306_SWITCHCAPVCC, OLED_ADDRESS)) {
    while (true);
  }

  // WiFi
  WiFi.begin(ssid, password);
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
  }

  // Show first frame immediately
  fetchFrame(FRAME_CURRENT_URL);
}

void loop() {
  // WiFi reconnection check
  if (WiFi.status() != WL_CONNECTED) {
    Serial.println("WiFi lost, reconnecting...");
    WiFi.disconnect();
    WiFi.begin(ssid, password);
    int retries = 0;
    while (WiFi.status() != WL_CONNECTED && retries < 20) {
      delay(500);
      retries++;
    }
    if (WiFi.status() != WL_CONNECTED) {
      delay(5000); // Wait before retry
      return;
    }
    Serial.println("WiFi reconnected");
  }

  // Only fetch next frame
  int duration = fetchFrame(FRAME_NEXT_URL);
  delay(duration);
}
