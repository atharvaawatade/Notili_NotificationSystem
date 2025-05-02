document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('notli_token');
    const userString = localStorage.getItem('notli_user');
    let user = null;

    const apiKeyTableBody = document.getElementById('api-keys-table-body');
    const createKeyForm = document.getElementById('create-key-form');
    const keyNameInput = document.getElementById('key-name');
    const newKeyDisplay = document.getElementById('new-key-display');
    const newApiKeyElement = document.getElementById('new-api-key');
    const copyKeyButton = document.getElementById('copy-key-button');
    const logoutButton = document.getElementById('logout-button');
    const userEmailElement = document.getElementById('user-email');
    const createKeyErrorElement = document.getElementById('create-key-error');
    const listKeysErrorElement = document.getElementById('list-keys-error');

    // --- Helper Functions ---
    const checkAuth = () => {
        if (!token) {
            console.log("No token found, redirecting to login.");
            window.location.href = '/login.html';
            return false;
        }
        if (userString) {
            try {
                user = JSON.parse(userString);
                if (userEmailElement && user.email) {
                    userEmailElement.textContent = `Logged in as: ${user.email}`;
                }
            } catch (e) {
                console.error("Failed to parse user data:", e);
                // Potentially clear storage and redirect if user data is corrupted
            }
        } else {
             console.warn("User data not found in local storage.");
             // Can function without it for now, but display might be limited
        }
        return true;
    };

    const clearError = (element) => {
        if(element) element.textContent = '';
    }
    const showError = (element, message) => {
        if(element) element.textContent = message;
        console.error('Dashboard Error:', message);
    }

    const apiFetch = async (endpoint, options = {}) => {
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            }
        };
        const mergedOptions = { ...defaultOptions, ...options };
        mergedOptions.headers = { ...defaultOptions.headers, ...options.headers };

        try {
            const response = await fetch(`/api${endpoint}`, mergedOptions);
            if (response.status === 401) { // Unauthorized
                localStorage.removeItem('notli_token');
                localStorage.removeItem('notli_user');
                window.location.href = '/login.html';
                throw new Error('Unauthorized access. Redirecting to login.');
            }
            const data = await response.json();
            if (!response.ok) {
                throw new Error(data.error || `API request failed: ${response.status}`);
            }
            return data;
        } catch (error) {
             console.error(`API Fetch Error (${endpoint}):`, error);
            throw error; // Re-throw the error to be caught by the caller
        }
    };

    // --- API Key Functions ---
    const displayApiKeys = (keys) => {
        apiKeyTableBody.innerHTML = ''; // Clear existing rows or loading message
         if (!keys || keys.length === 0) {
            apiKeyTableBody.innerHTML = '<tr><td colspan="5" class="text-center py-4 text-gray-500">No API keys found. Create one above!</td></tr>';
            return;
        }

        keys.forEach(key => {
            const row = document.createElement('tr');
            const lastUsed = key.last_used ? new Date(key.last_used).toLocaleString() : 'Never';
            const createdAt = new Date(key.created_at).toLocaleString();

            row.innerHTML = `
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">${escapeHtml(key.name)}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${escapeHtml(key.prefix)}...</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${createdAt}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">${lastUsed}</td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button data-key-id="${key.id}" class="text-red-600 hover:text-red-900 delete-key-button">Delete</button>
                </td>
            `;
            apiKeyTableBody.appendChild(row);
        });

        // Add event listeners to delete buttons AFTER they are added to the DOM
        document.querySelectorAll('.delete-key-button').forEach(button => {
            button.addEventListener('click', handleDeleteKey);
        });
    };

    const loadApiKeys = async () => {
        clearError(listKeysErrorElement);
        apiKeyTableBody.innerHTML = '<tr><td colspan="5" class="text-center py-4 text-gray-500">Loading keys...</td></tr>';
        try {
            const keys = await apiFetch('/keys');
            displayApiKeys(keys);
        } catch (error) {
            showError(listKeysErrorElement, `Failed to load API keys: ${error.message}`);
            apiKeyTableBody.innerHTML = `<tr><td colspan="5" class="text-center py-4 text-red-500">Error loading keys.</td></tr>`;
        }
    };

    const handleCreateKey = async (event) => {
        event.preventDefault();
        clearError(createKeyErrorElement);
        newKeyDisplay.classList.add('hidden');
        const name = keyNameInput.value.trim();
        if (!name) {
             showError(createKeyErrorElement,'Key name cannot be empty.');
            return;
        }

        try {
            const newKeyData = await apiFetch('/keys', {
                method: 'POST',
                body: JSON.stringify({ name })
            });

            // Display the new key temporarily
            newApiKeyElement.textContent = newKeyData.apiKey;
            newKeyDisplay.classList.remove('hidden');
            keyNameInput.value = ''; // Clear input field

            // Reload the list of keys
            await loadApiKeys();

        } catch (error) {
             showError(createKeyErrorElement,`Failed to create key: ${error.message}`);
        }
    };

    const handleDeleteKey = async (event) => {
         const keyId = event.target.getAttribute('data-key-id');
         if (!keyId) return;

         if (!confirm('Are you sure you want to delete this API key? This action cannot be undone.')) {
            return;
         }

         clearError(listKeysErrorElement); // Clear previous list errors

         try {
            await apiFetch(`/keys/${keyId}`, { method: 'DELETE' });
            await loadApiKeys(); // Refresh the list
         } catch (error) {
            showError(listKeysErrorElement, `Failed to delete key: ${error.message}`);
         }
    };

     const handleCopyKey = () => {
        const keyToCopy = newApiKeyElement.textContent;
        navigator.clipboard.writeText(keyToCopy).then(() => {
            // Optional: Provide feedback like changing button text
            copyKeyButton.textContent = 'Copied!';
            setTimeout(() => { copyKeyButton.textContent = 'Copy'; }, 2000);
        }).catch(err => {
            console.error('Failed to copy key: ', err);
            alert('Failed to copy key to clipboard.');
        });
    };

    // --- Logout --- 
    const handleLogout = () => {
        localStorage.removeItem('notli_token');
        localStorage.removeItem('notli_user');
        window.location.href = '/login.html';
    };

    // --- HTML Escape Helper ---
    const escapeHtml = (unsafe) => {
        if (typeof unsafe !== 'string') return unsafe; // Handle non-strings
        return unsafe
             .replace(/&/g, "&amp;")
             .replace(/</g, "&lt;")
             .replace(/>/g, "&gt;")
             .replace(/"/g, "&quot;")
             .replace(/'/g, "&#039;");
     }

    // --- Initialization ---
    if (checkAuth()) {
        loadApiKeys();
        createKeyForm.addEventListener('submit', handleCreateKey);
        logoutButton.addEventListener('click', handleLogout);
        copyKeyButton.addEventListener('click', handleCopyKey);
        // Note: Delete button listeners are added dynamically in displayApiKeys
    }
});
