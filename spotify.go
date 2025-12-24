package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ==========================================
// SPOTIFY INTEGRATION
// ==========================================
// Display currently playing track from Spotify with album art

// SpotifyCredentials stores OAuth tokens for Spotify API
type SpotifyCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}

// SpotifyTrack represents the currently playing track
type SpotifyTrack struct {
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	AlbumArtURL string `json:"albumArtUrl"`
	IsPlaying   bool   `json:"isPlaying"`
	ProgressMs  int    `json:"progressMs"`
	DurationMs  int    `json:"durationMs"`
}

// Spotify API endpoints
const (
	spotifyAuthURL     = "https://accounts.spotify.com/authorize"
	spotifyTokenURL    = "https://accounts.spotify.com/api/token"
	spotifyPlayerURL   = "https://api.spotify.com/v1/me/player/currently-playing"
	spotifyCallbackURL = "/api/spotify/callback"
	spotifyScopes      = "user-read-playback-state user-read-currently-playing"
)

// Global Spotify state
var (
	spotifyCredentials  SpotifyCredentials
	spotifyEnabled      bool
	spotifyCredsFromEnv bool // True if credentials came from environment variables
	spotifyLastTrack    *SpotifyTrack
	spotifyAlbumArt     []int // Cached 1-bit album art bitmap
	spotifyAlbumArtURL  string
	spotifyLastFetch    time.Time         // Last time we fetched from API
	spotifyFetchError   error             // Last error from API
	spotifyFetching     bool              // Currently fetching
	spotifyPollInterval = 5 * time.Second // How often to poll Spotify

	// Scroll state for long text (marquee effect)
	spotifySongScrollStartTime   time.Time // Start time for song name scrolling
	spotifyArtistScrollStartTime time.Time // Start time for artist name scrolling
	spotifyLastSongName          string    // Track song changes to reset scroll
	spotifyLastArtistName        string    // Track artist changes to reset scroll
)

