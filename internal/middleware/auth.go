package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
	"github.com/joakim/fintrack-api/pkg/response"
)

const ContextUserID = "userID"

func Auth(accessSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, apperror.Unauthorized("missing bearer token"))
			c.Abort()
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtutil.ParseAccessToken(token, accessSecret)
		if err != nil {
			response.Error(c, apperror.Unauthorized("invalid or expired token"))
			c.Abort()
			return
		}
		c.Set(ContextUserID, claims.UserID)
		c.Next()
	}
}
