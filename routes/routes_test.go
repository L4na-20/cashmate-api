package routes_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"cashmate-api/routes"
	"github.com/gin-gonic/gin"
)

func TestProtectedEndpointReturnsNormalizedUnauthorizedEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.Setup(r)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/summary", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.Code)
	}
	if body := res.Body.String(); body != `{"message":"token tidak ada"}` {
		t.Fatalf("expected normalized error envelope, got %s", body)
	}
}
