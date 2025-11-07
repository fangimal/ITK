package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/fangimal/ITK/internal/config"
	"github.com/fangimal/ITK/internal/errors"
)

type WalletRepository interface {
	CreateWallet(ctx context.Context) (uuid.UUID, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error)
	UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64, isDeposit bool) error
}

type PostgresWalletRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWalletRepository(cfg *config.Config) (*PostgresWalletRepository, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName, cfg.DBSSLMode,
	)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Проверяем подключение
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Подключение к PostgreSQL установлено")
	return &PostgresWalletRepository{pool: pool}, nil
}

func (r *PostgresWalletRepository) Close() {
	r.pool.Close()
}

// CreateWallet создаёт новый кошелёк и возвращает его ID
func (r *PostgresWalletRepository) CreateWallet(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO wallets (balance) 
		VALUES (0) 
		RETURNING id
	`).Scan(&id)
	return id, err
}

// GetBalance возвращает текущий баланс
func (r *PostgresWalletRepository) GetBalance(ctx context.Context, walletID uuid.UUID) (int64, error) {
	var balance int64
	err := r.pool.QueryRow(ctx, `
		SELECT balance 
		FROM wallets 
		WHERE id = $1
	`, walletID).Scan(&balance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, fmt.Errorf("%w: %s", errors.WalletNotFound, walletID)
		}
		return 0, err
	}
	return balance, nil
}

// UpdateBalance — атомарное обновление баланса
// isDeposit = true → +amount, false → -amount (с проверкой на отрицательный баланс!)
func (r *PostgresWalletRepository) UpdateBalance(ctx context.Context, walletID uuid.UUID, amount int64, isDeposit bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // откат при ошибке

	// 🔒 Блокируем строку кошелька на время транзакции
	var currentBalance int64

	sqlQuery := `SELECT balance 
		FROM wallets 
		WHERE id = $1 
		FOR UPDATE`

	err = tx.QueryRow(ctx, sqlQuery, walletID).Scan(&currentBalance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("%w: %s", errors.WalletNotFound, walletID)
		}
		return fmt.Errorf("select for update: %w", err)
	}

	// Проверяем, не уйдёт ли баланс в минус при WITHDRAW
	if !isDeposit && currentBalance < amount {
		return fmt.Errorf("%w: balance %d, withdraw %d", errors.InsufficientFunds, currentBalance, amount)
	}

	// Обновляем баланс
	newBalance := currentBalance
	if isDeposit {
		newBalance += amount
	} else {
		newBalance -= amount
	}

	sqlQuery = `
	UPDATE wallets
	SET balance = $1, updated_at = NOW()
	WHERE id = $2
	`
	_, err = tx.Exec(ctx, sqlQuery, newBalance, walletID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	// Логируем операцию в transactions
	opType := "DEPOSIT"
	if !isDeposit {
		opType = "WITHDRAW"
	}

	sqlQuery = `
		INSERT INTO transactions (wallet_id, operation_type, amount)
		VALUES ($1, $2, $3)`

	_, err = tx.Exec(ctx, sqlQuery, walletID, opType, amount)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	return tx.Commit(ctx)
}
