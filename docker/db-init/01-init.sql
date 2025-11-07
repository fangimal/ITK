-- Создаём расширение для UUID (требуется явно)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Таблица кошельков
CREATE TABLE IF NOT EXISTS wallets (
                                       id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    balance    BIGINT NOT NULL DEFAULT 0,  -- в копейках/центах (целое!)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Таблица операций (аудит)
CREATE TABLE IF NOT EXISTS transactions (
                                            id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id      UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    operation_type TEXT NOT NULL CHECK (operation_type IN ('DEPOSIT', 'WITHDRAW')),
    amount         BIGINT NOT NULL CHECK (amount > 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

-- Индексы для производительности (1000 RPS!)
CREATE INDEX IF NOT EXISTS idx_wallets_id ON wallets(id);
CREATE INDEX IF NOT EXISTS idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);
