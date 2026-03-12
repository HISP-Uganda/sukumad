package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/internal/repo"
	"sukumad/internal/types"
)

type PermissionHandler struct{ repo *repo.PermissionRepo }

func NewPermissionHandler(db *pgxpool.Pool) *PermissionHandler {
	return &PermissionHandler{repo: repo.NewPermissionRepo(db)}
}

// List godoc
// @Summary List permissions
// @Tags    permissions
// @Success 200 {array} types.Permission
// @Router  /api/permissions [get]
func (h *PermissionHandler) List(c *gin.Context) {
	out, err := h.repo.List(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, out)
}

// Create godoc
// @Summary Create permission
// @Tags    permissions
// @Accept  json
// @Param   permission body types.Permission true "Permission payload"
// @Success 201 {object} types.Permission
// @Router  /api/permissions [post]
func (h *PermissionHandler) Create(c *gin.Context) {
	var in types.Permission
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
// @Summary Update permission
// @Tags    permissions
// @Accept  json
// @Param   id path int true "Permission ID"
// @Param   permission body types.Permission true "Permission payload"
// @Success 200 {object} types.Permission
// @Router  /api/permissions/{id} [put]
func (h *PermissionHandler) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var in types.Permission
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
// @Summary Delete permission
// @Tags    permissions
// @Param   id path int true "Permission ID"
// @Success 204
// @Router  /api/permissions/{id} [delete]
func (h *PermissionHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(c, int32(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}
