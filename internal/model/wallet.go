package model

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// OperationType — тип операции (enum)
type OperationType string

const (
	OperationDeposit  OperationType = "DEPOSIT"
	OperationWithdraw OperationType = "WITHDRAW"
)

// WalletOperation — входящий запрос на изменение баланса
type WalletOperation struct {
	WalletID      uuid.UUID     `json:"walletId"`
	OperationType OperationType `json:"operationType"`
	Amount        int64         `json:"amount"`
}

// Custom JSON unmarshal для OperationType (валидация)
func (ot *OperationType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case string(OperationDeposit), string(OperationWithdraw):
		*ot = OperationType(s)
		return nil
	default:
		return fmt.Errorf("invalid operationType: %q, expected DEPOSIT or WITHDRAW", s)
	}
}

// Custom JSON marshal (опционально, для логов/ответов)
func (ot OperationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(ot))
}
