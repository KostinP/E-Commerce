package middleware

import (
	"github.com/gin-gonic/gin"
)

func DebugCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		println("🔥 CORS Debug - Request Origin:", origin)
		println("🔥 CORS Debug - Request Method:", c.Request.Method)
		println("🔥 CORS Debug - Request Path:", c.Request.URL.Path)

		// Разрешаем все origins для теста
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			println("🔥 CORS Debug - OPTIONS request, aborting with 204")
			c.AbortWithStatus(204)
			return
		}

		c.Next()

		println("🔥 CORS Debug - Response Status:", c.Writer.Status())
		println("🔥 CORS Debug - Response Headers:", c.Writer.Header().Get("Access-Control-Allow-Origin"))
	}
}
