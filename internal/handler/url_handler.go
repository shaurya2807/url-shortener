package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shaurya2807/url-shortener/internal/model"
	"github.com/shaurya2807/url-shortener/internal/service"
	"go.uber.org/zap"
)

type URLHandler struct {
	svc *service.URLService
	log *zap.Logger
}

func NewURLHandler(svc *service.URLService, log *zap.Logger) *URLHandler {
	return &URLHandler{svc: svc, log: log}
}

func (h *URLHandler) Redirect(c *gin.Context) {
	code := c.Param("code")
	originalURL, err := h.svc.Redirect(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "short code not found"})
			return
		}
		h.log.Error("redirect failed", zap.String("short_code", code), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.Redirect(http.StatusMovedPermanently, originalURL)
}

func (h *URLHandler) Stats(c *gin.Context) {
	code := c.Param("code")
	resp, err := h.svc.GetStats(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "short code not found"})
			return
		}
		h.log.Error("get stats failed", zap.String("short_code", code), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *URLHandler) Shorten(c *gin.Context) {
	var req model.ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Shorten(c.Request.Context(), req.OriginalURL)
	if err != nil {
		h.log.Error("shorten failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
