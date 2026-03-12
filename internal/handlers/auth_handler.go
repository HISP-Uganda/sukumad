package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"sukumad/config"
	"sukumad/internal/auth"
	"sukumad/internal/repo"
	"sukumad/internal/types"
)

type AuthHandler struct {
	cfg       config.Config
	ur        *repo.UserRepo
	rr        *repo.RefreshRepo
	resetRepo *repo.ResetRepo

	// simple in-memory limiter: key => recent attempts timestamps
	mu       sync.Mutex
	attempts map[string][]time.Time
}

func NewAuthHandler(db *pgxpool.Pool, cfg config.Config) *AuthHandler {
	return &AuthHandler{
		cfg: cfg, ur: repo.NewUserRepo(db), rr: repo.NewRefreshRepo(db), resetRepo: repo.NewResetRepo(db),
		attempts: make(map[string][]time.Time),
	}
}

func (h *AuthHandler) allowAttempt(key string, max int, window time.Duration) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	qs := h.attempts[key]
	// drop old
	i := 0
	for _, t := range qs {
		if now.Sub(t) <= window {
			qs[i] = t
			i++
		}
	}
	qs = qs[:i]
	if len(qs) >= max {
		h.attempts[key] = qs
		return false
	}
	h.attempts[key] = append(qs, now)
	return true
}

// Login godoc
// @Description Login to the system and obtain a JWT token for further requests.
// @Summary Login and obtain a JWT
// @Tags    auth
// @Accept  json
// @Param   payload body types.LoginRequest true "Login"
// @Success 200 {object} types.LoginResponse
// @Failure 401 {object} types.APIError
// @Router  /api/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var in types.LoginRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// rate limit: 5 attempts per 5 minutes per IP+username
	key := c.ClientIP() + "|" + in.Username
	if !h.allowAttempt(key, 5, 5*time.Minute) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts"})
		return
	}

	user, hash, err := h.ur.GetWithHashByUsername(c, in.Username)
	if err != nil || user == nil || (user.IsActive != nil && !*user.IsActive) {
		_ = h.ur.RecordLoginFailure(c, in.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if !auth.IsIPAllowed(c.ClientIP(), user.AllowedIPs, user.DeniedIPs) {
		_ = h.ur.RecordLoginFailure(c, in.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "ip not allowed"})
		return
	}

	if user.LockedUntil != nil && time.Now().Before(*user.LockedUntil) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "account locked", "until": user.LockedUntil})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		// _ = h.ur.RecordLoginFailure(c, in.Username)
		_ = h.ur.RecordLoginFailureWithLockout(
			c, in.Username, time.Now(), h.cfg.LockoutThreshold, h.cfg.LockoutWindow, h.cfg.LockoutDuration)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	perms, _ := h.ur.ListEffectivePermissions(c, user.ID)
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.Code)
	}

	// Access token
	access, err := auth.Sign(h.cfg.JWTSecret, h.cfg.JWTIssuer, h.cfg.JWTExpiry, user.ID, user.Username, user.UserRole, codes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}

	// Refresh token (longer TTL, e.g. 7 days)
	jti := auth.NewJTI()
	refresh, exp, err := auth.SignRefresh(h.cfg.JWTSecret, h.cfg.JWTIssuer, 24*7*time.Hour, user.ID, user.Username, jti)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login failed"})
		return
	}
	_ = h.ur.RecordLoginSuccess(c, user.ID)
	_ = h.ur.ClearFailuresAndUnlock(c, user.ID)
	_ = h.rr.Create(c, user.ID, jti, exp, c.GetHeader("User-Agent"), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"access": access, "refresh": refresh})
}

