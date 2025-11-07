package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fangimal/ITK/internal/config"
	"github.com/fangimal/ITK/internal/handlers"
	"github.com/fangimal/ITK/internal/repository"
)

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	cfg := &config.Config{
		DBHost:    "localhost",
		DBPort:    "5433", // как у тебя
		DBUser:    "wallet_user",
		DBPass:    "secure_password_123",
		DBName:    "wallet_db",
		DBSSLMode: "disable",
	}

	repo, err := repository.NewPostgresWalletRepository(cfg)
	require.NoError(t, err)

	// Очистка — через публичный метод
	ctx := context.Background()
	err = repo.TruncateTables(ctx)
	require.NoError(t, err)

	handler := handlers.NewWalletHandler(repo)

	router := httprouter.New()
	router.POST("/api/v1/wallets", handler.CreateWallet)
	router.POST("/api/v1/wallet", handler.Operation)
	router.GET("/api/v1/wallets/:uuid", handler.GetBalance)

	ts := httptest.NewServer(router)
	return ts, func() {
		ts.Close()
		repo.Close()
	}
}

func TestE2E_WalletFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e in -short mode")
	}

	ts, cleanup := setupTestServer(t)
	defer cleanup()

	t.Run("Full wallet lifecycle", func(t *testing.T) {
		// 1. Создать кошелёк
		resp := doRequest(t, ts, "POST", "/api/v1/wallets", nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var createResp struct {
			WalletID uuid.UUID `json:"walletId"`
		}
		err := json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		walletID := createResp.WalletID

		// 2. Пополнить
		op := map[string]interface{}{
			"walletId":      walletID.String(),
			"operationType": "DEPOSIT",
			"amount":        1000,
		}
		resp = doRequest(t, ts, "POST", "/api/v1/wallet", op)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 3. Проверить баланс
		resp = doRequest(t, ts, "GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), nil)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var balanceResp struct {
			WalletID uuid.UUID `json:"walletId"`
			Balance  int64     `json:"balance"`
		}
		err = json.NewDecoder(resp.Body).Decode(&balanceResp)
		require.NoError(t, err)
		assert.Equal(t, walletID, balanceResp.WalletID)
		assert.Equal(t, int64(1000), balanceResp.Balance)

		// 4. Списать
		op["operationType"] = "WITHDRAW"
		op["amount"] = 400
		resp = doRequest(t, ts, "POST", "/api/v1/wallet", op)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// 5. Проверить баланс снова
		resp = doRequest(t, ts, "GET", fmt.Sprintf("/api/v1/wallets/%s", walletID), nil)
		err = json.NewDecoder(resp.Body).Decode(&balanceResp)
		require.NoError(t, err)
		assert.Equal(t, int64(600), balanceResp.Balance)

		// 6. Попытка списать слишком много → 422
		op["amount"] = 1000
		resp = doRequest(t, ts, "POST", "/api/v1/wallet", op)
		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

		var errResp struct {
			Error string `json:"error"`
		}
		err = json.NewDecoder(resp.Body).Decode(&errResp)
		require.NoError(t, err)
		assert.Equal(t, "insufficient funds", errResp.Error)
	})
}

func doRequest(t *testing.T, ts *httptest.Server, method, path string, body interface{}) *http.Response {
	var bodyReader io.Reader // ← тип io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b) // или bytes.NewBuffer(b) — но лучше NewReader
	}

	req, err := http.NewRequest(method, ts.URL+path, bodyReader) // ← nil или *bytes.Reader

	require.NoError(t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}
