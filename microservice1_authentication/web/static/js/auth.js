document.addEventListener('DOMContentLoaded', () => {
    const signupForm = document.getElementById('signup-form');
    const loginForm = document.getElementById('login-form');
    const errorMessageElement = document.getElementById('error-message');

    const clearError = () => {
        if (errorMessageElement) {
            errorMessageElement.textContent = '';
        }
    };

    const showError = (message) => {
         if (errorMessageElement) {
            errorMessageElement.textContent = message;
        }
        console.error('Auth Error:', message);
    };

    const handleAuthResponse = async (response) => {
        const data = await response.json();
        console.log('Auth response received:', data); // Debug log
        
        if (!response.ok) {
            throw new Error(data.error || `HTTP error! status: ${response.status}`);
        }
        
        if (data.token) {
            console.log('Token found:', data.token.substring(0, 8) + '...'); // Partial token log for security
            localStorage.setItem('notli_token', data.token); // Store token
            localStorage.setItem('notli_user', JSON.stringify(data.user)); // Store user info
            window.location.href = '/static/dashboard.html'; // Redirect to dashboard
        } else {
            console.error('Response data has no token:', data);
            throw new Error('No token received from server.');
        }
    };

    // --- Signup Form Handler ---
    if (signupForm) {
        signupForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            clearError();

            const email = signupForm.email.value;
            const password = signupForm.password.value;

            if (password.length < 8) {
                showError('Password must be at least 8 characters long.');
                return;
            }

            try {
                console.log('Submitting signup form:', { email });
                const response = await fetch('/api/auth/signup', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ email, password }),
                });
                console.log('Signup response status:', response.status);
                await handleAuthResponse(response);
            } catch (error) {
                showError(`Signup failed: ${error.message}`);
            }
        });
    }

    // --- Login Form Handler ---
    if (loginForm) {
        loginForm.addEventListener('submit', async (event) => {
            event.preventDefault();
            clearError();

            const email = loginForm.email.value;
            const password = loginForm.password.value;

            try {
                const response = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ email, password }),
                });
                await handleAuthResponse(response);
            } catch (error) {
                 showError(`Login failed: ${error.message}`);
            }
        });
    }
});
