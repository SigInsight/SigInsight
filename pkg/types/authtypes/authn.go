package authtypes

import (
	"context"
	"encoding/json"

	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/valuer"
)

var (
	AuthNProviderEmailPassword = AuthNProvider{valuer.NewString("email_password")}
)

type AuthNProvider struct{ valuer.String }

type Identity struct {
	UserID        valuer.UUID    `json:"userId"`
	OrgID         valuer.UUID    `json:"orgId"`
	IdenNProvider IdentNProvider `json:"identNProvider"`
	Email         valuer.Email   `json:"email"`
	Role          types.Role     `json:"role"`
}

func NewIdentity(userID valuer.UUID, orgID valuer.UUID, email valuer.Email, role types.Role, identNProvider IdentNProvider) *Identity {
	return &Identity{
		UserID:        userID,
		OrgID:         orgID,
		Email:         email,
		Role:          role,
		IdenNProvider: identNProvider,
	}
}

func (typ Identity) MarshalBinary() ([]byte, error) {
	return json.Marshal(typ)
}

func (typ *Identity) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, typ)
}

func (typ *Identity) ToClaims() Claims {
	return Claims{
		UserID:         typ.UserID.String(),
		Email:          typ.Email.String(),
		Role:           typ.Role,
		OrgID:          typ.OrgID.String(),
		IdentNProvider: typ.IdenNProvider.StringValue(),
	}
}

type AuthNStore interface {
	// Get user and factor password by email and orgID.
	GetActiveUserAndFactorPasswordByEmailAndOrgID(ctx context.Context, email string, orgID valuer.UUID) (*types.User, *types.FactorPassword, []*UserRole, error)
}
