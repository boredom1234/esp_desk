# 🧠 GOLDEN RULE (Read This First)

> **The ESP32 must NEVER decide _what_ to show.**
> It only decides **how to draw instructions** it receives.

If you keep this rule, your system will stay clean forever.

---

# 1️⃣ Keep a Strict Contract (Backend ↔ ESP32)

### ✅ Rule

Once a JSON field exists, **do not change its meaning**.

### ❌ Don’t

```json
"size": "large"     // bad (string today, int tomorrow?)
```

### ✅ Do

```json
"size": 2           // numeric, predictable
```

### Best Practice

- Treat your JSON like an **API version**
- Add fields, **never rename/remove**
- ESP32 should ignore unknown fields

---

# 2️⃣ Add New Element Types — Don’t Modify Old Ones

### ❌ Bad

> “Let me add graph support by changing text behavior”

### ✅ Good

```json
{ "type": "graph", "data": [1, 2, 3] }
```

### ESP32 Code Pattern

```cpp
if (type == "text") { ... }
else if (type == "bar") { ... }
else if (type == "graph") { ... }
```

Never break existing types.

---

# 3️⃣ Backend Owns Logic, ESP32 Owns Rendering

### Backend Responsibilities

✔ Page order
✔ Timing
✔ What data is shown
✔ Conditional pages (alerts, errors)

### ESP32 Responsibilities

✔ Drawing pixels
✔ Fonts
✔ Placement
✔ Screen refresh

📌 If you catch yourself adding logic like:

> “If temperature > 40, show alert”

❌ **STOP** — backend should decide that.

---

# 4️⃣ Always Provide Defaults on ESP32

### Why

Backend bugs should not crash your device.

### Example

```cpp
int x = el["x"] | 0;
int y = el["y"] | 0;
int size = el["size"] | 1;
int duration = doc["duration"] | 3000;
```

📌 This makes your ESP32 **bulletproof**.

---

# 5️⃣ Never Block Forever (Watchdog Safety)

### ❌ Bad

```cpp
while (true) {}
```

### ✅ Good

```cpp
if (httpCode != 200) {
  delay(1000);
  return;
}
```

Add **timeouts** and **fallback delays**.

---

# 6️⃣ Keep the ESP32 Stateless

### ❌ Bad

```cpp
currentPage++;
```

### ✅ Good

```cpp
GET /frame/next
```

ESP32:

- Does not remember state
- Can reboot anytime
- Backend stays in control

This is how **real IoT systems** are built.

---

# 7️⃣ Version Your Protocol (Future-Proof)

Add this **early**, even if unused:

```json
{
  "version": 1,
  "duration": 3000,
  "elements": [...]
}
```

ESP32:

```cpp
int version = doc["version"] | 1;
```

When you need breaking changes:

- Introduce version 2
- Keep version 1 working

---

# 8️⃣ Enforce Screen Limits in ESP32

Your OLED is **128×64**.
Never trust backend coordinates blindly.

### Safe Guard

```cpp
if (x < 0 || x > 127) continue;
if (y < 0 || y > 63) continue;
```

This prevents rendering bugs and memory issues.

---

# 9️⃣ Keep Rendering Fast & Predictable

### Rules

- No delays inside `drawFrame()`
- No WiFi calls inside drawing
- No dynamic memory allocation per frame

Rendering must be **pure & deterministic**.

---

# 🔟 Add One Feature at a Time (Non-Negotiable)

Every new feature must follow this order:

1. Extend **backend JSON**
2. Test backend via `curl`
3. Add ESP32 rendering support
4. Test with **one frame only**
5. Then add to rotation

Never add backend + ESP32 + UI changes all at once.

---

# 1️⃣1️⃣ Logging Saves You

### ESP32

```cpp
Serial.println("Rendering text");
```

### Backend

```go
log.Println("Serving frame", index)
```

Disable logs later — but **use them now**.

---

