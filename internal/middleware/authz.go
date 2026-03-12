package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"sukumad/config"
	"sukumad/internal/auth"
)

const (
	ctxUserID   = "auth.user_id"
	ctxUsername = "auth.username"
	ctxPerms    = "auth.perms"
	ctxRoleID   = "auth.role_id"
)

func Authn(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		tok := strings.TrimSpace(h[len("Bearer "):])
		claims, err := auth.Parse(cfg.JWTSecret, tok)
		if err != nil {
			log.WithError(err).Warn("jwt parse")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		// attach to context
		c.Set(ctxUserID, claims.UserID)
		c.Set(ctxUsername, claims.Username)
		c.Set(ctxPerms, claims.Perms)
		if claims.RoleID != nil {
			c.Set(ctxRoleID, *claims.RoleID)
		}
		c.Next()
	}
}

// RequirePerm blocks if the user lacks the given permission code.
func RequirePerm(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, ok := c.Get(ctxPerms)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no permissions"})
			return
		}
		perms, _ := val.([]string)
		for _, p := range perms {
			if p == code {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	}
}

// RequireAny ("p1","p2",...)
func RequireAny(codes ...string) gin.HandlerFunc {
	set := map[string]struct{}{}
	for _, c := range codes {
		set[c] = struct{}{}
	}
	return func(c *gin.Context) {
		val, ok := c.Get(ctxPerms)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no permissions"})
			return
		}
		perms, _ := val.([]string)
		for _, p := range perms {
			if _, ok := set[p]; ok {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
	}
}

// RequireAll ("p1","p2",...)
func RequireAll(codes ...string) gin.HandlerFunc {
	want := map[string]struct{}{}
	for _, c := range codes {
		want[c] = struct{}{}
	}
	return func(c *gin.Context) {
		val, ok := c.Get(ctxPerms)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "no permissions"})
			return
		}
		perms, _ := val.([]string)
		got := map[string]struct{}{}
		for _, p := range perms {
			got[p] = struct{}{}
		}
		for w := range want {
			if _, ok := got[w]; !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission denied"})
				return
			}
		}
		c.Next()
	}
}
