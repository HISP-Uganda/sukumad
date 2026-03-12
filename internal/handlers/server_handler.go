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

type ServerHandler struct {
	db   *pgxpool.Pool
	repo *repo.ServerRepo
}

func NewServerHandler(db *pgxpool.Pool) *ServerHandler {
	return &ServerHandler{db: db, repo: repo.NewServerRepo(db)}
}

// List godoc
// @Summary      List servers
// @Description  List servers with pagination
// @Tags         servers
// @Param        limit   query   int     false  "Limit" default(50)
// @Param        offset  query   int     false  "Offset" default(0)
// @Success      200     {object}  map[string]any
// @Failure      500     {object}  types.APIError
// @Router       /api/servers [get]
func (h *ServerHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(defaultIfEmpty(c.Query("limit"), "50"))
	offset, _ := strconv.Atoi(defaultIfEmpty(c.Query("offset"), "0"))
	items, err := h.repo.List(c, limit, offset)
	if err != nil {
		log.WithError(err).Error("servers list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "limit": limit, "offset": offset})
}

// Get godoc
// @Summary      Get server
// @Tags         servers
// @Param        id   path     int  true  "Server ID"
// @Success      200  {object} types.Server
// @Failure      404  {object} types.APIError
// @Router       /api/servers/{id} [get]
func (h *ServerHandler) Get(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	item, err := h.repo.Get(c, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

// Create godoc
// @Summary      Create server
// @Tags         servers
// @Accept       json
// @Produce      json
// @Param        server  body     types.Server  true  "Server payload"
// @Success      201     {object} types.Server
// @Failure      400     {object} types.APIError
// @Failure      500     {object} types.APIError
// @Router       /api/servers [post]
func (h *ServerHandler) Create(c *gin.Context) {
	var in types.Server
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.repo.Create(c, &in)
	if err != nil {
		log.WithError(err).Error("create server")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// Update godoc
// @Summary      Update server
// @Tags         servers
// @Accept       json
// @Param        id      path     int          true  "Server ID"
// @Param        server  body     types.Server true  "Server payload"
// @Success      200     {object} types.Server
// @Failure      400     {object} types.APIError
// @Failure      500     {object} types.APIError
// @Router       /api/servers/{id} [put]
func (h *ServerHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var in types.Server
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.ID = int32(id)
	updated, err := h.repo.Update(c, &in)
	if err != nil {
		log.WithError(err).Error("update server")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete godoc
// @Summary      Delete server
// @Tags         servers
// @Param        id   path  int  true  "Server ID"
// @Success      204  {string}  string  "No Content"
// @Failure      400  {object}  types.APIError
// @Failure      500  {object}  types.APIError
// @Router       /api/servers/{id} [delete]
func (h *ServerHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.repo.Delete(c, int32(id)); err != nil {
		log.WithError(err).Error("delete server")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

func defaultIfEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
