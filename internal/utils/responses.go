package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data, "success": true, "message": "success"})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data, "success": true, "message": "created"})
}

func Message(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, gin.H{"data": nil, "success": true, "message": msg})
}

func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"data": nil, "success": false, "message": msg})
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
