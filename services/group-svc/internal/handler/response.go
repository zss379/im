package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: 0, Msg: "success", Data: data})
}

func Error(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, Response{Code: code, Msg: msg})
}

func BadRequest(c *gin.Context, msg string) {
	Error(c, http.StatusBadRequest, 40001, msg)
}

func Unauthorized(c *gin.Context, msg string) {
	Error(c, http.StatusUnauthorized, 40002, msg)
}

func NotFound(c *gin.Context, msg string) {
	Error(c, http.StatusNotFound, 40004, msg)
}

func InternalError(c *gin.Context, msg string) {
	Error(c, http.StatusInternalServerError, 50001, msg)
}

func getTenantID(c *gin.Context) int64 {
	tid, _ := c.Get("tenant_id")
	if tid, ok := tid.(int64); ok {
		return tid
	}
	return 0
}

func getUserID(c *gin.Context) int64 {
	uid, _ := c.Get("user_id")
	if uid, ok := uid.(int64); ok {
		return uid
	}
	return 0
}
