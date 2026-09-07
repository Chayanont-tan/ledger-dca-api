package worker

import (
	"context"
	"log"
	"time"

	interfaces "ledger-dca-engine/internal/interface"
)

type DCAWorker struct {
	dcaService interfaces.DCAService
	interval   time.Duration
}

func NewDCAWorker(dcaService interfaces.DCAService, interval time.Duration) *DCAWorker {
	return &DCAWorker{
		dcaService: dcaService,
		interval:   interval,
	}
}

func (w *DCAWorker) Start(ctx context.Context) {

	ticker := time.NewTicker(w.interval)

	go func() {

		defer ticker.Stop()
		log.Printf("🚀 DCA Background Worker started (checking every %v)", w.interval)

		for {
			select {
			case <-ticker.C:

				if err := w.dcaService.ProcessDuePlans(ctx); err != nil {
					log.Printf("❌ DCA Worker Polling Error: %v", err)
				}

			case <-ctx.Done():
				log.Println("🛑 Stopping DCA Worker safely...")
				return
			}
		}
	}()
}
