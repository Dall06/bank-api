package middlewares

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type Health struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

func Akagi() echo.HandlerFunc {
	return func(ctx echo.Context) error {
		health := Health{
			Status:    "OK",
			Timestamp: time.Now().UTC(),
		}

		return ctx.JSON(http.StatusOK, health)
	}
}
