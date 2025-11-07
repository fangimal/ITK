package errors

import "errors"

var (
	WalletNotFound    = errors.New("wallet not found")
	InsufficientFunds = errors.New("insufficient funds")
	InvalidAmount     = errors.New("amount must be positive")
	InvalidOperation  = errors.New("invalid operation type")
)

// Is — для поддержки errors.Is()
func Is(target, err error) bool {
	return errors.Is(err, target)
}
