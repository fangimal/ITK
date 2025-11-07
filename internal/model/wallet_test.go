package model

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletOperation_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name: "valid DEPOSIT",
			input: `{
				"walletId": "123e4567-e89b-12d3-a456-426614174000",
				"operationType": "DEPOSIT",
				"amount": 1000
			}`,
			expectErr: false,
		},
		{
			name: "valid WITHDRAW",
			input: `{
				"walletId": "123e4567-e89b-12d3-a456-426614174000",
				"operationType": "WITHDRAW",
				"amount": 500
			}`,
			expectErr: false,
		},
		{
			name:      "invalid operationType",
			input:     `{"walletId":"123e4567-e89b-12d3-a456-426614174000","operationType":"TRANSFER","amount":100}`,
			expectErr: true,
		},
		{
			name:      "invalid UUID",
			input:     `{"walletId":"invalid","operationType":"DEPOSIT","amount":100}`,
			expectErr: true,
		},
		{
			name:      "negative amount — allowed in model",
			input:     `{"walletId":"123e4567-e89b-12d3-a456-426614174000","operationType":"DEPOSIT","amount":-100}`,
			expectErr: false, // ✅ модель не валидирует amount
		},
		{
			name:      "zero amount — allowed in model",
			input:     `{"walletId":"123e4567-e89b-12d3-a456-426614174000","operationType":"DEPOSIT","amount":0}`,
			expectErr: false,
		},
		{
			name:      "unknown field — ignored by json.Unmarshal",
			input:     `{"walletId":"123e4567-e89b-12d3-a456-426614174000","operationType":"DEPOSIT","amount":100,"extra":"field"}`,
			expectErr: false, // ✅ json.Unmarshal не падает от extra-полей
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var op WalletOperation
			err := json.Unmarshal([]byte(tt.input), &op)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Можно добавить минимальные проверки, если хочешь:
				if tt.name == "valid DEPOSIT" {
					assert.Equal(t, uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), op.WalletID)
					assert.Equal(t, OperationDeposit, op.OperationType)
					assert.Equal(t, int64(1000), op.Amount)
				}
			}
		})
	}
}

func TestOperationType_MarshalJSON(t *testing.T) {
	op := WalletOperation{
		WalletID:      uuid.New(),
		OperationType: OperationDeposit,
		Amount:        1000,
	}

	data, err := json.Marshal(op)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"operationType":"DEPOSIT"`)
}
