package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo"
)

// Heartbeat endpoint middleware useful to setting up a path like
// `/ping` that load balancers or uptime testing external services
// can make a request before hitting any routes. It's also convenient
// to place this above ACL middlewares as well.
func Heartbeat(endpoint string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if r := c.Request(); r.Method == "GET" && strings.EqualFold(r.URL.Path, endpoint) {
				return c.String(http.StatusOK, ".")
			}
			return next(c)
		}
	}
}
