




const AUTH_TOKEN_KEY = "esp_desk_auth_token";


let authToken = localStorage.getItem(AUTH_TOKEN_KEY);
let authRequired = false; 


async function checkAuth() {
  try {
    const res = await fetch("/api/auth/verify", {
      headers: authToken ? { Authorization: `Bearer ${authToken}` } : {},
    });
    const data = await res.json();

    authRequired = data.authRequired;

    if (!data.authRequired) {
      
      showDashboard();
      initAfterAuth(); 
      return;
    }

    if (data.authenticated) {
      showDashboard();
      initAfterAuth(); 
    } else {
      
      authToken = null;
      localStorage.removeItem(AUTH_TOKEN_KEY);
      showLogin();
    }
  } catch (err) {
    console.error("Auth check failed:", err);
    
    showDashboard();
    initAfterAuth();
  }
}


function showLogin() {
  document.getElementById("loginOverlay").style.display = "flex";
  document.getElementById("mainContainer").classList.add("blur");
  document.getElementById("loginPassword").focus();
}


function showDashboard() {
  document.getElementById("loginOverlay").style.display = "none";
  document.getElementById("mainContainer").classList.remove("blur");

  
  if (authRequired) {
    document.getElementById("logoutBtn").style.display = "block";
  }
}


async function handleLogin(event) {
  event.preventDefault();

  const password = document.getElementById("loginPassword").value;
  const loginBtn = document.getElementById("loginBtn");
  const errorDiv = document.getElementById("loginError");

  
  errorDiv.textContent = "";
  errorDiv.style.display = "none";

  
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
      localStorage.setItem(AUTH_TOKEN_KEY, authToken); 
      showDashboard();
      initAfterAuth(); 
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
  localStorage.removeItem(AUTH_TOKEN_KEY); 
  showLogin();
}


function authFetch(url, options = {}) {
  if (authToken) {
    options.headers = options.headers || {};
    options.headers["Authorization"] = `Bearer ${authToken}`;
  }
  return fetch(url, options).then((res) => {
    
    if (res.status === 401 && authRequired) {
      showLogin();
      throw new Error("Unauthorized");
    }
    return res;
  });
}
