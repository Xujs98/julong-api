package middleware

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

// ImageGenerationTaskImageAuth allows configured sources to read generated
// image files without a token. All non-matching requests retain owner-bound
// read-only token authentication.
func ImageGenerationTaskImageAuth() gin.HandlerFunc {
	tokenAuth := TokenAuthReadOnly()
	return func(c *gin.Context) {
		if common.ImageGenerationLogImageAuthWhitelistEnabled &&
			common.ImageGenerationLogImageAuthWhitelistMatches(
				common.ImageGenerationLogImageAuthWhitelist,
				c.ClientIP(),
				c.GetHeader("Origin"),
				c.GetHeader("Referer"),
			) {
			c.Set(string(constant.ContextKeyImageGenerationImageAuthBypassed), true)
			c.Next()
			return
		}
		tokenAuth(c)
	}
}