# 1️⃣2️⃣ Canonical Element Checklist

Before adding a new element type, answer:

- What is its `type`?
- What fields are required?
- What fields are optional?
- What are safe defaults?
- How does it fail gracefully?

If you can’t answer → don’t add it yet.

---

# 1️⃣3️⃣ Things You Can Safely Add (Without Risk)

These are **safe expansions**:

✔ New element types
✔ New optional JSON fields
✔ New pages
✔ New animations
✔ New backend logic
✔ More ESP32 devices

These are **dangerous**:

❌ Renaming fields
❌ ESP32-side logic
❌ Hardcoded layouts
❌ Blocking loops
❌ Tight coupling

---

# 🧠 Mental Model (Remember This)

> **Backend = Brain** > **ESP32 = GPU**

If you treat it like this, your system will stay clean, scalable, and impressive.

---

---

# 📦 BACKEND-ONLY FEATURE GUIDE

**Goal**: Add new features by modifying ONLY backend (Go) and frontend (HTML/JS) — WITHOUT touching `main.ino`

---

## 🔧 What the ESP32 Already Supports (Use These!)

The firmware already renders these element types. **Reuse them** instead of creating new ones:

| Element Type | Fields                        | Use For                                    |
| ------------ | ----------------------------- | ------------------------------------------ |
| `text`       | `x, y, size, value`           | Any text display (time, labels, values)    |
| `bitmap`     | `x, y, width, height, bitmap` | Images, QR codes, icons, graphs            |
| `line`       | `x, y, width, height`         | Frames, borders, separators, progress bars |

**Key insight**: If you can express your feature using text + lines + bitmaps, you don't need to modify `main.ino`!

---

## ✅ Checklist for Adding a New Cycle Item Type

### Step 1: Backend Data Model (`types.go`)

Add new fields to `CycleItem` struct:

```go
// Example: Adding a "quote" type
Quote       string `json:"quote,omitempty"`       // For quote: the quote text
QuoteAuthor string `json:"quoteAuthor,omitempty"` // For quote: attribution
```

**Rules**:

- Use `omitempty` to keep JSON clean
- Use clear, descriptive field names
- Document with comments

---

### Step 2: Backend Frame Generation (`background.go`)

Add a new `case` in the `switch item.Type` block inside `updateLoop()`:

```go
case "quote":
    if item.Quote != "" {
        elements := []Element{
            {Type: "text", X: 4, Y: 20, Size: 1, Value: item.Quote},
        }
        if item.QuoteAuthor != "" {
            elements = append(elements, Element{
                Type: "text", X: 60, Y: 50, Size: 1, Value: "— " + item.QuoteAuthor,
            })
        }
        frames = append(frames, Frame{
            Version: 1, Duration: duration, Clear: true, Elements: elements,
        })
    }
```

**Rules**:

- Build frames using only `text`, `bitmap`, and `line` elements
- Always check for empty/nil data
- Use `calcCenteredX()` for centered text
- Set reasonable default duration

---

### Step 3: Frontend Dropdown (`index.html`)

Add option to the cycle widget selector:

```html
<option value="quote">💬 Quote</option>
```

---

### Step 4: Frontend Config Panel (`index.html`)

Add a hidden configuration panel:

```html
<div class="quote-item-config" id="quoteItemConfig" style="display: none">
  <input type="text" id="quoteText" placeholder="Enter quote..." />
  <input type="text" id="quoteAuthor" placeholder="Author (optional)" />
  <button onclick="confirmAddQuote()">Save</button>
</div>
```

---

### Step 5: Frontend JavaScript (`cycle.js` or new file)

1. Update `getTypeIcon()`:

```javascript
quote: "💬",
```

2. Update `addCycleItem()`:

```javascript
if (type === "quote") {
  document.getElementById("quoteItemConfig").style.display = "block";
  return;
}
```

3. Add `confirmAdd[Type]()` function:

