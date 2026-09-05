package handler

import (
	"encoding/json"
	"errors"
	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/interface"
	"ledger-dca-engine/internal/service"
	"net/http"
)

type TransferHandler struct {
	// แก้จาก service.TransferService เป็น interfaces.TransferService
	service interfaces.TransferService
}

func NewTransferHandler(service interfaces.TransferService) *TransferHandler {
	return &TransferHandler{service: service}
}

func (h *TransferHandler) HandlerTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req dto.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	res, err := h.service.ExecuteTransfer(r.Context(), req)

	if err != nil {
		if errors.Is(err, service.ErrInsufficientBalance) {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		if errors.Is(err, service.ErrInvalidAmount) || errors.Is(err, service.ErrSameAccountTransfer) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if res.Status == "ALREADY_PROCESSED" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(res)

}
