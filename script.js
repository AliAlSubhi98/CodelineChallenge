function login(event) {
  event.preventDefault();
  
  var username = document.getElementById('username').value;
  var password = document.getElementById('password').value;
  
  // Perform authorization check here (e.g., compare username and password with stored values)
  if (username === 'admin' && password === 'admin') {
    // Authorized, store login status in localStorage
    localStorage.setItem('isLoggedIn', true);
    
    // Redirect to index.html
    window.location.href = 'index.html';
  } else {
    // Unauthorized, show error message
    document.getElementById('error-message').textContent = 'Invalid username or password';
  }
}

// Attach the login event listener
document.getElementById('login-form').addEventListener('submit', login);
