package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, data)
}

func Message(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"message": msg})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

func BadRequest(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, msg)
}

func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, msg)
}

func Conflict(c *gin.Context, msg string) {
	Error(c, http.StatusConflict, msg)
}

func Gone(c *gin.Context, msg string) {
	Error(c, http.StatusGone, msg)
}

func InternalError(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, msg)
}
