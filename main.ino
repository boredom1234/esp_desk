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

      display.setTextSize(size);
      display.setTextColor(SSD1306_WHITE);
      display.setCursor(x, y);
      display.print(value);
    }
  }

  display.display();
}

// ===== FUNCTION: FETCH FRAME =====
int fetchFrame(const char* url) {
  HTTPClient http;
  http.begin(url);
  int code = http.GET();

  if (code != 200) {
    http.end();
    return 1000;
  }

  StaticJsonDocument<1024> doc;
  deserializeJson(doc, http.getString());
  http.end();

  drawFrame(doc);

  return doc["duration"] | 3000;
}

void setup() {
  Serial.begin(115200);
  Wire.begin();

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
  // Only fetch next frame. 
  // This reduces network lag by 50% compared to fetching current+next.
  int duration = fetchFrame(FRAME_NEXT_URL);
  delay(duration);
}
