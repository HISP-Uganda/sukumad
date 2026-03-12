package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"

	"sukumad/internal/repo"
	"sukumad/internal/types"
)

type UserHandler struct{ repo *repo.UserRepo }

func NewUserHandler(db *pgxpool.Pool) *UserHandler { return &UserHandler{repo: repo.NewUserRepo(db)} }

// List godoc
// @Summary List users
// @Tags    users
// @Param   limit  query int false "Limit" default(50)
// @Param   offset query int false "Offset" default(0)
// @Success 200 {object} map[string]any
// @Router  /api/users [get]
func (h *UserHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(defaultIfEmpty(c.Query("limit"), "50"))
	offset, _ := strconv.Atoi(defaultIfEmpty(c.Query("offset"), "0"))
	items, err := h.repo.List(c, limit, offset)
	if err != nil {
		log.WithError(err).Error("users list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "limit": limit, "offset": offset})
}

// Get godoc
// @Summary Get user
// @Tags    users
// @Param   id path int true "User ID"
// @Success 200 {object} types.User
// @Router  /api/users/{id} [get]
func (h *UserHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	item, err := h.repo.Get(c, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Create godoc
// @Summary Create user
// @Tags    users
// @Accept  json
// @Param   user body types.User true "User payload (password required)"
// @Success 201 {object} types.User
// @Router  /api/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var in types.User
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.Create(c, &in)
	if err != nil {
		log.WithError(err).Error("create user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, out)
}

// Update godoc
// @Summary Update user
// @Tags    users
// @Accept  json
// @Param   id   path int       true "User ID"
// @Param   user body types.User true "User payload (password optional)"
// @Success 200  {object} types.User
// @Router  /api/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.User
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.ID = int32(id)
	out, err := h.repo.Update(c, &in)
	if err != nil {
		log.WithError(err).Error("update user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// Delete godoc
// @Summary Delete user
// @Tags    users
// @Param   id path int true "User ID"
// @Success 204
// @Router  /api/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(c, int32(id)); err != nil {
		log.WithError(err).Error("delete user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GrantPermission godoc
// @Summary Grant permission to user
// @Tags    users
// @Accept  json
// @Param   id path int true "User ID"
// @Param   payload body types.GrantUserPermission true "Permission grant"
// @Success 200 {object} map[string]any
// @Router  /api/users/{id}/permissions [post]
func (h *UserHandler) GrantPermission(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.GrantUserPermission
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.GrantUserPermission(c, int32(id), in.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "grant failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RevokePermission godoc
// @Summary Revoke permission from user
// @Tags    users
// @Accept  json
// @Param   id path int true "User ID"
// @Param   payload body types.GrantUserPermission true "Permission revoke"
// @Success 200 {object} map[string]any
// @Router  /api/users/{id}/permissions [delete]
func (h *UserHandler) RevokePermission(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.GrantUserPermission
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.RevokeUserPermission(c, int32(id), in.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// SetRole godoc
// @Summary Set user role
// @Tags    users
// @Accept  json
// @Param   id path int true "User ID"
// @Param   payload body types.SetUserRole true "Set role"
// @Success 200 {object} map[string]any
// @Router  /api/users/{id}/role [post]
func (h *UserHandler) SetRole(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.SetUserRole
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.SetUserRole(c, int32(id), in.RoleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "set role failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ListEffectivePermissions godoc
// @Summary List user's effective permissions
// @Tags    users
// @Param   id path int true "User ID"
// @Success 200 {array} types.Permission
// @Router  /api/users/{id}/permissions [get]
func (h *UserHandler) ListEffectivePermissions(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	out, err := h.repo.ListEffectivePermissions(c, int32(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// Unlock godoc
// @Summary  Admin: unlock a user account
// @Tags     users
// @Param    id   path   int  true  "User ID"
// @Success  200  {object} map[string]any
// @Failure  403  {object} types.APIError
// @Failure  500  {object} types.APIError
// @Router   /api/users/{id}/unlock [post]
func (h *UserHandler) Unlock(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.UnlockAccount(c, int32(id)); err != nil {
		log.WithError(err).Error("unlock user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "unlock failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
