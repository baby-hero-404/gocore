package stdgin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdres"
	"github.com/gin-gonic/gin"
)

func ctxFor(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func TestNoContent(t *testing.T) {
	c, w := ctxFor(http.MethodDelete, "/x")
	NoContent(c)
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
		t.Errorf("got %d with %d body bytes", w.Code, w.Body.Len())
	}
}

func TestJSON(t *testing.T) {
	t.Run("wraps plain data in the success envelope", func(t *testing.T) {
		c, w := ctxFor(http.MethodGet, "/x")
		JSON(c, http.StatusAccepted, map[string]int{"n": 1})
		var env stdres.Envelope
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || w.Code != http.StatusAccepted || env.Data == nil {
			t.Fatalf("status=%d env=%+v err=%v", w.Code, env, err)
		}
	})
	t.Run("an Envelope is sent as is", func(t *testing.T) {
		c, w := ctxFor(http.MethodGet, "/x")
		JSON(c, http.StatusTeapot, stdres.Envelope{Code: 7, Message: "custom"})
		var env stdres.Envelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if w.Code != http.StatusTeapot || env.Code != 7 || env.Message != "custom" {
			t.Fatalf("status=%d env=%+v", w.Code, env)
		}
	})
	t.Run("works without a request", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		JSON(c, http.StatusOK, "x")
		if w.Code != http.StatusOK {
			t.Errorf("status = %d", w.Code)
		}
	})
}

type upperTranslator struct{}

func (upperTranslator) Translate(context.Context, string, ...any) string { return "TRANSLATED" }

func TestError_NilAndTranslator(t *testing.T) {
	t.Run("nil error writes nothing", func(t *testing.T) {
		c, w := ctxFor(http.MethodGet, "/x")
		Error(c, nil)
		if w.Body.Len() != 0 {
			t.Errorf("unexpected body %q", w.Body.String())
		}
	})
	t.Run("a translator stored on the context is used", func(t *testing.T) {
		c, w := ctxFor(http.MethodGet, "/x")
		c.Set("translator", upperTranslator{})
		Error(c, stderr.NewNotFound("missing").WithKey("order.not_found"))
		var env stdres.Envelope
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		if w.Code != http.StatusNotFound || env.Message != "TRANSLATED" {
			t.Fatalf("status=%d env=%+v", w.Code, env)
		}
	})
	t.Run("a non translator value is ignored", func(t *testing.T) {
		c, w := ctxFor(http.MethodGet, "/x")
		c.Set("translator", "nope")
		Error(c, errors.New("plain failure"))
		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", w.Code)
		}
	})
}

func TestHandleValidationErrors_MalformedBody(t *testing.T) {
	c, w := ctxFor(http.MethodPost, "/x")
	HandleValidationErrors(c, struct{}{}, errors.New("unexpected EOF"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func bareContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w) // no c.Request, as in handler unit tests
	return c, w
}

func TestError_WithoutRequestDoesNotPanic(t *testing.T) {
	c, w := bareContext()
	Error(c, stderr.NewForbidden("nope"))
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
}

func TestHandleValidationErrors_WithoutRequestDoesNotPanic(t *testing.T) {
	c, w := bareContext()
	HandleValidationErrors(c, struct{}{}, errors.New("unexpected EOF"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}
