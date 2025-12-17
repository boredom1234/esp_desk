// ==========================================
// ESP DESK_OS - Authentication
// ==========================================

// Storage key for auth token
const AUTH_TOKEN_KEY = "esp_desk_auth_token";

// Restore token from localStorage on load
let authToken = localStorage.getItem(AUTH_TOKEN_KEY);
let authRequired = false; // Global flag to indicate if auth is enabled on the server

// Check authentication status on page load
async function checkAuth() {
  try {
    const res = await fetch("/api/auth/verify", {
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
    });
    const data = await res.json();

    authRequired = data.authRequired;

    if (!data.authRequired) {
      // Auth not enabled on server, show dashboard directly
      showDashboard();
      initAfterAuth(); // Initialize app after auth verified
      return;
    }

    if (data.authenticated) {
      showDashboard();
      initAfterAuth(); // Initialize app after auth verified
    } else {
      // Token invalid, clear it
      authToken = null;
      localStorage.removeItem(AUTH_TOKEN_KEY);
      showLogin();
    }
  } catch (err) {
    console.error("Auth check failed:", err);
    // On error, try to show dashboard (might work if no auth)
    showDashboard();
    initAfterAuth();
  }
}

// Show login overlay
function showLogin() {
  document.getElementById("loginOverlay").style.display = "flex";
  document.getElementById("mainContainer").classList.add("blur");
  document.getElementById("loginPassword").focus();
}

// Show dashboard (hide login)
function showDashboard() {
  document.getElementById("loginOverlay").style.display = "none";
  document.getElementById("mainContainer").classList.remove("blur");

  // Show logout button if auth is enabled
  if (authRequired) {
    document.getElementById("logoutBtn").style.display = "block";
  }
}

// Handle login form submission
async function handleLogin(event) {
  event.preventDefault();

  const password = document.getElementById("loginPassword").value;
  const loginBtn = document.getElementById("loginBtn");
  const errorDiv = document.getElementById("loginError");

  // Clear previous error
  errorDiv.textContent = "";
  errorDiv.style.display = "none";

  // Show loading state
  loginBtn.disabled = true;
  loginBtn.textContent = "Authenticating...";

  try {
    const res = await fetch("/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    });

    const data = await res.json();

    if (data.success) {
      authToken = data.token;
      localStorage.setItem(AUTH_TOKEN_KEY, authToken); // Persist token
      showDashboard();
      initAfterAuth(); // Initialize app after login
    } else {
      errorDiv.textContent = data.error || "Invalid password";
      errorDiv.style.display = "block";
      document.getElementById("loginPassword").value = "";
      document.getElementById("loginPassword").focus();
    }
  } catch (err) {
    console.error("Login failed:", err);
    errorDiv.textContent = "Connection error. Please try again.";
    errorDiv.style.display = "block";
  } finally {
    loginBtn.disabled = false;
    loginBtn.textContent = "Access Dashboard";
  }
}

// Handle logout
async function handleLogout() {
  try {
    await fetch("/api/auth/logout", {
      method: "POST",
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
    });
  } catch (err) {
    console.error("Logout error:", err);
  }

  authToken = null;
  localStorage.removeItem(AUTH_TOKEN_KEY); // Clear persisted token
  showLogin();
}

// Add auth header to fetch requests
function authFetch(url, options = {}) {
  if (authToken) {
    options.headers = options.headers || {};
    options.headers["Authorization"] = `Bearer ${authToken}`;
  }
  return fetch(url, options).then((res) => {
    // If we get 401, show login
    if (res.status === 401 && authRequired) {
      showLogin();
      throw new Error("Unauthorized");
    }
    return res;
  });
}
