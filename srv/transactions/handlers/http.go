package handlers

import (
	"context"
	"net/http"
	"strconv"

	"bank-api/env/consts"
	"bank-api/pkg/errs"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type TransactionHandler struct {
	purchaseUseCase ports.PurchaseUseCase
	getUseCase      ports.GetUseCase
	updateUseCase   ports.UpdateUseCase
}

func NewTransactionHandler(
	purchaseUseCase ports.PurchaseUseCase,
	getUseCase ports.GetUseCase,
	updateUseCase ports.UpdateUseCase,
) *TransactionHandler {
	return &TransactionHandler{purchaseUseCase: purchaseUseCase, getUseCase: getUseCase, updateUseCase: updateUseCase}
}

// errs.handle es para el manejador de errores globar solo deberiamos retornar el errr errs.Value, etc

// @Summary Create a new transaction
// @Description Creates a new credit or debit transaction. Supports Idempotency via X-Idempotency-Key.
// @Tags Transactions
// @Accept json
// @Produce json
// @Param request body domain.CreateTransactionRequest true "Transaction request"
// @Success 201 {object} domain.TransactionResponse
// @Router /transactions [post]
func (h *TransactionHandler) Create(ctx echo.Context) error {
	idemKey := ctx.Request().Header.Get("X-Idempotency-Key")
	if idemKey == "" {
		// loggear warning
		idemKey = uuid.New().String()
	}

	reqCtx := context.WithValue(ctx.Request().Context(), consts.IdempotencyKeyContextKey, idemKey)

	var req domain.CreateTransactionRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ValueError("invalid request body")
	}

	resp, err := h.purchaseUseCase.Purchase(reqCtx, req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, resp)
}

// Get es el endpoint externo. El JWT solo valida que el token sea válido.
// accountId es un query param opcional para filtrar transacciones.
//
// @Summary Get transactions
// @Description Retrieves paginated transactions, optionally filtered by accountId.
// @Tags Transactions
// @Produce json
// @Param accountId query string false "Account ID to filter by"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} domain.GetTransactionsResponse
// @Router /transactions [get]
func (h *TransactionHandler) Get(ctx echo.Context) error {
	var page, limit int
	var err error

	if pageStr := ctx.QueryParam("page"); pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return errs.ValueError("invalid page value")
		}
	}

	if limitStr := ctx.QueryParam("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return errs.ValueError("invalid limit value")
		}
	}

	req := domain.GetTransactionsRequest{
		AccountID: ctx.QueryParam("accountId"),
		Status:    ctx.QueryParam("status"),
		Type:      ctx.QueryParam("type"),
		Page:      page,
		Limit:     limit,
	}

	reqCtx := ctx.Request().Context()
	if ctx.Request().Header.Get("Cache-Control") == "no-cache" {
		reqCtx = context.WithValue(reqCtx, consts.BypassCacheContextKey, true)
	}

	resp, err := h.getUseCase.GetAll(reqCtx, req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, resp)
}

// GetInternal es el endpoint interno protegido por Sigil.
// Solo acepta filtro por status y paginación. Sin accountId (lo usa el Cron).
func (h *TransactionHandler) GetInternal(ctx echo.Context) error {
	var page, limit int
	var err error

	if pageStr := ctx.QueryParam("page"); pageStr != "" {
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return errs.ValueError("invalid page value")
		}
	}

	if limitStr := ctx.QueryParam("limit"); limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return errs.ValueError("invalid limit value")
		}
	}

	req := domain.GetTransactionsRequest{
		Status: ctx.QueryParam("status"),
		Page:   page,
		Limit:  limit,
	}

	reqCtx := ctx.Request().Context()
	if ctx.Request().Header.Get("Cache-Control") == "no-cache" {
		reqCtx = context.WithValue(reqCtx, consts.BypassCacheContextKey, true)
	}

	resp, err := h.getUseCase.GetAll(reqCtx, req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, resp)
}

// UpdateStatusInternal es llamado solo desde Sigil (crons). El router lo protege con SigilVerifier.
func (h *TransactionHandler) UpdateStatusInternal(ctx echo.Context) error {
	txID := ctx.Param("id")
	if txID == "" {
		return errs.ValueError("missing transaction id")
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := ctx.Bind(&req); err != nil {
		return errs.ValueError("invalid body")
	}

	if err := h.updateUseCase.UpdateStatus(ctx.Request().Context(), txID, req.Status); err != nil {
		return err
	}

	return ctx.NoContent(http.StatusNoContent)
}
