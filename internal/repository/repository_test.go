package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/fangimal/ITK/internal/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPostgresWalletRepository(t *testing.T) {
	ctx := context.Background()

	// Запускаем PostgreSQL через GenericContainer
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_pass",
			"POSTGRES_DB":       "test_wallet_db",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer container.Terminate(ctx)

	// Получаем host и port для подключения
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connStr := fmt.Sprintf(
		"host=%s port=%s user=test_user password=test_pass dbname=test_wallet_db sslmode=disable",
		host, port.Port(),
	)

	// Подключаемся
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	// Инициализация БД (минимальная)
	sqlQuery := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE TABLE IF NOT EXISTS wallets (
		id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		balance    BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS transactions (
		id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		wallet_id      UUID NOT NULL REFERENCES wallets(id),
		operation_type TEXT NOT NULL,
		amount         BIGINT NOT NULL,
		created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

	_, err = pool.Exec(ctx, sqlQuery)
	require.NoError(t, err)

	repo := &PostgresWalletRepository{pool: pool}

	t.Run("CreateWallet", func(t *testing.T) {
		id, err := repo.CreateWallet(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, id)
	})

	t.Run("GetBalance and UpdateBalance", func(t *testing.T) {
		id, err := repo.CreateWallet(ctx)
		require.NoError(t, err)

		balance, err := repo.GetBalance(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, int64(0), balance)

		// DEPOSIT
		err = repo.UpdateBalance(ctx, id, 1000, true)
		require.NoError(t, err)

		balance, err = repo.GetBalance(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), balance)

		// WITHDRAW
		err = repo.UpdateBalance(ctx, id, 300, false)
		require.NoError(t, err)

		balance, err = repo.GetBalance(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, int64(700), balance)

		// Insufficient funds
		err = repo.UpdateBalance(ctx, id, 1000, false)
		assert.Error(t, err)
		assert.ErrorIs(t, err, errors.InsufficientFunds)
	})

	t.Run("WalletNotFound", func(t *testing.T) {
		fakeID := uuid.New()
		_, err := repo.GetBalance(ctx, fakeID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, errors.WalletNotFound)
	})
}

func TestConcurrency_OnlyDeposits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrency test in -short mode")
	}

	ctx := context.Background()

	// Запускаем PostgreSQL
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_pass",
			"POSTGRES_DB":       "test_wallet_db",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor:   wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer container.Terminate(ctx)

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432")
	require.NoError(t, err)

	connStr := fmt.Sprintf(
		"host=%s port=%s user=test_user password=test_pass dbname=test_wallet_db sslmode=disable",
		host, port.Port(),
	)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	defer pool.Close()

	sqlQuery := `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE TABLE IF NOT EXISTS wallets (
		id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		balance    BIGINT NOT NULL DEFAULT 0,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS transactions (
		id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		wallet_id      UUID NOT NULL REFERENCES wallets(id),
		operation_type TEXT NOT NULL,
		amount         BIGINT NOT NULL,
		created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);`

	_, err = pool.Exec(ctx, sqlQuery)
	require.NoError(t, err)

	repo := &PostgresWalletRepository{pool: pool}

	// Создаём кошелёк
	walletID, err := repo.CreateWallet(ctx)
	require.NoError(t, err)

	// Параметры
	const (
		numGoroutines = 1000
		amount        = 7 // любое положительное число
	)

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines)

	// Запускаем 1000 горутин: все — DEPOSIT
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := repo.UpdateBalance(ctx, walletID, amount, true) // только DEPOSIT
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	// Проверяем, что НЕТ критических ошибок
	for err := range errCh {
		t.Fatalf("❌ Критическая ошибка в конкурентной среде: %v", err)
	}

	// Проверяем баланс
	balance, err := repo.GetBalance(ctx, walletID)
	require.NoError(t, err)

	expected := int64(numGoroutines) * amount
	if balance != expected {
		t.Errorf("❌ Баланс неверен: получено %d, ожидалось %d", balance, expected)
		t.Fail()
	}

	t.Logf("✅ Успешно: %d операций DEPOSIT по %d → баланс = %d", numGoroutines, amount, balance)
}
