package handlers

import (
	"net/http"

	"bank-api/pkg/errs"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/ports"

	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	get    ports.GetUseCase
	login  ports.LoginUseCase
	signup ports.SignupUseCase
}

func NewUserHandler(
	getUseCase ports.GetUseCase,
	signupUseCase ports.SignupUseCase,
	loginUseCase ports.LoginUseCase,
) ports.UserHandler {
	return &UserHandler{
		signup: signupUseCase,
		login:  loginUseCase,
		get:    getUseCase,
	}
}

// @Summary Register a new user
// @Description Creates a new user account.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body domain.SignupRequest true "Signup request"
// @Success 201 {object} domain.AuthResponse
// @Router /auth/signup [post]
func (h *UserHandler) Signup(ctx echo.Context) error {
	var req domain.SignupRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ValueError("cuerpo de petición inválido")
	}

	res, err := h.signup.Signup(ctx.Request().Context(), req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, res)
}

// @Summary Authenticate user
// @Description Logs in a user and returns a JWT token.
// @Tags Users
// @Accept json
// @Produce json
// @Param request body domain.LoginRequest true "Login request"
// @Success 200 {object} domain.AuthResponse
// @Router /auth/login [post]
func (h *UserHandler) Login(ctx echo.Context) error {
	var req domain.LoginRequest
	if err := ctx.Bind(&req); err != nil {
		return errs.ValueError("cuerpo de petición inválido")
	}

	res, err := h.login.Login(ctx.Request().Context(), req)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, res)
}

// GetMe sirve tanto para JWT externo (user_id viene del contexto) como para Sigil interno (:id en el param)
func (h *UserHandler) GetMe(ctx echo.Context) error {
	id := ctx.Param("id")

	if id == "" {
		userID, ok := ctx.Get("user_id").(string)
		if !ok || userID == "" {
			return errs.UnauthorizedError("no se pudo identificar el usuario")
		}
		id = userID
	}

	user, err := h.get.GetByID(ctx.Request().Context(), id)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, user)
}
