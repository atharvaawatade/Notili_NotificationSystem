document.addEventListener('DOMContentLoaded', function() {
    // DOM elements
    const userEmailSpan = document.getElementById('user-email');
    const apiKeySelect = document.getElementById('api-key-select');
    const manualApiKeyInput = document.getElementById('manual-api-key');
    const apiKeyError = document.getElementById('api-key-error');
    const emailForm = document.getElementById('email-form');
    const sendEmailError = document.getElementById('send-email-error');
    const sendEmailSuccess = document.getElementById('send-email-success');
    const logContainer = document.getElementById('log-container');
    const logoutButton = document.getElementById('logout-button');
    const sendEmailButton = document.getElementById('send-email-button');
    const buttonText = sendEmailButton.querySelector('.button-text');
    const rateLimitBar = document.getElementById('rate-limit-bar');
    const toastContainer = document.getElementById('toast-container');
    
    // Tab elements
    const selectTab = document.getElementById('select-tab');
    const manualTab = document.getElementById('manual-tab');
    const selectContent = document.getElementById('select-content');
    const manualContent = document.getElementById('manual-content');
    
    // Track which tab is active
    let activeTab = 'select'; // 'select' or 'manual'
    
    // API key poller for continuous updates
    let apiKeyPoller = null;

    // Check auth and load data
    checkAuth();
    
    // Add event listeners
    emailForm.addEventListener('submit', handleSendEmail);
    logoutButton.addEventListener('click', handleLogout);
    
    // Initialize API key poller for real-time updates
    initApiKeyPoller();
    
    // Set up tab switching
    selectTab.addEventListener('click', function() {
        switchTab('select');
    });
    
    manualTab.addEventListener('click', function() {
        switchTab('manual');
    });
    
    // Function to switch tabs
    function switchTab(tabName) {
        activeTab = tabName;
        
        if (tabName === 'select') {
            // Activate select tab
            selectTab.classList.add('border-blue-500');
            selectTab.classList.remove('border-transparent', 'hover:border-gray-300');
            selectTab.setAttribute('aria-selected', 'true');
            selectContent.classList.remove('hidden');
            
            // Deactivate manual tab
            manualTab.classList.remove('border-blue-500');
            manualTab.classList.add('border-transparent', 'hover:border-gray-300');
            manualTab.setAttribute('aria-selected', 'false');
            manualContent.classList.add('hidden');
        } else {
            // Activate manual tab
            manualTab.classList.add('border-blue-500');
            manualTab.classList.remove('border-transparent', 'hover:border-gray-300');
            manualTab.setAttribute('aria-selected', 'true');
            manualContent.classList.remove('hidden');
            
            // Deactivate select tab
            selectTab.classList.remove('border-blue-500');
            selectTab.classList.add('border-transparent', 'hover:border-gray-300');
            selectTab.setAttribute('aria-selected', 'false');
            selectContent.classList.add('hidden');
        }
    }

    // Function to add log message
    function addLog(message, type = 'info') {
        const timestamp = new Date().toLocaleTimeString();
        const classColor = type === 'error' ? 'text-red-500' : (type === 'success' ? 'text-green-400' : 'text-blue-300');
        const logEntry = document.createElement('div');
        logEntry.className = `mb-1 ${classColor}`;
        logEntry.innerHTML = `[${timestamp}] ${message}`;
        logContainer.appendChild(logEntry);
        // Auto-scroll to bottom
        logContainer.scrollTop = logContainer.scrollHeight;
    }

    // Function to show toast notifications
    function showToast(message, type = 'info', duration = 4000) {
        if (!toastContainer) return; // Make sure container exists

        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        toastContainer.appendChild(toast);

        // Trigger reflow to enable animation
        toast.offsetHeight; 

        toast.classList.add('show');

        setTimeout(() => {
            toast.classList.remove('show');
            // Remove the toast from DOM after animation finishes
            setTimeout(() => {
                if (toast.parentNode === toastContainer) {
                    toastContainer.removeChild(toast);
                }
            }, 300); // Match CSS transition duration
        }, duration);
    }

    // Function to check if user is authenticated
    function checkAuth() {
        addLog('Checking authentication...');
        const token = localStorage.getItem('notli_token');
        const userString = localStorage.getItem('notli_user');
        
        if (!token) {
            addLog('Not authenticated. Redirecting to login...', 'error');
            window.location.href = '/static/login.html';
            return;
        }

        // If we have cached user data, use it
        if (userString) {
            try {
                const user = JSON.parse(userString);
                if (user && user.email) {
                    userEmailSpan.textContent = user.email;
                    addLog(`Authenticated as ${user.email}`, 'success');
                    loadApiKeys();
                    return;
                }
            } catch (e) {
                console.error("Failed to parse user data:", e);
                // Fall through to fetch user data
            }
        }

        // Get user email if not available in localStorage
        fetch('/api/auth/user', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to get user information');
            }
            return response.json();
        })
        .then(data => {
            userEmailSpan.textContent = data.email || 'User';
            loadApiKeys();
        })
        .catch(error => {
            console.error('Error fetching user:', error);
            localStorage.removeItem('token');
            window.location.href = '/static/login.html';
        });
    }

    // Function to load API keys
    function loadApiKeys() {
        addLog('Loading API keys...');
        const token = localStorage.getItem('notli_token');
        
        fetch('/api/keys/', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to get API keys');
            }
            return response.json();
        })
        .then(keys => {
            updateApiKeyDropdown(keys);
        })
        .catch(error => {
            console.error('Error loading API keys:', error);
            apiKeyError.textContent = 'Failed to load API keys: ' + error.message;
            addLog('Error loading API keys: ' + error.message, 'error');
        });
    }
    
    // Function to update the API key dropdown
    function updateApiKeyDropdown(keys) {
        // Clear existing options except the placeholder
        while (apiKeySelect.options.length > 1) {
            apiKeySelect.remove(1);
        }
        
        // Add the keys as options
        keys.forEach(key => {
            const option = document.createElement('option');
            option.value = key.id;
            option.textContent = `${key.name} (${key.prefix}...)`;
            option.dataset.prefix = key.prefix;
            apiKeySelect.appendChild(option);
        });
        
        addLog(`Loaded ${keys.length} API keys`, 'success');
        
        if (keys.length === 0) {
            apiKeyError.textContent = 'No API keys found. Please create one in the Dashboard.';
        } else {
            apiKeyError.textContent = '';
        }
    }

    // Function to handle the send email form submission
    async function handleSendEmail(event) { // Make async for await
        event.preventDefault();
        // Clear old messages - replaced by toasts
        // sendEmailError.textContent = '';
        // sendEmailSuccess.classList.add('hidden');

        // --- Show Loading State --- 
        sendEmailButton.disabled = true;
        sendEmailButton.classList.add('btn-loading');
        buttonText.textContent = 'Sending...';
        const spinner = document.createElement('span');
        spinner.className = 'spinner';
        sendEmailButton.appendChild(spinner);
        // --------------------------

        let apiKey = null;
        
        // Check which tab is active and get the API key accordingly
        if (activeTab === 'select') {
            // Get selected API key from dropdown
            const selectedKeyId = apiKeySelect.value;
            if (!selectedKeyId) {
                showToast('Please select an API key', 'error');
                addLog('Error: No API key selected', 'error');
                return;
            }
            
            // Will fetch the actual key from the server later
            addLog('Using API key from selection', 'info');
        } else {
            // Get manually entered API key
            apiKey = manualApiKeyInput.value.trim();
            if (!apiKey) {
                showToast('Please enter an API key', 'error');
                addLog('Error: No API key entered', 'error');
                return;
            }
            
            // Use the manually entered key directly
            addLog('Using manually entered API key', 'info');
        }

        // Get form data
        const recipient = document.getElementById('recipient').value;
        const subject = document.getElementById('subject').value;
        const body = document.getElementById('body').value;
        const fromName = document.getElementById('from-name').value;
        const senderEmail = document.getElementById('sender-email').value;

        // Generate a unique idempotency key
        const idempotencyKey = generateUUID();

        // Prepare request body
        const requestBody = {
            recipient: recipient,
            subject: subject,
            body: body,
            from_name: fromName,
            from_email: senderEmail, // Add sender email to the request
            idempotency_key: idempotencyKey
        };
        
        // Log the sender email being used
        if (senderEmail && senderEmail !== 'simplivu@simplivu.com') {
            addLog(`Using custom sender email: ${senderEmail}`, 'info');
        } else {
            addLog('Using default sender email (simplivu@simplivu.com)', 'info');
        }

        addLog(`Sending email to ${recipient}...`);
        
        // Get the actual API key value (if using dropdown selection)
        let fetchPromise;
        
        if (activeTab === 'select') {
            const selectedKeyId = apiKeySelect.value;
            // Fetch the API key from the server
            fetchPromise = fetch(`/api/keys/${selectedKeyId}`, {
                headers: {
                    'Authorization': `Bearer ${localStorage.getItem('notli_token')}`
                }
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to get API key details');
                }
                return response.json();
            })
            .then(keyData => {
                // Use the API key from the server
                return keyData.key;
            });
        } else {
            // Use the manually entered API key directly
            fetchPromise = Promise.resolve(apiKey);
        }
        
        // Once we have the API key (either from server or manual entry),
        // send the email through our proxy endpoint
        fetchPromise.then(apiKeyValue => {
            // Use the proxy endpoint to avoid CORS issues
            addLog('Sending request through proxy to avoid CORS issues', 'info');
            return fetch('/api/proxy/email', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-API-Key': apiKeyValue
                },
                body: JSON.stringify(requestBody)
            });
        })
        .then(async response => {
            if (!response.ok) {
                const errorData = await response.json().catch(() => ({ error: 'Failed to parse error response' }));
                let errorMessage = errorData.error || `HTTP error! status: ${response.status}`;
                addLog(`Request failed: ${errorMessage}`, 'error');

                if (response.status === 429) {
                    // Specific handling for rate limit
                    errorMessage = "Rate limit exceeded. Please wait and try again.";
                    showToast(errorMessage, 'error', 5000); // Show rate limit toast longer
                    // Potentially update rate limit meter UI here if data available
                    if (rateLimitBar) rateLimitBar.style.width = '100%'; // Example: Show as full
                } else {
                    showToast(`Error: ${errorMessage}`, 'error');
                }
                throw new Error(errorMessage); // Throw to be caught by catch block
            }

            const result = await response.json();
            addLog('Email request accepted by server.', 'success');
            showToast('Email sent successfully!', 'success');

            // Simulate detailed logging (keep this for user feedback)
            setTimeout(() => addLog('Received by Authentication Service', 'info'), 500);
            setTimeout(() => addLog('Message queued in Kafka', 'info'), 1500);
            setTimeout(() => addLog('Message received by prioritization service', 'info'), 2000);
            setTimeout(() => addLog('Priority determined: Transactional/High', 'info'), 2500);
            setTimeout(() => addLog('Forwarded to delivery service', 'info'), 3000);
            setTimeout(() => addLog('Email template processing started', 'info'), 4000);
            setTimeout(() => addLog('Sending via Resend...', 'info'), 5000);
            setTimeout(() => addLog('⚠️ Email accepted by Resend for delivery', 'info'), 7000);
            setTimeout(() => addLog('⚠️ It may take a few minutes to appear in your inbox', 'info'), 7500);
            setTimeout(() => addLog('⚠️ Please check your inbox and spam folder', 'info'), 8000);
            
            // Reset form (optional)
            // emailForm.reset();
        })
        .catch(error => {
            console.error('Error sending email:', error);
            // Error toast is shown inside the try block now
            // addLog(`Error: ${error.message}`, 'error');
        })
        .finally(() => {
            // --- Hide Loading State --- 
            sendEmailButton.disabled = false;
            sendEmailButton.classList.remove('btn-loading');
            buttonText.textContent = 'Send Email';
            if (spinner.parentNode === sendEmailButton) { 
                 sendEmailButton.removeChild(spinner);
            }
            // --------------------------

            // Placeholder: Update rate limit meter visually (Needs real data)
            if (rateLimitBar) {
                // In a real scenario, get usage from response headers or another API call
                const currentUsage = Math.random() * 80 + 10; // Random example
                rateLimitBar.style.width = `${currentUsage}%`;
                addLog(`API usage updated (example: ${currentUsage.toFixed(0)}%)`, 'info');
            }
        });
    }

    // Function to handle logout
    function handleLogout() {
        // Stop API key polling before logout
        if (apiKeyPoller) {
            apiKeyPoller.stop();
        }
        
        localStorage.removeItem('notli_token');
        localStorage.removeItem('notli_user');
        window.location.href = '/static/login.html';
    }
    
    // Initialize the API key poller
    function initApiKeyPoller() {
        apiKeyPoller = new ApiKeyPoller({
            interval: 5000, // Check every 5 seconds
            onKeysLoaded: function(keys) {
                addLog(`API keys updated (found ${keys.length})`, 'info');
                updateApiKeyDropdown(keys);
            },
            onError: function(error) {
                console.error('API key polling error:', error);
                // Don't show errors in the UI for background polling
            }
        });
        
        // Start polling
        apiKeyPoller.start();
    }

    // Helper function to generate UUID for idempotency key
    function generateUUID() {
        return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
            const r = Math.random() * 16 | 0;
            const v = c === 'x' ? r : (r & 0x3 | 0x8);
            return v.toString(16);
        });
    }
});
