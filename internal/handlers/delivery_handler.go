package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"

	"sukumad/internal/repo"
)

type DeliveryHandler struct {
	repo *repo.DeliveryRepo
}

func NewDeliveryHandler(db *pgxpool.Pool) *DeliveryHandler {
	return &DeliveryHandler{repo: repo.NewDeliveryRepo(db)}
}

func (h *DeliveryHandler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(defaultIfEmpty(c.Query("limit"), "50"))
	offset, _ := strconv.Atoi(defaultIfEmpty(c.Query("offset"), "0"))
	status := c.Query("status") // optional filter
	out, err := h.repo.List(c, status, limit, offset)
	if err != nil {
		log.WithError(err).Error("deliveries list")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": out, "limit": limit, "offset": offset})
}

func (h *DeliveryHandler) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	item, err := h.repo.Get(c, int64(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *DeliveryHandler) Patch(c *gin.Context) {
	// Small operational updates: status, next_run_at, priority
	id, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Status    *string    `json:"status"`
		NextRunAt *time.Time `json:"next_run_at"`
		Priority  *int32     `json:"priority"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.Patch(c, int64(id), body.Status, body.NextRunAt, body.Priority); err != nil {
		log.WithError(err).Error("patch delivery")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "patch failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *DeliveryHandler) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.repo.Delete(c, int64(id)); err != nil {
		log.WithError(err).Error("delete delivery")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.Status(http.StatusNoContent)
}
