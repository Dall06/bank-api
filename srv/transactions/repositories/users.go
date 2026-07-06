package repositories

import (
	"bank-api/pkg/requestor"
	"context"
	"fmt"
	"net/http"
	"time"

	"bank-api/opt/middlewares"
	"bank-api/pkg/errs"
	"bank-api/pkg/sigil"
	"bank-api/srv/transactions/ports"
)

type usersRepository struct {
	userURL     string
	sigilSigner *sigil.Signer
	httpClient  *requestor.Client
}

func NewUsersRepository(userURL string, signer *sigil.Signer) ports.UsersRepository {
	// 1. Instanciamos el cliente nativo de Go con el timeout
	nativeClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	return &usersRepository{
		userURL:     userURL,
		sigilSigner: signer,
		// 2. Lo envolvemos usando el constructor de tu paquete "requestor"
		httpClient: requestor.NewClient(nativeClient),
	}
}

func (c *usersRepository) ValidateUser(ctx context.Context, accountID string) error {
	url := fmt.Sprintf("%s/%s", c.userURL, accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return errs.InternalError("error creando request a users: %v", err)
	}

	middlewares.NewSigilHeaders(c.sigilSigner).AddHeaders(req, nil)

	// Retorna solo el objeto res y el error
	res, err := c.httpClient.Do(ctx, req, true)
	if err != nil {
		return err
	}

	// Evaluamos leyendo directamente el campo de la estructura devuelta
	if res.StatusCode == http.StatusNotFound {
		return errs.ValueError("el accountID proveído no existe o no es válido")
	}
	if res.StatusCode != http.StatusOK {
		return errs.InternalError("respuesta inesperada del servicio users: %d", res.StatusCode)
	}

	return nil
}
