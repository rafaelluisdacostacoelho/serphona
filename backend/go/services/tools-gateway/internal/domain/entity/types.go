// Package entity contains domain entities for tools gateway.
package entity

// AuthType represents the authentication type.
type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeOAuth2 AuthType = "oauth2"
	AuthTypeCustom AuthType = "custom"
)

// String returns the string representation of AuthType.
func (a AuthType) String() string {
	return string(a)
}

// IsValid checks if the AuthType is valid.
func (a AuthType) IsValid() bool {
	switch a {
	case AuthTypeNone, AuthTypeAPIKey, AuthTypeBearer, AuthTypeBasic, AuthTypeOAuth2, AuthTypeCustom:
		return true
	default:
		return false
	}
}
