package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fangimal/ITK/internal/model"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// WalletHandler — структура для хендлеров (пока без зависимостей, позже — с репозиторием)
type WalletHandler struct{}

// NewWalletHandler — конструктор
func NewWalletHandler() *WalletHandler {
	return &WalletHandler{}
}

// === Обработчики ===

func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	id := uuid.New() // v4 UUID
	fmt.Fprintf(w, `{"walletId":"%s"}`, id)
}

// walletHandler — POST /api/v1/wallet
func (h *WalletHandler) Operation(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var op model.WalletOperation

	// ⚠️ Всегда ограничиваем размер тела!
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // защита от опечаток в полях

	if err := decoder.Decode(&op); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	// Валидация amount > 0
	if op.Amount <= 0 {
		http.Error(w, `{"error":"amount must be positive integer"}`, http.StatusBadRequest)
		return
	}

	// Готово — логика будет тут позже
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"walletId": "%s",
		"operationType": "%s",
		"amount": %d,
		"status": "accepted"
	}`, op.WalletID, op.OperationType, op.Amount)
}

// walletsHandler — GET /api/v1/wallets/:uuid
func (h *WalletHandler) GetBalance(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uuidStr := ps.ByName("uuid")
	if _, err := uuid.Parse(uuidStr); err != nil {
		http.Error(w, `{"error":"invalid UUID"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `[
		{"id":"t1","walletId":"%s","type":"DEPOSIT","amount":1000,"timestamp":"2025-11-06T12:00:00Z"},
		{"id":"t2","walletId":"%s","type":"WITHDRAW","amount":300,"timestamp":"2025-11-06T12:05:00Z"}
	]`, uuidStr, uuidStr)
}

func (h *WalletHandler) GetTransactions(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	uuidStr := ps.ByName("uuid")
	if _, err := uuid.Parse(uuidStr); err != nil {
		http.Error(w, `{"error":"invalid UUID"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `[
		{"id":"t1","walletId":"%s","type":"DEPOSIT","amount":1000,"timestamp":"2025-11-06T12:00:00Z"},
		{"id":"t2","walletId":"%s","type":"WITHDRAW","amount":300,"timestamp":"2025-11-06T12:05:00Z"}
	]`, uuidStr, uuidStr)
}
