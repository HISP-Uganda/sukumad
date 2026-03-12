package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID   int32    `json:"uid"`
	Username string   `json:"uname"`
	RoleID   *int32   `json:"role_id,omitempty"`
	Perms    []string `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID   int32  `json:"uid"`
	Username string `json:"uname"`
	JTI      string `json:"jti"`
	jwt.RegisteredClaims
}

func Sign(secret, issuer string, ttl time.Duration, userID int32, username string, roleID *int32, perms []string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RoleID:   roleID,
		Perms:    perms,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

func Parse(secret string, token string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tok.Claims.(*Claims); ok && tok.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

// IP allow/deny helpers
func IsIPAllowed(clientIP string, allowed, denied []string) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}
	// explicit deny takes precedence
	for _, d := range denied {
		if inCidr(ip, d) {
			return false
		}
	}
	// if allowed list provided, must match one
	if len(allowed) > 0 {
		for _, a := range allowed {
			if inCidr(ip, a) {
				return true
			}
		}
		return false
	}
	// no allowed list => allowed unless denied
	return true
}

func inCidr(ip net.IP, cidr string) bool {
	cidr = strings.TrimSpace(cidr)
	if cidr == "" {
		return false
	}
	_, block, err := net.ParseCIDR(cidr)
	if err != nil {
		// try plain IP
		return ip.Equal(net.ParseIP(cidr))
	}
	return block.Contains(ip)
}

func NewJTI() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func SignRefresh(secret, issuer string, ttl time.Duration, userID int32, username string, jti string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)
	claims := RefreshClaims{
		UserID:   userID,
		Username: username,
		JTI:      jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString([]byte(secret))
	return s, exp, err
}

func ParseRefresh(secret, token string) (*RefreshClaims, error) {
	tok, err := jwt.ParseWithClaims(token, &RefreshClaims{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tok.Claims.(*RefreshClaims); ok && tok.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