// Static music note icon (32x32px) for Spotify display
var spotifyMusicIcon = []int{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x00,
	0x00, 0x00, 0x80, 0x00, 0x00, 0x01, 0xc0, 0x00, 0x00, 0x01, 0xc0, 0x00, 0x00, 0x05, 0xc0, 0x00,
	0x00, 0x07, 0xe0, 0x00, 0x00, 0x07, 0xf0, 0x00, 0x00, 0x07, 0xf8, 0x00, 0x00, 0x07, 0xe0, 0x00,
	0x00, 0x05, 0xc0, 0x00, 0x00, 0x01, 0xc0, 0x00, 0x00, 0x01, 0xc0, 0x00, 0x00, 0x00, 0x80, 0x00,
	0x00, 0x00, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// startSpotifyPoller starts background polling for Spotify data
func startSpotifyPoller() {
	go func() {
		for {
			time.Sleep(spotifyPollInterval)

			mutex.Lock()
			enabled := spotifyEnabled
			fetching := spotifyFetching
			mutex.Unlock()

			if !enabled || fetching {
				continue
			}

			// Mark as fetching
			mutex.Lock()
			spotifyFetching = true
			mutex.Unlock()

			// Fetch current track (this is the only place we call the API)
			track, err := getCurrentlyPlayingAsync()

			mutex.Lock()
			spotifyFetching = false
			spotifyLastFetch = time.Now()
			spotifyFetchError = err
			if err == nil {
				spotifyLastTrack = track
				// Spotify album art is no longer displayed, so we don't fetch it
				// if track != nil && track.AlbumArtURL != spotifyAlbumArtURL && track.AlbumArtURL != "" {
				// 	go fetchAndCacheAlbumArt(track.AlbumArtURL)
				// }
			}
			mutex.Unlock()
		}
	}()
}

// handleSpotifyAuth initiates the OAuth flow
func handleSpotifyAuth(w http.ResponseWriter, r *http.Request) {
	mutex.Lock()
	clientID := spotifyCredentials.ClientID
	mutex.Unlock()

	if clientID == "" {
		jsonError(w, "Spotify Client ID not configured", http.StatusBadRequest)
		return
	}

	// Build the callback URL dynamically
	// Check X-Forwarded-Proto for reverse proxy environments (Render, Heroku, etc.)
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	redirectURI := fmt.Sprintf("%s://%s%s", scheme, r.Host, spotifyCallbackURL)

	// Build authorization URL
	params := url.Values{}
	params.Set("client_id", clientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", spotifyScopes)
	params.Set("show_dialog", "true") // Force consent screen

	authURL := fmt.Sprintf("%s?%s", spotifyAuthURL, params.Encode())
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// handleSpotifyCallback handles the OAuth callback from Spotify
func handleSpotifyCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		errorMsg := r.URL.Query().Get("error")
		html := fmt.Sprintf(`<!DOCTYPE html><html><head><title>Spotify Auth Failed</title></head>
			<body><h2>Authorization Failed</h2><p>Error: %s</p>
			<script>setTimeout(function(){ window.close(); }, 3000);</script></body></html>`, errorMsg)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}

	// Build the callback URL dynamically
	// Check X-Forwarded-Proto for reverse proxy environments (Render, Heroku, etc.)
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	redirectURI := fmt.Sprintf("%s://%s%s", scheme, r.Host, spotifyCallbackURL)

	// Exchange code for tokens
	mutex.Lock()
	clientID := spotifyCredentials.ClientID
	clientSecret := spotifyCredentials.ClientSecret
	mutex.Unlock()

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Spotify token exchange failed: %v", err)
		html := `<!DOCTYPE html><html><head><title>Spotify Auth Failed</title></head>
			<body><h2>Authorization Failed</h2><p>Could not exchange code for tokens</p>
			<script>setTimeout(function(){ window.close(); }, 3000);</script></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResp)

	if tokenResp.AccessToken == "" {
		html := `<!DOCTYPE html><html><head><title>Spotify Auth Failed</title></head>
			<body><h2>Authorization Failed</h2><p>No access token received</p>
			<script>setTimeout(function(){ window.close(); }, 3000);</script></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
		return
	}

	// Store tokens
	mutex.Lock()
	spotifyCredentials.AccessToken = tokenResp.AccessToken
	spotifyCredentials.RefreshToken = tokenResp.RefreshToken
	spotifyCredentials.ExpiresAt = time.Now().Unix() + int64(tokenResp.ExpiresIn) - 60 // 60s buffer
	spotifyEnabled = true
	mutex.Unlock()

	// Save to config
	go saveConfig()

	log.Println("🎵 Spotify connected successfully")

	// Success page that auto-closes
	html := `<!DOCTYPE html><html><head><title>Spotify Connected</title></head>
		<body style="font-family: sans-serif; text-align: center; padding: 50px;">
		<h2>✅ Spotify Connected!</h2><p>You can close this window.</p>
		<script>setTimeout(function(){ window.close(); }, 2000);</script></body></html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// refreshSpotifyToken refreshes an expired access token
func refreshSpotifyToken() error {
	mutex.Lock()
	refreshToken := spotifyCredentials.RefreshToken
	clientID := spotifyCredentials.ClientID
	clientSecret := spotifyCredentials.ClientSecret
	mutex.Unlock()

	if refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, _ := http.NewRequest("POST", spotifyTokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret)))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	json.NewDecoder(resp.Body).Decode(&tokenResp)

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("failed to refresh token")
	}

	mutex.Lock()
	spotifyCredentials.AccessToken = tokenResp.AccessToken
	spotifyCredentials.ExpiresAt = time.Now().Unix() + int64(tokenResp.ExpiresIn) - 60
	mutex.Unlock()

	go saveConfig()
	log.Println("🎵 Spotify token refreshed")
	return nil
}

// getCurrentlyPlayingAsync fetches the currently playing track from Spotify (called by poller)
func getCurrentlyPlayingAsync() (*SpotifyTrack, error) {
	mutex.Lock()
	accessToken := spotifyCredentials.AccessToken
	expiresAt := spotifyCredentials.ExpiresAt
	enabled := spotifyEnabled
	mutex.Unlock()

	if !enabled || accessToken == "" {
		return nil, fmt.Errorf("spotify not connected")
	}

	// Refresh token if expired
	if time.Now().Unix() >= expiresAt {
		if err := refreshSpotifyToken(); err != nil {
			return nil, err
		}
		mutex.Lock()
		accessToken = spotifyCredentials.AccessToken
		mutex.Unlock()
	}

	req, _ := http.NewRequest("GET", spotifyPlayerURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		// No track currently playing
		return nil, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("spotify API error: %s", string(body))
	}

	var playerResp struct {
		IsPlaying  bool `json:"is_playing"`
		ProgressMs int  `json:"progress_ms"`
		Item       struct {
			Name       string `json:"name"`
			DurationMs int    `json:"duration_ms"`
			Artists    []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Name   string `json:"name"`
				Images []struct {
					URL    string `json:"url"`
					Height int    `json:"height"`
					Width  int    `json:"width"`
				} `json:"images"`
			} `json:"album"`
		} `json:"item"`
	}
	json.NewDecoder(resp.Body).Decode(&playerResp)

	if playerResp.Item.Name == "" {
		return nil, nil
	}

	// Get smallest album art (64x64)
	albumArtURL := ""
	for _, img := range playerResp.Item.Album.Images {
		if img.Width == 64 || img.Height == 64 {
			albumArtURL = img.URL
			break
		}
	}
	// Fallback to last image (usually smallest)
	if albumArtURL == "" && len(playerResp.Item.Album.Images) > 0 {
		albumArtURL = playerResp.Item.Album.Images[len(playerResp.Item.Album.Images)-1].URL
	}

	// Get first artist name
	// Get all artist names
	var artistNames []string
	for _, artist := range playerResp.Item.Artists {
		artistNames = append(artistNames, artist.Name)
	}
	artistName := strings.Join(artistNames, ", ")

	return &SpotifyTrack{
		Name:        playerResp.Item.Name,
		Artist:      artistName,
		Album:       playerResp.Item.Album.Name,
		AlbumArtURL: albumArtURL,
		IsPlaying:   playerResp.IsPlaying,
		ProgressMs:  playerResp.ProgressMs,
		DurationMs:  playerResp.Item.DurationMs,
	}, nil
}

// generateSpotifyFrame creates a frame displaying the current track
// SIMPLIFIED: Shows only song name, artist, and seek bar (no album art, no header, no icons)
func generateSpotifyFrame(duration int) Frame {
	elements := []Element{}

	// Use cached track data (updated by background poller, never blocks)
	track := spotifyLastTrack
	enabled := spotifyEnabled

	if !enabled || track == nil {
		// Show "Not Playing" state - simple centered message
		msg := "~ Not Playing"
		if !enabled {
			msg = "~ Connect Spotify"
		}
		textSize := getScaledTextSize(1)
		elements = append(elements,
			Element{Type: "text", X: calcCenteredX(msg, textSize), Y: 28, Size: textSize, Value: msg},
		)
		return Frame{
			Version:  1,
			Duration: duration,
			Clear:    true,
			Elements: elements,
		}
	}

	// Layout with static icon on left, text on right
	iconX := 0
	iconY := 16

	// Add static music note icon (32x32)
	elements = append(elements, Element{
		Type:   "bitmap",
		X:      iconX,
		Y:      iconY,
		Width:  32,
		Height: 32,
		Bitmap: spotifyMusicIcon,
	})

	// Text position (starts right after icon)
	textSize := getScaledTextSize(1)
	textX := iconX + 34                // 32px icon + 2px margin
	maxDisplayWidth := 128 - textX - 4 // Remaining width with small right margin
	charWidth := 6 * textSize          // Approx pixels per character
	maxChars := maxDisplayWidth / charWidth

	// Reset scroll positions if track changed
	now := time.Now()
	if track.Name != spotifyLastSongName {
		spotifySongScrollStartTime = now
		spotifyLastSongName = track.Name
	}
	if track.Artist != spotifyLastArtistName {
		spotifyArtistScrollStartTime = now
		spotifyLastArtistName = track.Artist
	}

	// Calculate scroll positions based on time (15 pixels per second)
	pixelsPerSec := 15.0

	// Song name - with scrolling if too long
	songY := iconY + 2
	songName := normalizeText(track.Name)
	songRunes := []rune(songName)
	if len(songRunes) > maxChars {
		// Create scrolling window with wrap-around padding
		paddedSong := songName + "   " + songName
		paddedRunes := []rune(paddedSong)
		totalLen := len([]rune(songName)) + 3

		// Calculate position
		elapsed := now.Sub(spotifySongScrollStartTime).Seconds()
		scrollPos := int(elapsed * pixelsPerSec)

		// Extract visible portion
		startIdx := scrollPos % totalLen
		endIdx := startIdx + maxChars
		if endIdx > len(paddedRunes) {
			endIdx = len(paddedRunes)
		}
		songName = string(paddedRunes[startIdx:endIdx])
	}
	elements = append(elements, Element{
		Type:  "text",
		X:     textX,
		Y:     songY,
		Size:  textSize,
		Value: songName,
	})

	// Artist name - below song, with scrolling if too long
	artistY := songY + 10
	artistName := normalizeText(track.Artist)
	artistRunes := []rune(artistName)
	if len(artistRunes) > maxChars {
		// Create scrolling window
		paddedArtist := artistName + "   " + artistName
		paddedRunes := []rune(paddedArtist)
		totalLen := len([]rune(artistName)) + 3

		// Calculate position
		elapsed := now.Sub(spotifyArtistScrollStartTime).Seconds()
		scrollPos := int(elapsed * pixelsPerSec)

		startIdx := scrollPos % totalLen
		endIdx := startIdx + maxChars
		if endIdx > len(paddedRunes) {
			endIdx = len(paddedRunes)
		}
		artistName = string(paddedRunes[startIdx:endIdx])
	}
	elements = append(elements, Element{
		Type:  "text",
		X:     textX,
		Y:     artistY,
		Size:  textSize,
		Value: artistName,
	})

	// Progress/seek bar - simple bar at bottom, aligned with text
	if track.DurationMs > 0 {
		barY := artistY + 12
		barX := textX
		barWidth := 128 - textX - 4 // Match text width

		progress := float64(track.ProgressMs) / float64(track.DurationMs)
		filledWidth := int(progress * float64(barWidth))

		// Progress bar background (thin line)
		elements = append(elements, Element{
			Type:   "line",
			X:      barX,
			Y:      barY + 3,
			Width:  barWidth,
			Height: 2,
		})

		// Progress bar filled portion (thicker)
		if filledWidth > 0 {
			elements = append(elements, Element{
				Type:   "line",
				X:      barX,
				Y:      barY + 1,
				Width:  filledWidth,
				Height: 6,
			})
		}
	}

	return Frame{
		Version:  1,
		Duration: duration,
		Clear:    true,
		Elements: elements,
	}
}

// handleSpotifySettings handles Spotify configuration get/set
func handleSpotifySettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		mutex.Lock()
		response := map[string]interface{}{
			"enabled":        spotifyEnabled,
			"hasCredentials": spotifyCredentials.ClientID != "",
			"isConnected":    spotifyCredentials.RefreshToken != "",
			"credsFromEnv":   spotifyCredsFromEnv,
			"currentTrack":   spotifyLastTrack,
		}
		mutex.Unlock()
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			ClientID     string `json:"clientId,omitempty"`
			ClientSecret string `json:"clientSecret,omitempty"`
			Disconnect   bool   `json:"disconnect,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		mutex.Lock()
		if req.Disconnect {
			// Clear all spotify data
			spotifyCredentials = SpotifyCredentials{}
			spotifyEnabled = false
			spotifyLastTrack = nil
			spotifyAlbumArt = nil
			spotifyAlbumArtURL = ""
			log.Println("🎵 Spotify disconnected")
		} else {
			if req.ClientID != "" {
				spotifyCredentials.ClientID = req.ClientID
			}
			if req.ClientSecret != "" {
				spotifyCredentials.ClientSecret = req.ClientSecret
			}
		}
		response := map[string]interface{}{
			"enabled":        spotifyEnabled,
			"hasCredentials": spotifyCredentials.ClientID != "",
			"isConnected":    spotifyCredentials.RefreshToken != "",
			"status":         "updated",
		}
		mutex.Unlock()

		go saveConfig()
		json.NewEncoder(w).Encode(response)
		return
	}

	jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// normalizeText replaces accented characters with ASCII equivalents
func normalizeText(s string) string {
	replacements := map[string]string{
		"Á": "A", "À": "A", "Â": "A", "Ã": "A", "Ä": "A",
		"á": "a", "à": "a", "â": "a", "ã": "a", "ä": "a",
		"É": "E", "È": "E", "Ê": "E", "Ë": "E",
		"é": "e", "è": "e", "ê": "e", "ë": "e",
		"Í": "I", "Ì": "I", "Î": "I", "Ï": "I",
		"í": "i", "ì": "i", "î": "i", "ï": "i",
		"Ó": "O", "Ò": "O", "Ô": "O", "Õ": "O", "Ö": "O",
		"ó": "o", "ò": "o", "ô": "o", "õ": "o", "ö": "o",
		"Ú": "U", "Ù": "U", "Û": "U", "Ü": "U",
		"ú": "u", "ù": "u", "û": "u", "ü": "u",
		"Ñ": "N", "ñ": "n", "Ç": "C", "ç": "c",
		"Ý": "Y", "ý": "y", "ÿ": "y",
		// Add more as needed
	}

	for old, new := range replacements {
		s = strings.ReplaceAll(s, old, new)
	}
	return s
}
