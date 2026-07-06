package router_test

import (
	"testing"

	"bank-api/opt/router"

	"github.com/labstack/echo/v4"
)

func TestSetAppUse(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Should set middlewares without panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			router.SetAppUse(e)
		})
	}
}