```javascript
function confirmAddQuote() {
  const quote = document.getElementById("quoteText").value.trim();
  const author = document.getElementById("quoteAuthor").value.trim();

  cycleItems.push({
    id: `quote-${Date.now()}`,
    type: "quote",
    label: "💬 Quote",
    quote: quote,
    quoteAuthor: author,
    enabled: true,
    duration: 5000,
  });

  saveCycleItems();
  renderCycleItems(cycleItems);
  // Reset and hide panel
}
```

---

## 🖼️ Adding Bitmap-Based Features

For graphics, charts, QR codes, icons:

1. **Generate bitmap on backend** (Go):

```go
func generateMyFeatureBitmap(data interface{}) ([]int, int, int, error) {
    // Create image using Go's image package
    // Convert to 1-bit monochrome ([]int where each int is 0 or 1)
    // Return bitmap, width, height, error
}
```

2. **Use existing bitmap element**:

```go
{Type: "bitmap", X: offsetX, Y: offsetY, Width: w, Height: h, Bitmap: bitmap}
```

---

## 📊 Progress Bars (Using Lines)

Create horizontal progress bars with `line` elements:

```go
// Background bar
{Type: "line", X: 4, Y: 32, Width: 120, Height: 8},
// Fill bar (width based on percentage)
{Type: "line", X: 5, Y: 33, Width: int(118 * percentage), Height: 6},
```

---

## 🚨 Adding "Display Now" Features

For immediate display (bypassing cycle):

1. **Create API endpoint** (`your_feature.go`):

```go
func handleYourFeature(w http.ResponseWriter, r *http.Request) {
    // Parse request
    // Generate frame
    mutex.Lock()
    isCustomMode = true
    isGifMode = false
    frames = []Frame{frame}
    index = 0
    mutex.Unlock()
    // Return success
}
```

2. **Register endpoint** (`main.go`):

```go
http.HandleFunc("/api/yourfeature", loggingMiddleware(authMiddleware(handleYourFeature)))
```

3. **Call from frontend**:

```javascript
await authFetch("/api/yourfeature", { method: "POST", ... });
```

---

## 📏 Display Constraints

Always remember:

- OLED is **128×64 pixels**
- Font size 1 = ~6×8 pixels per character (~21 chars per line)
- Font size 2 = ~12×16 pixels per character (~10 chars per line)
- Bitmap maximum = 1024 bytes (128×64 / 8)

---

## 🔍 Debugging Tips

1. **Check logs**: `go run .` shows all backend activity
2. **Browser console**: F12 → Console for JS errors
3. **Test API first**: Use Invoke-RestMethod before UI:

```powershell
Invoke-RestMethod -Uri "http://localhost:3000/api/settings" -Method Get
```

4. **Verify frame generation**: Add `log.Printf` in `background.go`

---

## 📁 File Reference

| File            | Purpose          | When to Modify                |
| --------------- | ---------------- | ----------------------------- |
| `types.go`      | Data structures  | Add new CycleItem fields      |
| `background.go` | Frame generation | Add new frame rendering logic |
| `main.go`       | API routes       | Register new endpoints        |
| `[feature].go`  | Feature logic    | Create for complex features   |
| `index.html`    | UI structure     | Add config panels, dropdowns  |
| `cycle.js`      | Cycle management | Handle new item types         |
| `[feature].js`  | Feature UI       | Create for complex UI logic   |

---

## ✅ Quick Reference: What's Safe vs. Dangerous

### ✅ SAFE (Backend-Only)

- New cycle item types using text/bitmap/line
- New API endpoints
- New frontend panels and controls
- Data fetched from external APIs
- Complex calculations and logic
- Animated content (pre-generated frames)

### ❌ REQUIRES FIRMWARE CHANGE

- New element types (e.g., `circle`, `arc`)
- New rendering methods
- Hardware changes (new pins, sensors)
- New LED effects beyond existing modes
- WebSocket instead of polling
