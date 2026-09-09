package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"ledger-dca-engine/internal/config"
	"ledger-dca-engine/internal/handler"
	"ledger-dca-engine/internal/repository"
	"ledger-dca-engine/internal/service"
	"ledger-dca-engine/internal/worker"
)

func main() {
	db := config.InitDB()
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Ledger Module (ของเดิม)
	ledgerRepo := repository.NewLedgerRepository(db)
	transferService := service.NewTransferService(ledgerRepo)
	transferHandler := handler.NewTransferHandler(transferService)

	// 2. DCA Module (เพิ่มเข้ามาใหม่ โดยเอา transferService ตัวเดิมส่งเข้าไป)
	dcaRepo := repository.NewDCARepository(db)
	dcaService := service.NewDCAService(dcaRepo, transferService)
	dcaHandler := handler.NewDCAHandler(dcaService)

	// 3. เริ่มรัน DCA Background Worker (ตรวจรอบทุกๆ 5 วินาที)
	dcaWorker := worker.NewDCAWorker(dcaService, 5*time.Second)
	dcaWorker.Start(ctx)

	// 4. Routes
	http.HandleFunc("/api/v1/transfers", transferHandler.HandlerTransfer)
	http.HandleFunc("/api/v1/dca/plans", dcaHandler.HandleCreatePlan)

	log.Println("Ledger & DCA API server running on port :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
