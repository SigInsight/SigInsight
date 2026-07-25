package authtypes

import (
	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type SessionContext struct {
	Exists bool                 `json:"exists"`
	Orgs   []*OrgSessionContext `json:"orgs"`
}

type OrgSessionContext struct {
	ID           valuer.UUID  `json:"id"`
	Name         string       `json:"name"`
	AuthNSupport AuthNSupport `json:"authNSupport"`
	Warning      *errors.JSON `json:"warning,omitempty"`
}

type AuthNSupport struct {
	Password []PasswordAuthNSupport `json:"password"`
}

type PasswordAuthNSupport struct {
	Provider AuthNProvider `json:"provider"`
}

func NewSessionContext() *SessionContext {
	return &SessionContext{Exists: false, Orgs: []*OrgSessionContext{}}
}

func NewOrgSessionContext(orgID valuer.UUID, name string) *OrgSessionContext {
	return &OrgSessionContext{
		ID:   orgID,
		Name: name,
		AuthNSupport: AuthNSupport{
			Password: []PasswordAuthNSupport{},
		},
		Warning: nil,
	}
}

func (s *SessionContext) AddOrgContext(orgContext *OrgSessionContext) *SessionContext {
	s.Orgs = append(s.Orgs, orgContext)
	return s
}

func (s *OrgSessionContext) AddPasswordAuthNSupport(provider AuthNProvider) *OrgSessionContext {
	s.AuthNSupport.Password = append(s.AuthNSupport.Password, PasswordAuthNSupport{Provider: provider})
	return s
}

func (s *OrgSessionContext) AddWarning(warning error) *OrgSessionContext {
	s.Warning = errors.AsJSON(warning)
	return s
}
