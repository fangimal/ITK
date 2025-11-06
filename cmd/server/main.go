package main

import (
	"log"
	"net/http"
	"time"

	"github.com/fangimal/ITK/internal/handlers"
	"github.com/julienschmidt/httprouter"
)

const (
	createWallet    = "/api/v1/wallets"                    // POST — создание
	operation       = "/api/v1/wallet"                     // POST — операция
	getBalance      = "/api/v1/wallets/:uuid"              // GET — баланс
	getTransactions = "/api/v1/wallets/:uuid/transactions" // GET — аудит
)

func main() {
	router := httprouter.New()
	wallet := handlers.NewWalletHandler()

	// Регистрируем обработчики с логированием
	router.POST(createWallet, logRequest(wallet.CreateWallet))
	router.POST(operation, logRequest(wallet.Operation))
	router.GET(getBalance, logRequest(wallet.GetBalance))
	router.GET(getTransactions, logRequest(wallet.GetTransactions))

	log.Println("Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// logRequest — middleware для логирования
func logRequest(handler httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		log.Printf("[%s] %s %s", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.URL.Path)
		handler(w, r, ps)
	}
}
