package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBodyLimitRejectsDeclaredOversizeBeforeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.Use(BodyLimit(8))
	router.POST("/body", func(c *gin.Context) {
		called = true
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("123456789"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if called {
		t.Fatal("handler ran for a request with an oversized Content-Length")
	}
}

func TestBodyLimitTruncatesUnknownLengthBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	successPath := false
	router := gin.New()
	router.Use(BodyLimit(8))
	router.POST("/body", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err == nil {
			successPath = true
			c.Status(http.StatusNoContent)
			return
		}
		var maxBytesError *http.MaxBytesError
		if !errors.As(err, &maxBytesError) {
			t.Fatalf("read error = %v, want *http.MaxBytesError", err)
		}
		c.Status(http.StatusRequestEntityTooLarge)
	})

	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("123456789"))
	request.ContentLength = -1
	request.TransferEncoding = []string{"chunked"}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusRequestEntityTooLarge)
	}
	if successPath {
		t.Fatal("unknown-length oversized request entered the handler success path")
	}
}

func TestBodyLimitAllowsBodyAtLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(BodyLimit(8))
	router.POST("/body", func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		c.String(http.StatusOK, string(body))
	})

	request := httptest.NewRequest(http.MethodPost, "/body", strings.NewReader("12345678"))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "12345678" {
		t.Fatalf("status/body = %d/%q, want 200/%q", response.Code, response.Body.String(), "12345678")
	}
}
