package middlewares

import "github.com/gin-gonic/gin"

// CORS middleware adds Cross-Origin Resource Sharing headers.
//
// The allowed origins come from SERVER_CORS_ALLOWED_ORIGINS. When the list is
// empty every origin is allowed via the wildcard "*" — acceptable only while
// auth is a Bearer token (browsers reject Allow-Credentials together with a
// wildcard origin, and cookies are never used here, so credentials are never
// needed). When the list is set, the request Origin is echoed back only if it
// matches the allowlist exactly, and "Vary: Origin" is emitted so caches never
// serve one origin's Allow-Origin header to another.
func (m *Middleware) CORS() gin.HandlerFunc {
	// Build the lookup set once at construction, not per request.
	var allowed map[string]struct{}
	if m.cfg != nil && len(m.cfg.Server.CORSAllowedOrigins) > 0 {
		allowed = make(map[string]struct{}, len(m.cfg.Server.CORSAllowedOrigins))
		for _, origin := range m.cfg.Server.CORSAllowedOrigins {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		if allowed == nil {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			// The response now depends on the Origin request header, so caches
			// must key on it — even for disallowed origins, where the header is
			// simply absent.
			c.Writer.Header().Add("Vary", "Origin")
			if origin := c.GetHeader("Origin"); origin != "" {
				if _, ok := allowed[origin]; ok {
					c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				}
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
