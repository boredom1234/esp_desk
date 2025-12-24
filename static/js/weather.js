




function getAQIColorClass(aqi) {
  if (aqi <= 50) return "aqi-good";
  if (aqi <= 100) return "aqi-moderate";
  if (aqi <= 150) return "aqi-sensitive";
  if (aqi <= 200) return "aqi-unhealthy";
  if (aqi <= 300) return "aqi-very-unhealthy";
  return "aqi-hazardous";
}

function renderWeatherDisplay(data, display) {
  display.innerHTML = "";

  
  const row1 = document.createElement("div");
  row1.className = "weather-row";

  const iconSpan = safeElement("span", data.icon || "🌡", "weather-icon");
  const infoSpan = safeElement(
    "span",
    `${data.temperature} · ${data.condition}`,
    "weather-info"
  );
  row1.appendChild(iconSpan);
  row1.appendChild(infoSpan);

  
  const row2 = document.createElement("div");
  row2.className = "weather-row weather-row-secondary";

  const windSpan = safeElement("span", `💨 ${data.windspeed}`, "weather-wind");
  row2.appendChild(windSpan);

  
  if (data.aqi && data.aqi > 0) {
    const aqiContainer = document.createElement("span");
    aqiContainer.className = "weather-aqi";

    const aqiBadge = document.createElement("span");
    aqiBadge.className = `aqi-badge ${getAQIColorClass(data.aqi)}`;
    aqiBadge.textContent = `AQI ${data.aqi}`;

    const aqiLevel = safeElement("span", data.aqiLevel || "", "aqi-level");

    aqiContainer.appendChild(aqiBadge);
    aqiContainer.appendChild(aqiLevel);
    row2.appendChild(aqiContainer);
  }

  display.appendChild(row1);
  display.appendChild(row2);

  
  if (data.pm25 && data.pm25 !== "N/A") {
    const row3 = document.createElement("div");
    row3.className = "weather-row weather-row-pm";

    const pmSpan = safeElement(
      "span",
      `PM2.5: ${data.pm25} · PM10: ${data.pm10}`,
      "weather-pm"
    );
    row3.appendChild(pmSpan);
    display.appendChild(row3);
  }
}

function loadWeather() {
  
  authFetch("/api/weather")
    .then((res) => res.json())
    .then((data) => {
      const display = document.getElementById("weatherDisplay");
      if (display && data.city) {
        renderWeatherDisplay(data, display);
      }
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.warn("loadWeather error:", err.message);
      }
    });
}

function changeCity() {
  const select = document.getElementById("citySelect");
  const value = select.value;
  const [lat, lng, city] = value.split(",");

  
  authFetch("/api/weather", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      city: city,
      latitude: parseFloat(lat),
      longitude: parseFloat(lng),
    }),
  })
    .then((res) => res.json())
    .then((data) => {
      const display = document.getElementById("weatherDisplay");
      if (display) {
        renderWeatherDisplay(data, display);
      }
    })
    .catch((err) => {
      if (err.message !== "Unauthorized") {
        console.error("changeCity error:", err);
      }
    });
}
