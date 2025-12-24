package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)






func generateToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}


func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}


func isValidToken(token string) bool {
	if !authEnabled || token == "" {
		return !authEnabled 
	}

	authMutex.RLock()
	expiry, exists := authTokens[token]
	authMutex.RUnlock()

	if !exists {
		return false
	}

	if time.Now().After(expiry) {
		
		authMutex.Lock()
		delete(authTokens, token)
		authMutex.Unlock()
		return false
	}

	return true
}


func createSession() string {
	token := generateToken()
	expiry := time.Now().Add(24 * time.Hour) 

	authMutex.Lock()
	authTokens[token] = expiry
	authMutex.Unlock()

	return token
}


func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		
		if !authEnabled {
			next(w, r)
			return
		}

		
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if isValidToken(token) {
			next(w, r)
			return
		}

		
		cookie, err := r.Cookie("esp_desk_token")
		if err == nil && isValidToken(cookie.Value) {
			next(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}


func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	
	clientIP := r.RemoteAddr
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = strings.Split(forwardedFor, ",")[0]
	}

	
	if checkRateLimit(clientIP) {
		log.Printf("Rate limited login attempt from %s", clientIP)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Too many login attempts. Please try again later.",
		})
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	
	submittedHash := hashPassword(req.Password)
	if subtle.ConstantTimeCompare([]byte(submittedHash), []byte(dashboardPasswordHash)) != 1 {
		recordFailedLogin(clientIP)
		log.Printf("Failed login attempt from %s", clientIP)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid password",
		})
		return
	}

	
	clearLoginAttempts(clientIP)

	
	token := createSession()
	log.Printf("Successful login from %s", clientIP)

	
	http.SetCookie(w, &http.Cookie{
		Name:     "esp_desk_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400, 
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"token":   token,
	})
}


func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	
	if !authEnabled {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"authRequired":  false,
		})
		return
	}

	
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if isValidToken(token) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"authRequired":  true,
		})
		return
	}

	
	cookie, err := r.Cookie("esp_desk_token")
	if err == nil && isValidToken(cookie.Value) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"authenticated": true,
			"authRequired":  true,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"authenticated": false,
		"authRequired":  true,
	})
}


func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")

	if token != "" {
		authMutex.Lock()
		delete(authTokens, token)
		authMutex.Unlock()
	}

	
	cookie, err := r.Cookie("esp_desk_token")
	if err == nil {
		authMutex.Lock()
		delete(authTokens, cookie.Value)
		authMutex.Unlock()
	}

	
	http.SetCookie(w, &http.Cookie{
		Name:     "esp_desk_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}






func cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		now := time.Now()
		authMutex.Lock()
		count := 0
		for token, expiry := range authTokens {
			if now.After(expiry) {
				delete(authTokens, token)
				count++
			}
		}
		authMutex.Unlock()
		if count > 0 {
			log.Printf("Cleaned up %d expired auth tokens", count)
		}
	}
}






func checkRateLimit(ip string) bool {
	loginAttemptsMutex.RLock()
	attempt, exists := loginAttempts[ip]
	loginAttemptsMutex.RUnlock()

	if !exists {
		return false
	}

	
	if time.Since(attempt.LastReset) > loginLockoutTime {
		loginAttemptsMutex.Lock()
		delete(loginAttempts, ip)
		loginAttemptsMutex.Unlock()
		return false
	}

	return attempt.Count >= maxLoginAttempts
}


func recordFailedLogin(ip string) {
	loginAttemptsMutex.Lock()
	defer loginAttemptsMutex.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		loginAttempts[ip] = &LoginAttempt{Count: 1, LastReset: time.Now()}
		return
	}

	
	if time.Since(attempt.LastReset) > loginLockoutTime {
		attempt.Count = 1
		attempt.LastReset = time.Now()
	} else {
		attempt.Count++
	}
}


func clearLoginAttempts(ip string) {
	loginAttemptsMutex.Lock()
	delete(loginAttempts, ip)
	loginAttemptsMutex.Unlock()
}


func cleanupLoginAttempts() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		now := time.Now()
		loginAttemptsMutex.Lock()
		for ip, attempt := range loginAttempts {
			if now.Sub(attempt.LastReset) > loginLockoutTime*2 {
				delete(loginAttempts, ip)
			}
		}
		loginAttemptsMutex.Unlock()
	}
}
