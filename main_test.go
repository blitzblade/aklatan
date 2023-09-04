package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDefaultRoute(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	ctx, router := gin.CreateTestContext(recorder)
	setupRouter(router)

	req, err := http.NewRequestWithContext(ctx, "GET", "/", nil)
	if err != nil {
		t.Errorf("got error: %s", err)
	}

	router.ServeHTTP(recorder, req)
	if http.StatusOK != recorder.Code {
		t.Fatalf("expected response code %d, got %d", http.StatusOK, recorder.Code)
	}

	body := recorder.Body.String()

	expected := "Hello, gin!"

	if expected != strings.Trim(body, " \r\n") {
		t.Fatalf("expected response body '%s', got '%s'", expected, body)
	}
}
