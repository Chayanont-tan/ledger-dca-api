package main

import (
	"log"
	"net/http"

	"ledger-dca-engine/internal/config"
	"ledger-dca-engine/internal/handler"
	"ledger-dca-engine/internal/repository"
	"ledger-dca-engine/internal/service"
)

func main() {
	db := config.InitDB()
	defer db.Close()

	ledgerRepo := repository.NewLedgerRepository(db)
	transferService := service.NewTransferService(ledgerRepo)
	transferHandler := handler.NewTransferHandler(transferService)

	http.HandleFunc("/api/v1/transfers", transferHandler.HandlerTransfer)
	log.Println("Ledger API server running on port :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}

}
