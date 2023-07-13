package options

import (
	"strings"

	"github.com/go-openapi/spec"
	"github.com/yubo/apiserver/pkg/server"
	"github.com/yubo/golib/util/errors"
)

type SecurityScheme struct {
	Name             string              `json:"name"`
	Type             server.SecurityType `json:"type" description:"base|bearer|token|implicit|password|application|accessCode"`
	FieldName        string              `json:"fieldName" description:"used for token"`
	ValueSource      string              `json:"valueSource" description:"used for token, header|query|cookie"`
	AuthorizationURL string              `json:"authorizationURL" description:"used for OAuth2"`
	TokenURL         string              `json:"tokenURL" description:"used for OAuth2"`
}

func NewSecuritySchemes() []SecurityScheme {
	return []SecurityScheme{{
		Name: "BearerToken",
		Type: "bearer",
	}}
}

func ToSpecSecuritySchemes(in []SecurityScheme) ([]*spec.SecurityScheme, error) {
	var schemes []*spec.SecurityScheme
	for _, v := range in {
		scheme, err := v.SecurityScheme()
		if err != nil {
			return nil, err
		}
		schemes = append(schemes, scheme)
	}
	return schemes, nil
}

func (p *SecurityScheme) SecurityScheme() (*spec.SecurityScheme, error) {
	if p.Name == "" {
		return nil, errors.New("name must be set")
	}
	switch strings.ToLower(string(p.Type)) {
	case string(server.SecurityTypeBase):
		return spec.BasicAuth(), nil
	case string(server.SecurityTypeBearer):
		return spec.APIKeyAuth("Authorization", "header"), nil
	case string(server.SecurityTypeAPIKey):
		if p.FieldName == "" {
			return nil, errors.Errorf("fieldName must be set for %s", p.Type)
		}
		if p.ValueSource == "" {
			return nil, errors.Errorf("valueSource must be set for %s", p.Type)
		}
		return spec.APIKeyAuth(p.FieldName, p.ValueSource), nil
	case string(server.SecurityTypeImplicit):
		if p.AuthorizationURL == "" {
			return nil, errors.Errorf("authorizationURL must be set for %s", p.Type)
		}
		return spec.OAuth2Implicit(p.AuthorizationURL), nil
	case string(server.SecurityTypePassword):
		if p.TokenURL == "" {
			return nil, errors.Errorf("tokenURL must be set for %s", p.Type)
		}
		return spec.OAuth2Password(p.TokenURL), nil
	case string(server.SecurityTypeApplication):
		if p.TokenURL == "" {
			return nil, errors.Errorf("tokenURL must be set for %s", p.Type)
		}
		return spec.OAuth2Application(p.TokenURL), nil
	case string(server.SecurityTypeAccessCode):
		if p.TokenURL == "" {
			return nil, errors.Errorf("tokenURL must be set for %s", p.Type)
		}
		if p.AuthorizationURL == "" {
			return nil, errors.Errorf("authorizationURL must be set for %s", p.Type)
		}
		return spec.OAuth2AccessToken(p.AuthorizationURL, p.TokenURL), nil
	default:
		return nil, errors.Errorf("scheme.type %s is invalid, should be one of %s", p.Type,
			strings.Join([]string{
				string(server.SecurityTypeBase),
				string(server.SecurityTypeBearer),
				string(server.SecurityTypeAPIKey),
				string(server.SecurityTypeImplicit),
				string(server.SecurityTypePassword),
				string(server.SecurityTypeApplication),
				string(server.SecurityTypeAccessCode),
			}, ", "))
	}
}
