package ports

import (
	"github.com/labstack/echo/v4"
)

// TransactionHandler define la interfaz para el controlador HTTP (Puerto de Entrada)
type TransactionHandler interface {
	Create(c echo.Context) error
	Get(c echo.Context) error
}
