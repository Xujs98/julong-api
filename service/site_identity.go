package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// SiteIdentityCookieName is a signed, deny-only device marker. It never grants
// authentication, refreshes a session, or bypasses a permission check.
const SiteIdentityCookieName = "new_api_site_identity"
const siteIdentityTTL = 400 * 24 * time.Hour

func WriteSiteIdentityCookie(c *gin.Context, userID int, deviceID string) {
	if c == nil || userID <= 0 {
		return
	}
	normalized, err := model.NormalizeDeviceID(deviceID)
	if err != nil {
		return
	}
	expires := time.Now().Add(siteIdentityTTL)
	payload := strconv.Itoa(userID) + "." + normalized + "." + strconv.FormatInt(expires.Unix(), 10)
	mac := hmac.New(sha256.New, authSigningKey("site_identity"))
	_, _ = mac.Write([]byte(payload))
	http.SetCookie(c.Writer, &http.Cookie{
		Name: SiteIdentityCookieName, Value: payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)),
		Path: "/", MaxAge: int(siteIdentityTTL / time.Second), Expires: expires,
		HttpOnly: true, Secure: common.SessionCookieSecure, SameSite: http.SameSiteLaxMode,
	})
}

func ReadSiteIdentityCookie(c *gin.Context) (int, string) {
	raw, err := c.Cookie(SiteIdentityCookieName)
	if err != nil || len(raw) > 256 {
		return 0, ""
	}
	parts := strings.Split(raw, ".")
	if len(parts) != 4 {
		return 0, ""
	}
	uid, err := strconv.Atoi(parts[0])
	if err != nil || uid <= 0 {
		return 0, ""
	}
	device, err := model.NormalizeDeviceID(parts[1])
	if err != nil {
		return 0, ""
	}
	expires, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || expires <= time.Now().Unix() {
		return 0, ""
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return 0, ""
	}
	mac := hmac.New(sha256.New, authSigningKey("site_identity"))
	_, _ = mac.Write([]byte(strings.Join(parts[:3], ".")))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return 0, ""
	}
	return uid, device
}

// IdentifyRefreshTokenOwner verifies the secret even for a revoked session.
// Knowing a session ID alone must never be enough to poison an IP blacklist.
func IdentifyRefreshTokenOwner(raw string) (int, string) {
	sid, secret, ok := splitRefreshToken(raw)
	if !ok {
		return 0, ""
	}
	session, err := model.GetUserSessionBySID(sid)
	if err != nil || session.ExpiresAt <= time.Now().Unix() {
		return 0, ""
	}
	hash := hashRefreshSecret(secret)
	current := hmac.Equal([]byte(hash), []byte(session.RefreshHash))
	previous := session.PreviousRefreshHash != "" && time.Now().Unix() <= session.PreviousValidUntil && hmac.Equal([]byte(hash), []byte(session.PreviousRefreshHash))
	if !current && !previous {
		return 0, ""
	}
	return session.UserID, session.DeviceID
}

// MayPersistSiteIdentity rejects browser cross-origin credential injection.
// Native API clients have no fetch metadata; ordinary authentication still applies.
func MayPersistSiteIdentity(c *gin.Context) bool {
	if site := c.GetHeader("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	for _, header := range []string{"Origin", "Referer"} {
		value := c.GetHeader(header)
		if value == "" {
			continue
		}
		origin, err := url.Parse(value)
		if err != nil || origin.Host == "" || !strings.EqualFold(origin.Host, c.Request.Host) {
			return false
		}
	}
	return true
}
