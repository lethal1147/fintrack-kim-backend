package middleware

import "github.com/gin-gonic/gin"

// RateLimiter is a stub. Replace with a Redis-backed token bucket (e.g. go-redis/redis_rate)
// when rate limiting is required.
func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
