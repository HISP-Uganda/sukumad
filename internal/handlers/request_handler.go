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

type RequestHandler struct {
	repo *repo.RequestRepo
}

func NewRequestHandler(db *pgxpool.Pool) *RequestHandler {
	return &RequestHandler{repo: repo.NewRequestRepo(db)}
}

func (h *RequestHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(defaultIfEmpty(c.Query("limit"), "50"))
	offset, _ := strconv.Atoi(defaultIfEmpty(c.Query("offset"), "0"))
	items, err := h.repo.List(c, limit, offset)
	if err != nil {
		log.WithError(err).Error("requests list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "limit": limit, "offset": offset})
}

func (h *RequestHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	item, err := h.repo.Get(c, int64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *RequestHandler) Create(c *gin.Context) {
	var in types.Request
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	created, err := h.repo.Create(c, &in)
	if err != nil {
		log.WithError(err).Error("create request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *RequestHandler) Update(c *gin.Context) {
	var in types.Request
	id, _ := strconv.Atoi(c.Param("id"))
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in.ID = int64(id)
	updated, err := h.repo.Update(c, &in)
	if err != nil {
		log.WithError(err).Error("update request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *RequestHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(c, int64(id)); err != nil {
		log.WithError(err).Error("delete request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *RequestHandler) Expand(c *gin.Context) {
	// Accept optional priorities; default 100 / 50
	id, _ := strconv.Atoi(c.Param("id"))
	dest, _ := strconv.Atoi(c.Query("destination"))
	cc := c.QueryArray("cc") // e.g. ?cc=3&cc=5

	var ccIDs []int
	for _, s := range cc {
		if v, err := strconv.Atoi(s); err == nil {
			ccIDs = append(ccIDs, v)
		}
	}
	if err := h.repo.Expand(c, int64(id), int32(dest), ccIDs); err != nil {
		log.WithError(err).Error("expand request")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "expand failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
