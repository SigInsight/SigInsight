package implsession

import (
	"context"
	"net/url"
	"slices"
	"time"

	"github.com/SigNoz/signoz/pkg/authn"
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/factory"
	"github.com/SigNoz/signoz/pkg/modules/organization"
	"github.com/SigNoz/signoz/pkg/modules/session"
	"github.com/SigNoz/signoz/pkg/modules/user"
	"github.com/SigNoz/signoz/pkg/tokenizer"
	"github.com/SigNoz/signoz/pkg/types"
	"github.com/SigNoz/signoz/pkg/types/authtypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type module struct {
	settings   factory.ScopedProviderSettings
	authNs     map[authtypes.AuthNProvider]authn.AuthN
	userSetter user.Setter
	userGetter user.Getter
	tokenizer  tokenizer.Tokenizer
	orgGetter  organization.Getter
}

func NewModule(providerSettings factory.ProviderSettings, authNs map[authtypes.AuthNProvider]authn.AuthN, userSetter user.Setter, userGetter user.Getter, tokenizer tokenizer.Tokenizer, orgGetter organization.Getter) session.Module {
	return &module{
		settings:   factory.NewScopedProviderSettings(providerSettings, "github.com/SigNoz/signoz/pkg/modules/session/implsession"),
		authNs:     authNs,
		userSetter: userSetter,
		userGetter: userGetter,
		tokenizer:  tokenizer,
		orgGetter:  orgGetter,
	}
}

func (module *module) GetSessionContext(ctx context.Context, email valuer.Email, _ *url.URL) (*authtypes.SessionContext, error) {
	context := authtypes.NewSessionContext()

	orgs, err := module.orgGetter.ListByOwnedKeyRange(ctx)
	if err != nil {
		return nil, err
	}

	if len(orgs) == 0 {
		context.Exists = false
		return context, nil
	}

	var orgIDs []valuer.UUID
	for _, org := range orgs {
		orgIDs = append(orgIDs, org.ID)
	}

	users, err := module.userGetter.ListUsersByEmailAndOrgIDs(ctx, email, orgIDs)
	if err != nil {
		return nil, err
	}

	// filter out deleted users
	users = slices.DeleteFunc(users, func(user *types.User) bool { return user.ErrIfDeleted() != nil })

	if len(users) == 0 {
		context.Exists = false

		for _, org := range orgs {
			context = context.AddOrgContext(module.getOrgSessionContext(org))
		}

		return context, nil
	}

	context.Exists = true
	for _, user := range users {
		idx := slices.IndexFunc(orgs, func(org *types.Organization) bool {
			return org.ID == user.OrgID
		})

		if idx == -1 {
			continue
		}

		org := orgs[idx]
		context = context.AddOrgContext(module.getOrgSessionContext(org))
	}

	return context, nil
}

func (module *module) CreatePasswordAuthNSession(ctx context.Context, authNProvider authtypes.AuthNProvider, email valuer.Email, password string, orgID valuer.UUID) (*authtypes.Token, error) {
	passwordAuthN, err := getProvider[authn.PasswordAuthN](authNProvider, module.authNs)
	if err != nil {
		return nil, err
	}

	identity, err := passwordAuthN.Authenticate(ctx, email.String(), password, orgID)
	if err != nil {
		return nil, err
	}

	return module.tokenizer.CreateToken(ctx, identity, map[string]string{})
}

func (module *module) RotateSession(ctx context.Context, accessToken string, refreshToken string) (*authtypes.Token, error) {
	return module.tokenizer.RotateToken(ctx, accessToken, refreshToken)
}

func (module *module) DeleteSession(ctx context.Context, accessToken string) error {
	return module.tokenizer.DeleteToken(ctx, accessToken)
}

func (module *module) GetRotationInterval(context.Context) time.Duration {
	return module.tokenizer.Config().Rotation.Interval
}

func (module *module) getOrgSessionContext(org *types.Organization) *authtypes.OrgSessionContext {
	return authtypes.NewOrgSessionContext(org.ID, org.Name).AddPasswordAuthNSupport(authtypes.AuthNProviderEmailPassword)
}

func getProvider[T authn.AuthN](authNProvider authtypes.AuthNProvider, authNs map[authtypes.AuthNProvider]authn.AuthN) (T, error) {
	var provider T

	provider, ok := authNs[authNProvider].(T)
	if !ok {
		return provider, errors.New(errors.TypeNotFound, errors.CodeNotFound, "authn provider not found")
	}

	return provider, nil
}
