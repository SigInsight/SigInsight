package authn

import (
	"context"

	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type AuthN interface{}

type PasswordAuthN interface {
	// Authenticate a user using email, password and orgID
	Authenticate(context.Context, string, string, valuer.UUID) (*authtypes.Identity, error)
}
