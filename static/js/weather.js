// ==========================================
// ESP DESK_OS - Weather
// ==========================================

function loadWeather() {
  fetch("/api/weather")
    .then((res) => res.json())
    .then((data) => {
      const display = document.getElementById("weatherDisplay");
      if (display && data.city) {
        // XSS-safe rendering using DOM manipulation
        display.innerHTML = "";

        const iconSpan = safeElement("span", data.icon || "🌡", "weather-icon");
        const infoSpan = safeElement(
          "span",
          `${data.temperature} · ${data.condition}`,
          "weather-info"
        );
        const windSpan = safeElement("span", data.windspeed, "weather-wind");

        display.appendChild(iconSpan);
        display.appendChild(infoSpan);
        display.appendChild(windSpan);
      }
    })
    .catch(() => {});
}

function changeCity() {
  const select = document.getElementById("citySelect");
  const value = select.value;
  const [lat, lng, city] = value.split(",");

  fetch("/api/weather", {
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
        // XSS-safe rendering using DOM manipulation
        display.innerHTML = "";

        const iconSpan = safeElement("span", data.icon || "🌡", "weather-icon");
        const infoSpan = safeElement(
          "span",
          `${data.temperature} · ${data.condition}`,
          "weather-info"
        );
        const windSpan = safeElement("span", data.windspeed, "weather-wind");

        display.appendChild(iconSpan);
        display.appendChild(infoSpan);
        display.appendChild(windSpan);
      }
    })
    .catch(() => {});
}
