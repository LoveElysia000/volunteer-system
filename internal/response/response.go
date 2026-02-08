package response

import (
	"volunteer-system/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func Success(c *app.RequestContext, data interface{}) {
	c.JSON(consts.StatusOK, Response{
		Code:    consts.StatusOK,
		Message: consts.StatusMessage(consts.StatusOK),
		Data:    data,
	})
}

func Fail(c *app.RequestContext, err error) {
	FailWithCode(c, consts.StatusBadRequest, err)
}

func FailWithCode(c *app.RequestContext, code int, err error) {
	msg := consts.StatusMessage(code)
	if err != nil {
		logger.GetLogger().Info("[failWithCode] error is %+v", err)
		msg = err.Error()
	}
	c.JSON(consts.StatusOK, Response{
		Code:    code,
		Message: msg,
		Data:    struct{}{},
	})
}

func FailWithError(c *app.RequestContext, errors *Errors) {
	if errors != nil {
		logger.GetLogger().Info("[failWithError] error is %+v", errors)
	}
	err := GetErrorFromCode(errors.Code())
	c.JSON(consts.StatusOK, Response{
		Code:    errors.code,
		Message: err.Error(),
		Data:    struct{}{},
	})
}

func Unauthorized(c *app.RequestContext, err error) {
	FailWithCode(c, consts.StatusUnauthorized, err)
}

// SuccessWithMessage 带自定义消息的成功响应
func SuccessWithMessage(c *app.RequestContext, message string, data interface{}) {
	c.JSON(consts.StatusOK, Response{
		Code:    consts.StatusOK,
		Message: message,
		Data:    data,
	})
}

// Created 创建成功响应
func Created(c *app.RequestContext, data interface{}) {
	c.JSON(consts.StatusCreated, Response{
		Code:    consts.StatusCreated,
		Message: consts.StatusMessage(consts.StatusCreated),
		Data:    data,
	})
}
