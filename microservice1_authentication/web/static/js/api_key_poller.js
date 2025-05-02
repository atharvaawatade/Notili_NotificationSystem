/**
 * API Key Poller - Continuously checks for API key changes
 */
class ApiKeyPoller {
    constructor(options = {}) {
        this.options = Object.assign({
            interval: 5000, // Check every 5 seconds by default
            onKeysLoaded: null, // Callback when keys are loaded
            onError: null, // Callback when error occurs
            maxRetries: 3 // Max consecutive failures before stopping
        }, options);
        
        this.isPolling = false;
        this.lastKeyCount = 0;
        this.failureCount = 0;
        this.timerId = null;
        this.keys = [];
    }
    
    /**
     * Start polling for API key changes
     */
    start() {
        if (this.isPolling) return;
        
        this.isPolling = true;
        this.poll();
        console.log("API key polling started");
    }
    
    /**
     * Stop polling for API key changes
     */
    stop() {
        if (!this.isPolling) return;
        
        this.isPolling = false;
        if (this.timerId) {
            clearTimeout(this.timerId);
            this.timerId = null;
        }
        console.log("API key polling stopped");
    }
    
    /**
     * Perform a single poll operation
     */
    poll() {
        if (!this.isPolling) return;
        
        const token = localStorage.getItem('notli_token');
        if (!token) {
            console.warn("No authentication token found, stopping API key polling");
            this.stop();
            return;
        }
        
        fetch('/api/keys/', {
            headers: {
                'Authorization': `Bearer ${token}`
            }
        })
        .then(response => {
            if (!response.ok) {
                throw new Error(`Failed to get API keys: ${response.status}`);
            }
            return response.json();
        })
        .then(keys => {
            this.failureCount = 0; // Reset failure count on success
            
            // Check if the keys have changed
            const hasChanged = this.hasKeysChanged(keys);
            this.keys = keys;
            
            if (hasChanged && typeof this.options.onKeysLoaded === 'function') {
                this.options.onKeysLoaded(keys);
            }
            
            // Schedule next poll
            this.scheduleNextPoll();
        })
        .catch(error => {
            console.error("API key polling error:", error);
            this.failureCount++;
            
            if (typeof this.options.onError === 'function') {
                this.options.onError(error);
            }
            
            // If too many consecutive failures, stop polling
            if (this.failureCount >= this.options.maxRetries) {
                console.warn(`Stopping API key polling after ${this.failureCount} consecutive failures`);
                this.stop();
                return;
            }
            
            // Schedule next poll despite error
            this.scheduleNextPoll();
        });
    }
    
    /**
     * Schedule the next poll
     */
    scheduleNextPoll() {
        if (!this.isPolling) return;
        
        this.timerId = setTimeout(() => {
            this.poll();
        }, this.options.interval);
    }
    
    /**
     * Check if the keys have changed from the last poll
     */
    hasKeysChanged(newKeys) {
        if (this.lastKeyCount !== newKeys.length) {
            this.lastKeyCount = newKeys.length;
            return true;
        }
        
        // More sophisticated comparison could be implemented here
        // For now we just check if the count changed
        
        return false;
    }
}

// Export for use in other modules
window.ApiKeyPoller = ApiKeyPoller;
