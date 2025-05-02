package utils

import (
	"net/http"
	"time"

	"github.com/appointy/notli/microservice1_authentication/internal/config"
	"github.com/gin-gonic/gin"
)

const (
	// SessionCookieName is the name of the cookie that stores the session ID
	SessionCookieName = "notli_session"
	
	// RefreshTokenCookieName is the name of the cookie that stores the refresh token
	RefreshTokenCookieName = "notli_refresh_token"
)

// SetSessionCookie sets a secure session cookie
func SetSessionCookie(c *gin.Context, cfg *config.Config, sessionID string, expiry time.Time) {
	cookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Domain:   cfg.CookieDomainName,
		Expires:  expiry,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
	
	http.SetCookie(c.Writer, cookie)
}

// SetRefreshTokenCookie sets a secure refresh token cookie
func SetRefreshTokenCookie(c *gin.Context, cfg *config.Config, refreshToken string, expiry time.Time) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		Domain:   cfg.CookieDomainName,
		Expires:  expiry,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
	
	http.SetCookie(c.Writer, cookie)
}

// GetSessionCookie retrieves the session ID from the cookie
func GetSessionCookie(c *gin.Context) (string, error) {
	cookie, err := c.Request.Cookie(SessionCookieName)
	if err != nil {
		return "", err
	}
	
	return cookie.Value, nil
}

// GetRefreshTokenCookie retrieves the refresh token from the cookie
func GetRefreshTokenCookie(c *gin.Context) (string, error) {
	cookie, err := c.Request.Cookie(RefreshTokenCookieName)
	if err != nil {
		return "", err
	}
	
	return cookie.Value, nil
}

// ClearSessionCookies removes all session-related cookies
func ClearSessionCookies(c *gin.Context, cfg *config.Config) {
	// Clear session cookie
	sessionCookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cfg.CookieDomainName,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, sessionCookie)
	
	// Clear refresh token cookie
	refreshCookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cfg.CookieDomainName,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.SecureCookies,
		SameSite: http.SameSiteStrictMode,
	}
	http.SetCookie(c.Writer, refreshCookie)
}
