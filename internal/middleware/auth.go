package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/jwtutil"
	"github.com/joakim/fintrack-api/pkg/response"
)

const (
	ContextUserID    = "userID"
	ContextSessionID = "sessionID"
)

// Auth validates the Bearer access token. If sessionExists is provided it is
// called with the session ID embedded in the token — returning false causes an
// immediate 401, enabling instant revocation when a session is deleted.
func Auth(accessSecret string, sessionExists ...func(id string) bool) gin.HandlerFunc {
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
		if len(sessionExists) > 0 && sessionExists[0] != nil {
			if !sessionExists[0](claims.SessionID) {
				response.Error(c, apperror.Unauthorized("session has been revoked"))
				c.Abort()
				return
			}
		}
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextSessionID, claims.SessionID)
		c.Next()
	}
}
