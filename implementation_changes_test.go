package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func hasTextElement(elements []Element, value string) bool {
	for _, el := range elements {
		if el.Type == "text" && el.Value == value {
			return true
		}
	}
	return false
}

func TestCurrentFrameClampsInvalidIndex(t *testing.T) {
	oldFrames := frames
	oldIndex := index
	oldEsp := espRefreshDuration
	defer func() {
		frames = oldFrames
		index = oldIndex
		espRefreshDuration = oldEsp
	}()

	frames = []Frame{
		{Version: 1, Duration: 111, Clear: true, Elements: []Element{{Type: "text", Value: "A"}}},
		{Version: 1, Duration: 222, Clear: true, Elements: []Element{{Type: "text", Value: "B"}}},
	}
	index = 99
	espRefreshDuration = 3000

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/frame/current", nil)
	currentFrame(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if index != 0 {
		t.Fatalf("expected index reset to 0, got %d", index)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if int(payload["duration"].(float64)) != 3000 {
		t.Fatalf("expected duration 3000, got %v", payload["duration"])
	}
}

func TestHandleFramesPostNoDeadlockAndAdvances(t *testing.T) {
	oldFrames := frames
	oldIndex := index
	oldEsp := espRefreshDuration
	defer func() {
		frames = oldFrames
		index = oldIndex
		espRefreshDuration = oldEsp
	}()

	frames = []Frame{
		{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "1"}}},
		{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "2"}}},
	}
	index = 0
	espRefreshDuration = 1234

	done := make(chan struct{})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/frames", nil)
	go func() {
		handleFrames(rr, req)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("handleFrames POST appears blocked (possible deadlock)")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if index != 1 {
		t.Fatalf("expected index advanced to 1, got %d", index)
	}
}

func TestHandleFramesMethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/frames", nil)
	handleFrames(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestGetClientIPPrefersRemoteWhenPublic(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")

	ip := getClientIP(req)
	if ip != "8.8.8.8" {
		t.Fatalf("expected remote public ip, got %q", ip)
	}
}

func TestGetClientIPUsesForwardedWhenPrivateProxy(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "10.0.0.10:54321"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.10")

	ip := getClientIP(req)
	if ip != "203.0.113.5" {
		t.Fatalf("expected forwarded ip, got %q", ip)
	}
}

func TestGetClientIPFallsBackWhenRemoteNoPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("X-Forwarded-For", "198.51.100.1")

	ip := getClientIP(req)
	if ip != "198.51.100.1" {
		t.Fatalf("expected forwarded ip for loopback without port, got %q", ip)
	}
}

func TestHandleSettingsMethodNotAllowed(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/settings", nil)
	handleSettings(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestWriteConfigToDiskReplacesExistingConfig(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(oldWD)
	}()

	if err := os.WriteFile(configFile, []byte(`{"stale":true}`), 0o644); err != nil {
		t.Fatalf("seed config write failed: %v", err)
	}

	autoPlay = true
	showHeaders = false
	frameDuration = 777
	espRefreshDuration = 4000
	gifFps = 0
	displayRotation = 0
	cycleItems = []CycleItem{{ID: "time-1", Type: "time", Label: "Time", Enabled: true, Duration: 3000}}
	cycleItemCounter = 1
	currentCity = "Bangalore"
	cityLat = 12.96
	cityLng = 77.57
	timezoneName = "Asia/Kolkata"

	writeConfigToDisk()

	path := filepath.Join(tmp, configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config failed: %v", err)
	}

	var cfg PersistentConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("invalid config json: %v", err)
	}
	if cfg.FrameDuration != 777 {
		t.Fatalf("expected frameDuration 777, got %d", cfg.FrameDuration)
	}
}

func TestGenerateAnalogFrameHonorsHeaderFlag(t *testing.T) {
	frameNoHeader := generateAnalogFrame(3000, time.UTC, false, false, false)
	if hasTextElement(frameNoHeader.Elements, "= CLOCK =") {
		t.Fatal("did not expect clock header when headers=false")
	}

	frameHeader := generateAnalogFrame(3000, time.UTC, true, false, false)
	if !hasTextElement(frameHeader.Elements, "= CLOCK =") {
		t.Fatal("expected clock header when headers=true")
	}
}

func TestGenerateBCDFrameHonorsHeaderFlag(t *testing.T) {
	frameNoHeader := generateBCDFrame(3000, time.UTC, false, true, true)
	if hasTextElement(frameNoHeader.Elements, "= BCD CLOCK =") {
		t.Fatal("did not expect bcd header when headers=false")
	}

	frameHeader := generateBCDFrame(3000, time.UTC, true, true, true)
	if !hasTextElement(frameHeader.Elements, "= BCD CLOCK =") {
		t.Fatal("expected bcd header when headers=true")
	}
}

