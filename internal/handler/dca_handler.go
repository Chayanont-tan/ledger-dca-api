package handler

import (
	"encoding/json"
	"net/http"

	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/interface"
)

type DCAHandler struct {
	service interfaces.DCAService
}

func NewDCAHandler(service interfaces.DCAService) *DCAHandler {
	return &DCAHandler{service: service}
}

func (h *DCAHandler) HandleCreatePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.CreateDCAPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	res, err := h.service.CreatePlan(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}
