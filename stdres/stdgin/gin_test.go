package stdgin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/baby-hero-404/gocore/stdres"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOKAndCreated(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"id": "123", "name": "Item"}
	OK(c, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var res stdres.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Code != 0 || res.Message != "success" {
		t.Errorf("expected code 0 and message success, got code %d, message %s", res.Code, res.Message)
	}

	// Test Created
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	Created(c2, data)
	if w2.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w2.Code)
	}

	// Test NoContent
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	NoContent(c3)
	if w3.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w3.Code)
	}
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req

	Error(c, stderr.NewUnauthorized("token required"))

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var res stdres.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Code != int(stderr.CodeUnauthorized) {
		t.Errorf("expected code %d, got %d", stderr.CodeUnauthorized, res.Code)
	}
}

type sampleForm struct {
	Email string `json:"user_email" validate:"required,email"`
	Age   int    `json:"user_age" validate:"required,gte=18"`
}

func TestHandleValidationErrors(t *testing.T) {
	validate := validator.New()
	invalidObj := sampleForm{
		Email: "not-an-email",
		Age:   16,
	}
	valErr := validate.Struct(invalidObj)
	if valErr == nil {
		t.Fatal("expected validation error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/submit", bytes.NewBufferString("{}"))
	c.Request = req

	HandleValidationErrors(c, invalidObj, valErr)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status 422, got %d", w.Code)
	}

	var res stdres.Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Code != int(stderr.CodeValidationFailed) {
		t.Errorf("expected code %d, got %d", stderr.CodeValidationFailed, res.Code)
	}

	detailsRaw, err := json.Marshal(res.Details)
	if err != nil {
		t.Fatalf("failed to marshal details: %v", err)
	}
	var fieldErrors []stderr.FieldError
	if err := json.Unmarshal(detailsRaw, &fieldErrors); err != nil {
		t.Fatalf("failed to unmarshal details: %v", err)
	}

	if len(fieldErrors) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(fieldErrors))
	}

	// Verify json tag names were extracted ("user_email", "user_age")
	fields := map[string]bool{}
	for _, f := range fieldErrors {
		fields[f.Field] = true
	}
	if !fields["user_email"] {
		t.Errorf("expected field user_email in details, got %v", fields)
	}
	if !fields["user_age"] {
		t.Errorf("expected field user_age in details, got %v", fields)
	}
}
