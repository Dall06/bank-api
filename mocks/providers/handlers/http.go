package handlers

import (
	"errors"
	"log"
	"net/http"

	"bank-api/env/consts"
	"bank-api/mocks/providers/domain"
	"bank-api/mocks/providers/ports"

	"github.com/labstack/echo/v4"
)

type ProviderHandler struct {
	usecase ports.ProviderUsecase
}

func NewProviderHandler(usecase ports.ProviderUsecase) *ProviderHandler {
	return &ProviderHandler{usecase: usecase}
}

func (h *ProviderHandler) MockInterceptorMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		mockID := ctx.Request().Header.Get(consts.MockIDHeaderKey)
		log.Printf("X-Mock-Id recibido: %q", mockID)
		
		switch mockID {
		case "id-insufficient-funds":
			return ctx.JSON(http.StatusBadRequest, domain.ExecuteErrorResponse{
				Status:  "REJECTED",
				Code:    "INSUFFICIENT_FUNDS",
				Message: "The account does not have enough balance to complete the transaction",
			})
		case "id-error-500":
			return ctx.JSON(http.StatusInternalServerError, domain.ExecuteErrorResponse{
				Status:  "REJECTED",
				Code:    "INTERNAL_PROVIDER_ERROR",
				Message: "internal provider crash",
			})
		case "":
			// No mock header, continue to default handler
			return next(ctx)
		default:
			// If a mock ID is provided but unknown, we can either return an error or let it pass.
			return ctx.JSON(http.StatusBadRequest, domain.ExecuteErrorResponse{
				Status:  "REJECTED",
				Code:    "INVALID_MOCK_ID",
				Message: "The provided X-Mock-Id is not supported by the provider mock",
			})
		}
	}
}

func (h *ProviderHandler) Execute(ctx echo.Context) error {
	var req domain.ExecuteRequest
	if err := ctx.Bind(&req); err != nil {
		return ctx.JSON(http.StatusBadRequest, domain.ExecuteErrorResponse{
			Status:  "REJECTED",
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	resp, err := h.usecase.Execute(ctx.Request().Context(), req)
	if err != nil {
		var providerErr *domain.ProviderError
		if errors.As(err, &providerErr) {
			return ctx.JSON(http.StatusBadRequest, domain.ExecuteErrorResponse{
				Status:  "REJECTED",
				Code:    providerErr.Code,
				Message: providerErr.Message,
			})
		}
		return ctx.JSON(http.StatusInternalServerError, domain.ExecuteErrorResponse{
			Status:  "REJECTED",
			Code:    "INTERNAL_ERROR",
			Message: "An internal error occurred",
		})
	}

	return ctx.JSON(http.StatusOK, resp)
}
