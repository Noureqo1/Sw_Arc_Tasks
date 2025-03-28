package auth

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slices"
	"net/http"
)

func RequireRoles(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get(CtxRolesKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "missing roles in token"})
			return
		}
		roles, ok := rolesVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid roles format"})
			return
		}

		// check if user has at least one of the required roles
		authorized := false
		for _, rr := range requiredRoles {
			if slices.Contains(roles, rr) {
				authorized = true
				break
			}
		}

		if !authorized {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden, insufficient roles"})
			return
		}

		c.Next()
	}
}