func TestGenerateWordClockFrameHonorsHeaderFlag(t *testing.T) {
	frameNoHeader := generateWordClockFrame(3000, time.UTC, false)
	if hasTextElement(frameNoHeader.Elements, "= WORD CLOCK =") {
		t.Fatal("did not expect word clock header when headers=false")
	}

	frameHeader := generateWordClockFrame(3000, time.UTC, true)
	if !hasTextElement(frameHeader.Elements, "= WORD CLOCK =") {
		t.Fatal("expected word clock header when headers=true")
	}
}

func TestGenerateMoonPhaseFrameHonorsHeaderFlag(t *testing.T) {
	data := MoonPhaseData{PhaseName: "Full Moon", Illumination: 1.0}

	frameNoHeader := generateMoonPhaseFrame(3000, data, false)
	if len(frameNoHeader.Elements) != 1 || frameNoHeader.Elements[0].Type != "bitmap" {
		t.Fatal("expected only bitmap when headers=false")
	}

	frameHeader := generateMoonPhaseFrame(3000, data, true)
	if len(frameHeader.Elements) < 2 {
		t.Fatal("expected extra text when headers=true")
	}
}

func TestGenerateSnakeFrameHonorsHeaderFlag(t *testing.T) {
	initSnakeGame()
	frameNoHeader := generateSnakeFrame(3000, false)
	if hasTextElement(frameNoHeader.Elements, "= SNAKE =") {
		t.Fatal("did not expect snake header when headers=false")
	}

	initSnakeGame()
	frameHeader := generateSnakeFrame(3000, true)
	if !hasTextElement(frameHeader.Elements, "= SNAKE =") {
		t.Fatal("expected snake header when headers=true")
	}

	foundScore := false
	for _, el := range frameHeader.Elements {
		if el.Type == "text" && strings.HasPrefix(el.Value, "Score:") {
			foundScore = true
			break
		}
	}
	if !foundScore {
		t.Fatal("expected score text when headers=true")
	}
}

func TestHandleCustomTextDisablesGifMode(t *testing.T) {
	oldFrames := frames
	oldIndex := index
	oldCustomMode := isCustomMode
	oldGifMode := isGifMode
	defer func() {
		frames = oldFrames
		index = oldIndex
		isCustomMode = oldCustomMode
		isGifMode = oldGifMode
	}()

	isCustomMode = false
	isGifMode = true
	frames = []Frame{{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "old"}}}}
	index = 0

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/custom/text", strings.NewReader(`{"text":"Hello","centered":true}`))
	handleCustomText(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if isGifMode {
		t.Fatal("expected isGifMode=false after custom text")
	}
	if !isCustomMode {
		t.Fatal("expected isCustomMode=true after custom text")
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if !hasTextElement(frames[0].Elements, "Hello") {
		t.Fatal("expected custom text frame to contain submitted text")
	}
}

func TestHandleCustomDisablesGifMode(t *testing.T) {
	oldFrames := frames
	oldIndex := index
	oldCustomMode := isCustomMode
	oldGifMode := isGifMode
	defer func() {
		frames = oldFrames
		index = oldIndex
		isCustomMode = oldCustomMode
		isGifMode = oldGifMode
	}()

	isCustomMode = false
	isGifMode = true
	frames = nil
	index = 0

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/custom", strings.NewReader(`{"text":"Plain"}`))
	handleCustom(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if isGifMode {
		t.Fatal("expected isGifMode=false after custom content")
	}
	if !isCustomMode {
		t.Fatal("expected isCustomMode=true after custom content")
	}
	if len(frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(frames))
	}
	if !hasTextElement(frames[0].Elements, "Plain") {
		t.Fatal("expected custom frame to contain submitted text")
	}
}

func TestApplyAutoFramesSkipsWhenCustomMode(t *testing.T) {
	oldFrames := frames
	oldIndex := index
	defer func() {
		frames = oldFrames
		index = oldIndex
	}()

	frames = []Frame{{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "keep"}}}}
	index = 0
	newFrames := []Frame{{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "replace"}}}}

	applyAutoFrames(newFrames, true)

	if len(frames) != 1 {
		t.Fatalf("expected frame count to stay 1, got %d", len(frames))
	}
	if !hasTextElement(frames[0].Elements, "keep") {
		t.Fatal("expected existing frame to remain unchanged in custom mode")
	}
}

func TestApplyAutoFramesReplacesWhenNotCustomMode(t *testing.T) {
	oldFrames := frames
	oldIndex := index
	defer func() {
		frames = oldFrames
		index = oldIndex
	}()

	frames = []Frame{{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "old"}}}}
	index = 99
	newFrames := []Frame{{Version: 1, Duration: 100, Clear: true, Elements: []Element{{Type: "text", Value: "new"}}}}

	applyAutoFrames(newFrames, false)

	if len(frames) != 1 {
		t.Fatalf("expected frame count to be 1, got %d", len(frames))
	}
	if !hasTextElement(frames[0].Elements, "new") {
		t.Fatal("expected frames to be replaced when not in custom mode")
	}
	if index != 0 {
		t.Fatalf("expected index clamped to 0, got %d", index)
	}
}
