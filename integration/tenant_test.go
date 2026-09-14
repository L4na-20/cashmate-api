package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"cashmate-api/config"
	"cashmate-api/models"
	"cashmate-api/routes"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestTenantRBACAndBalanceFlow(t *testing.T) {
	dsn := os.Getenv("CASHMATE_TEST_DSN")
	if dsn == "" {
		t.Skip("CASHMATE_TEST_DSN is not configured")
	}
	if !strings.Contains(strings.ToLower(dsn), "_test") {
		t.Fatal("CASHMATE_TEST_DSN must point to a database containing _test")
	}
	os.Setenv("JWT_SECRET", "integration-access-secret-32-characters-long")
	os.Setenv("JWT_REFRESH_SECRET", "integration-refresh-secret-32-characters-long")
	os.Setenv("APP_TIMEZONE", "Asia/Jakarta")
	config.LoadConfig()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()
	for _, model := range []any{&models.Transaction{}, &models.Category{}, &models.Wallet{}, &models.User{}, &models.Business{}} {
		if err := db.Migrator().DropTable(model); err != nil {
			t.Fatalf("reset test database: %v", err)
		}
	}
	if err := db.AutoMigrate(&models.Business{}, &models.User{}, &models.Wallet{}, &models.Category{}, &models.Transaction{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	config.DB = db

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	routes.Setup(engine)
	suffix := time.Now().UnixNano()

	ownerA := postJSON(t, engine, "/api/auth/register", map[string]any{
		"business_name": "Business A", "name": "Owner A", "email": fmt.Sprintf("owner-a-%d@example.test", suffix), "password": "password123",
	})
	ownerAUser := ownerA["user"].(map[string]any)
	ownerAToken := loginToken(t, engine, ownerAUser["email"].(string), "password123")
	ownerB := postJSON(t, engine, "/api/auth/register", map[string]any{
		"business_name": "Business B", "name": "Owner B", "email": fmt.Sprintf("owner-b-%d@example.test", suffix), "password": "password123",
	})
	ownerBUser := ownerB["user"].(map[string]any)
	ownerBToken := loginToken(t, engine, ownerBUser["email"].(string), "password123")

	staff := postJSONAuth(t, engine, "/api/staff", ownerAToken, map[string]any{
		"name": "Staff A", "email": fmt.Sprintf("staff-a-%d@example.test", suffix), "password": "password123",
	})
	staffToken := loginToken(t, engine, staff["email"].(string), "password123")
	wallet := postJSONAuth(t, engine, "/api/wallets", ownerAToken, map[string]any{"name": "Cash A"})
	category := postJSONAuth(t, engine, "/api/categories", ownerAToken, map[string]any{"name": "Expense A", "type": "expense"})
	bWallet := getJSONAuth(t, engine, "/api/wallets", ownerBToken)["data"].([]any)
	bWalletID := int(bWallet[0].(map[string]any)["id"].(float64))

	if status := requestStatus(engine, http.MethodGet, fmt.Sprintf("/api/wallets/%d", bWalletID), ownerAToken, nil); status != http.StatusNotFound {
		t.Fatalf("cross-tenant wallet status = %d, want 404", status)
	}
	if status := requestStatus(engine, http.MethodGet, "/api/dashboard/summary", staffToken, nil); status != http.StatusForbidden {
		t.Fatalf("staff dashboard status = %d, want 403", status)
	}

	walletID := int(wallet["id"].(float64))
	categoryID := int(category["id"].(float64))
	transaction := postJSONAuth(t, engine, "/api/transactions", staffToken, map[string]any{
		"wallet_id": walletID, "category_id": categoryID, "amount": 25000, "type": "expense",
	})
	if got := int(getJSONAuth(t, engine, "/api/dashboard/summary", ownerAToken)["data"].(map[string]any)["total_balance"].(float64)); got != -25000 {
		t.Fatalf("balance after expense = %d, want -25000", got)
	}
	history := getJSONAuth(t, engine, "/api/transactions?from_date=2020-01-01", staffToken)
	if total := int(history["meta"].(map[string]any)["total"].(float64)); total != 1 {
		t.Fatalf("staff history total = %d, want 1", total)
	}
	if status := requestStatus(engine, http.MethodPut, fmt.Sprintf("/api/transactions/%d", int(transaction["id"].(float64))), staffToken, map[string]any{
		"wallet_id": walletID, "category_id": categoryID, "amount": 100, "type": "expense",
	}); status != http.StatusForbidden {
		t.Fatalf("staff edit status = %d, want 403", status)
	}
	if status := requestStatus(engine, http.MethodPost, "/api/transactions", staffToken, map[string]any{
		"wallet_id": walletID, "category_id": categoryID, "amount": 100, "type": "expense", "date": "2020-01-01",
	}); status != http.StatusForbidden {
		t.Fatalf("staff backdate status = %d, want 403", status)
	}
	transactionID := int(transaction["id"].(float64))
	status, _ := request(t, engine, http.MethodPut, fmt.Sprintf("/api/transactions/%d", transactionID), ownerAToken, map[string]any{
		"wallet_id": walletID, "category_id": categoryID, "amount": 10000, "type": "expense",
	})
	if status != http.StatusOK {
		t.Fatalf("owner edit status = %d, want 200", status)
	}
	if got := int(getJSONAuth(t, engine, "/api/dashboard/summary", ownerAToken)["data"].(map[string]any)["total_balance"].(float64)); got != -10000 {
		t.Fatalf("balance after edit = %d, want -10000", got)
	}
	if status := requestStatus(engine, http.MethodDelete, fmt.Sprintf("/api/transactions/%d", transactionID), ownerAToken, nil); status != http.StatusOK {
		t.Fatalf("owner void status = %d, want 200", status)
	}
	if got := int(getJSONAuth(t, engine, "/api/dashboard/summary", ownerAToken)["data"].(map[string]any)["total_balance"].(float64)); got != 0 {
		t.Fatalf("balance after void = %d, want 0", got)
	}
	if status := requestStatus(engine, http.MethodPost, "/api/auth/logout", ownerAToken, nil); status != http.StatusOK {
		t.Fatalf("owner logout status = %d, want 200", status)
	}
	if status := requestStatus(engine, http.MethodGet, "/api/dashboard/summary", ownerAToken, nil); status != http.StatusUnauthorized {
		t.Fatalf("token after logout status = %d, want 401", status)
	}
}

func loginToken(t *testing.T, engine http.Handler, email, password string) string {
	data := postJSON(t, engine, "/api/auth/login", map[string]any{"email": email, "password": password})
	return data["access_token"].(string)
}

func postJSONAuth(t *testing.T, engine http.Handler, path, token string, payload map[string]any) map[string]any {
	status, body := request(t, engine, http.MethodPost, path, token, payload)
	if status < 200 || status >= 300 {
		t.Fatalf("POST %s returned %d: %s", path, status, body)
	}
	return body["data"].(map[string]any)
}

func postJSON(t *testing.T, engine http.Handler, path string, payload map[string]any) map[string]any {
	status, body := request(t, engine, http.MethodPost, path, "", payload)
	if status < 200 || status >= 300 {
		t.Fatalf("POST %s returned %d: %#v", path, status, body)
	}
	return body["data"].(map[string]any)
}

func getJSONAuth(t *testing.T, engine http.Handler, path, token string) map[string]any {
	status, body := request(t, engine, http.MethodGet, path, token, nil)
	if status < 200 || status >= 300 {
		t.Fatalf("GET %s returned %d: %#v", path, status, body)
	}
	return body
}

func requestStatus(engine http.Handler, method, path, token string, payload map[string]any) int {
	status, _ := requestWithBody(engine, method, path, token, payload)
	return status
}

func request(t *testing.T, engine http.Handler, method, path, token string, payload map[string]any) (int, map[string]any) {
	status, body := requestWithBody(engine, method, path, token, payload)
	if body == nil {
		t.Fatalf("%s %s returned invalid JSON", method, path)
	}
	return status, body
}

func requestWithBody(engine http.Handler, method, path, token string, payload map[string]any) (int, map[string]any) {
	var body bytes.Buffer
	if payload != nil {
		_ = json.NewEncoder(&body).Encode(payload)
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res := httptest.NewRecorder()
	engine.ServeHTTP(res, req)
	var decoded map[string]any
	_ = json.Unmarshal(res.Body.Bytes(), &decoded)
	return res.Code, decoded
}
