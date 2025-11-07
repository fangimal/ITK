package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	myerrors "github.com/fangimal/ITK/internal/errors"
	"github.com/fangimal/ITK/internal/model"
	"github.com/fangimal/ITK/internal/repository"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

type WalletHandler struct {
	repo repository.WalletRepository
}

func NewWalletHandler(repo repository.WalletRepository) *WalletHandler {

	return &WalletHandler{repo: repo}
}

// === Обработчики ===

func (h *WalletHandler) CreateWallet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	id, err := h.repo.CreateWallet(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to create wallet"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
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

	isDeposit := op.OperationType == model.OperationDeposit

	err := h.repo.UpdateBalance(r.Context(), op.WalletID, op.Amount, isDeposit)
	if err != nil {
		if errors.Is(err, myerrors.WalletNotFound) {
			http.Error(w, `{"error":"wallet not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, myerrors.InsufficientFunds) {
			http.Error(w, `{"error":"insufficient funds"}`, http.StatusUnprocessableEntity) // 422
			return
		}
		log.Printf("DB error: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
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
	walletID, err := uuid.Parse(uuidStr)
	if err != nil {
		http.Error(w, `{"error":"invalid UUID"}`, http.StatusBadRequest)
		return
	}

	balance, err := h.repo.GetBalance(r.Context(), walletID)
	if err != nil {
		if err.Error() == fmt.Sprintf("wallet not found: %s", walletID) {
			http.Error(w, `{"error":"wallet not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"walletId": "%s",
		"balance": %d
	}`, walletID, balance)
}

func (h *WalletHandler) GetTransactions(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `[]`)
}
