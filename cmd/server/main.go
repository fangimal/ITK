package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fangimal/ITK/internal/config"
	"github.com/fangimal/ITK/internal/handlers"
	"github.com/fangimal/ITK/internal/repository"
	"github.com/julienschmidt/httprouter"
)

const (
	createWallet    = "/api/v1/wallets"                    // POST — создание
	operation       = "/api/v1/wallet"                     // POST — операция
	getBalance      = "/api/v1/wallets/:uuid"              // GET — баланс
	getTransactions = "/api/v1/wallets/:uuid/transactions" // GET — аудит
)

func main() {
	cfg := config.Load()

	// Подключаемся к БД
	repo, err := repository.NewPostgresWalletRepository(cfg)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к БД: %v", err)
	}
	defer repo.Close()

	router := httprouter.New()
	walletHandler := handlers.NewWalletHandler(repo)

	// Регистрируем обработчики с логированием
	router.POST(createWallet, logRequest(walletHandler.CreateWallet))
	router.POST(operation, logRequest(walletHandler.Operation))
	router.GET(getBalance, logRequest(walletHandler.GetBalance))
	router.GET(getTransactions, logRequest(walletHandler.GetTransactions))

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("⏳ Получен сигнал завершения. Завершаем сервер...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	log.Printf("🚀 Сервер запущен на порту %s", cfg.AppPort)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("❌ Сервер упал: %v", err)
	}
	log.Println("✅ Сервер остановлен корректно")
}

// logRequest — middleware для логирования
func logRequest(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		log.Printf("[%s] %s %s", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.URL.Path)
		handler(w, r, ps)
	}
}
