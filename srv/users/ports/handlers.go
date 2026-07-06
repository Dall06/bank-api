package ports

import "github.com/labstack/echo/v4"

type UserHandler interface {
	Signup(c echo.Context) error
	Login(c echo.Context) error
	GetMe(c echo.Context) error
}
