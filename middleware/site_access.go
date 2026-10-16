package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SiteAccessGuard is registered before all routes, including assets and NoRoute.
// It only denies requests; normal route authentication still runs afterwards.
func SiteAccessGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		blocked, err := model.IsIPBlocked(c.ClientIP())
		if err != nil {
			c.Header("Cache-Control", "no-store")
			abortWithWebMessage(c, http.StatusServiceUnavailable, "SITE_ACCESS_UNAVAILABLE", common.NewMessage("Database error, please contact the administrator"))
			return
		}
		if blocked {
			c.Header("Cache-Control", "no-store")
			abortWithWebMessage(c, http.StatusForbidden, "SITE_ACCESS_DENIED", common.NewMessage("Access denied"))
			return
		}
		if model.DB == nil {
			c.Next()
			return
		}
		// A plain device UUID, user ID, session hint or proxy header is never identity.
		if uid, device := service.ReadSiteIdentityCookie(c); uid > 0 && DenyDisabledUser(c, uid, device) {
			return
		}
		if raw, err := c.Cookie(service.RefreshCookieName); err == nil {
			if uid, device := service.IdentifyRefreshTokenOwner(raw); uid > 0 && DenyDisabledUser(c, uid, device) {
				return
			}
		}
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if token, ok := authorizationToken(raw); ok {
			raw = token
		}
		// Match the relay's supported key transports without changing their precedence.
		if key := c.GetHeader("x-api-key"); key != "" && (strings.Contains(c.Request.URL.Path, "/v1/messages") || strings.Contains(c.Request.URL.Path, "/v1/models")) {
			raw = key
		}
		path := c.Request.URL.Path
		if path == "/v1/models" || strings.HasPrefix(path, "/v1beta/models") || strings.HasPrefix(path, "/v1beta/openai/models") || strings.HasPrefix(path, "/v1/models/") {
			if key := c.Query("key"); key != "" {
				raw = key
			}
			if key := c.GetHeader("x-goog-api-key"); key != "" {
				raw = key
			}
		}
		if raw == "" || raw == "midjourney-proxy" {
			raw = c.GetHeader("mj-api-secret")
		}
		// WebSocket auth is read from a clone; the existing relay owns header mutation.
		if raw == "" {
			headers := c.Request.Header.Clone()
			applyWebSocketSubprotocolAuthorization(headers)
			raw, _ = authorizationToken(headers.Get("Authorization"))
		}
		if raw != "" && len(raw) <= 8192 {
			identity, internal, err := service.ParseDashboardAccessToken(raw)
			if internal {
				if err == nil {
					device := ""
					if session, e := model.GetUserSessionBySID(identity.SessionID); e == nil && session.UserID == identity.UserID {
						device = session.DeviceID
					}
					if DenyDisabledUser(c, identity.UserID, device) {
						return
					}
				}
			} else if strings.HasPrefix(raw, model.AccessTokenPrefix) {
				lookup := lookupAccessToken(c, raw)
				if lookup.user != nil && DenyDisabledUser(c, lookup.user.Id, "") {
					return
				}
			} else {
				// Legacy dashboard tokens only resolve during their original transition.
				if user, e := model.ValidateAccessToken(raw); e == nil && user != nil && DenyDisabledUser(c, user.Id, "") {
					return
				}
				key := strings.TrimPrefix(raw, "sk-")
				key, _, _ = strings.Cut(key, "-")
				if key != "" {
					if token, e := model.GetTokenByKey(key, false); e == nil && DenyDisabledUser(c, token.UserId, "") {
						return
					}
				}
			}
		}
		c.Next()
	}
}

// DenyDisabledUser must only receive a credential-verified user ID and a
// server-bound device ID. It never authenticates the request as that user.
func DenyDisabledUser(c *gin.Context, userID int, deviceID string) bool {
	user, err := model.GetUserCache(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	if err != nil {
		c.Header("Cache-Control", "no-store")
		abortWithWebMessage(c, http.StatusServiceUnavailable, "SITE_ACCESS_UNAVAILABLE", common.NewMessage("Database error, please contact the administrator"))
		return true
	}
	if user.Status != common.UserStatusDisabled {
		return false
	}
	if service.MayPersistSiteIdentity(c) {
		blocked, err := model.BlockDisabledUserRequest(userID, c.ClientIP(), deviceID)
		if err != nil {
			common.SysError(common.LogText("failed to persist disabled user site access block"))
		}
		if err == nil && !blocked {
			return false
		}
		if deviceID != "" {
			service.WriteSiteIdentityCookie(c, userID, deviceID)
		}
	}
	c.Header("Cache-Control", "no-store")
	abortWithWebMessage(c, http.StatusForbidden, "SITE_ACCESS_DENIED", common.NewMessage("Access denied"))
	return true
}
