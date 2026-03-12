package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/internal/repo"
	"sukumad/internal/types"
)

type RoleHandler struct{ repo *repo.RoleRepo }

func NewRoleHandler(db *pgxpool.Pool) *RoleHandler { return &RoleHandler{repo: repo.NewRoleRepo(db)} }

// List godoc
// @Summary List roles
// @Tags    roles
// @Success 200 {array} types.Role
// @Router  /api/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	out, err := h.repo.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// Create godoc
// @Summary Create role
// @Tags    roles
// @Accept  json
// @Param   role body types.Role true "Role payload"
// @Success 201 {object} types.Role
// @Router  /api/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var in types.Role
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.repo.Create(c, &in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, out)
}

// Update godoc
// @Summary Update role
// @Tags    roles
// @Accept  json
// @Param   id path int true "Role ID"
// @Param   role body types.Role true "Role payload"
// @Success 200 {object} types.Role
// @Router  /api/roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.Role
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.ID = int32(id)
	out, err := h.repo.Update(c, &in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// Delete godoc
// @Summary Delete role
// @Tags    roles
// @Param   id path int true "Role ID"
// @Success 204
// @Router  /api/roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(c, int32(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

// GrantPermission godoc
// @Summary Grant permission to role
// @Tags    roles
// @Accept  json
// @Param   id path int true "Role ID"
// @Param   payload body types.GrantRolePermission true "Permission grant"
// @Success 200 {object} map[string]any
// @Router  /api/roles/{id}/permissions [post]
func (h *RoleHandler) GrantPermission(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.GrantRolePermission
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.GrantRolePermission(c, int32(id), in.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "grant failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// RevokePermission godoc
// @Summary Revoke permission from role
// @Tags    roles
// @Accept  json
// @Param   id path int true "Role ID"
// @Param   payload body types.GrantRolePermission true "Permission revoke"
// @Success 200 {object} map[string]any
// @Router  /api/roles/{id}/permissions [delete]
func (h *RoleHandler) RevokePermission(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.GrantRolePermission
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.RevokeRolePermission(c, int32(id), in.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revoke failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
