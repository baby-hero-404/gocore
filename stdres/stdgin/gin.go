// Package stdgin adapts the stdres response envelope to Gin.
package stdgin

import (
	"context"
	"net/http"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdlog"
	"github.com/baby-hero-404/gocore/stdres"
	"github.com/gin-gonic/gin"
)

// requestContext returns the request's context, or Background for a bare gin context (tests,
// handlers invoked without a request).
func requestContext(c *gin.Context) context.Context {
	if c == nil || c.Request == nil {
		return context.Background()
	}
	return c.Request.Context()
}

// OK responds with HTTP 200 and standard success envelope.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, stdres.SuccessCtx(requestContext(c), data))
}

// Created responds with HTTP 201 and created envelope.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, stdres.Envelope{
		Code:      int(stderr.CodeOK),
		Message:   "created",
		Data:      data,
		RequestID: stdres.SuccessCtx(requestContext(c), nil).RequestID,
	})
}

// NoContent responds with HTTP 204.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
	c.Writer.WriteHeaderNow()
}

// JSON sends data with custom HTTP status code wrapped in success envelope if not already an Envelope.
func JSON(c *gin.Context, status int, data any) {
	if env, ok := data.(stdres.Envelope); ok {
		c.JSON(status, env)
		return
	}
	env := stdres.SuccessCtx(requestContext(c), data)
	c.JSON(status, env)
}

// Error responds with a standardized error envelope and aborts the Gin context.
func Error(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var ctxTranslator stdres.Translator
	if val, exists := c.Get("translator"); exists {
		if t, ok := val.(stdres.Translator); ok {
			ctxTranslator = t
		}
	}

	env, status := stdres.FromError(c.Request.Context(), err, ctxTranslator)
	c.AbortWithStatusJSON(status, env)
}

// HandleValidationErrors inspects request binding errors and responds with formatted validation details.
func HandleValidationErrors(c *gin.Context, target any, err error) {
	stdErr := stdres.ValidationError(target, err)
	if stdErr.Code == stderr.CodeBadRequest && len(stdErr.Details) == 1 && stdErr.Details[0].Field == "body" {
		stdlog.WithContext(c.Request.Context()).Warn("binding error", "error", err.Error())
	}
	Error(c, stdErr)
}