// Refresh godoc
// @Description Refresh the access token using a valid refresh token.
// @Summary Refresh tokens (rotate refresh)
// @Tags    auth
// @Param   refresh header string true "Refresh token in Authorization: Bearer <token>"
// @Success 200 {object} map[string]string "access, refresh"
// @Failure 401 {object} types.APIError
// @Router  /api/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	hdr := c.GetHeader("Authorization")
	if !strings.HasPrefix(strings.ToLower(hdr), "bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing refresh token"})
		return
	}
	rtok := strings.TrimSpace(hdr[len("Bearer "):])
	rc, err := auth.ParseRefresh(h.cfg.JWTSecret, rtok)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh"})
		return
	}

	active, err := h.rr.IsActive(c, rc.JTI)
	if err != nil || !active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "refresh not active"})
		return
	}

	// rotate: revoke old
	_ = h.rr.Revoke(c, rc.JTI)

	// get latest perms
	user, _, err := h.ur.GetWithHashByUsername(c, rc.Username)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	perms, _ := h.ur.ListEffectivePermissions(c, user.ID)
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.Code)
	}

	access, err := auth.Sign(h.cfg.JWTSecret, h.cfg.JWTIssuer, h.cfg.JWTExpiry, user.ID, user.Username, user.UserRole, codes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh failed"})
		return
	}
	jti := auth.NewJTI()
	refresh, exp, err := auth.SignRefresh(h.cfg.JWTSecret, h.cfg.JWTIssuer, 24*7*time.Hour, user.ID, user.Username, jti)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refresh failed"})
		return
	}
	_ = h.rr.Create(c, user.ID, jti, exp, c.GetHeader("User-Agent"), c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"access": access, "refresh": refresh})
}

// Logout godoc
// @Description Logout the user by revoking all their refresh tokens.
// @Summary Logout (revoke all refresh tokens for user)
// @Tags    auth
// @Success 200 {object} map[string]any
// @Router  /api/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	uidVal, ok := c.Get("auth.user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	uid := uidVal.(int32)
	if err := h.rr.RevokeAllForUser(c, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RequestPasswordReset godoc
// @Description Request a password reset by providing username or email.
// @Summary Request password reset
// @Tags    auth
// @Accept  json
// @Param   payload body types.PasswordResetRequest true "Username or Email"
// @Success 200 {object} map[string]any
// @Router  /api/auth/password/reset/request [post]
func (h *AuthHandler) RequestPasswordReset(c *gin.Context) {
	var in types.PasswordResetRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	uid, err := h.ur.FindUserIDByUsernameOrEmail(c, in.UsernameOrEmail)
	if err != nil {
		// Always 200 to avoid user enumeration
		c.JSON(200, gin.H{"ok": true})
		return
	}
	plain, hash, err := auth.NewResetToken()
	if err != nil {
		c.JSON(500, gin.H{"error": "could not create token"})
		return
	}
	exp := time.Now().Add(h.cfg.ResetTTL)
	_ = h.resetRepo.Create(c, uid, hash, exp, c.GetHeader("User-Agent"), c.ClientIP())

	// TODO: send email/SMS with 'plain'; for now, if Debug, include token in response for manual testing.
	resp := gin.H{"ok": true}
	if h.cfg.Debug {
		resp["debug_token"] = plain
	}
	c.JSON(200, resp)
}

// ConfirmPasswordReset godoc
// @Description Confirm password reset by providing token and new password.
// @Summary Confirm password reset
// @Tags    auth
// @Accept  json
// @Param   payload body types.PasswordResetConfirm true "Token + New Password"
// @Success 200 {object} map[string]any
// @Failure 400 {object} types.APIError
// @Router  /api/auth/password/reset/confirm [post]
func (h *AuthHandler) ConfirmPasswordReset(c *gin.Context) {
	var in types.PasswordResetConfirm
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	hash := auth.HashResetToken(in.Token)
	uid, err := h.resetRepo.Use(c, hash)
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid or expired token"})
		return
	}
	if err := h.ur.SetPassword(c, uid, in.NewPassword); err != nil {
		c.JSON(500, gin.H{"error": "update failed"})
		return
	}
	c.JSON(200, gin.H{"ok": true})
}
