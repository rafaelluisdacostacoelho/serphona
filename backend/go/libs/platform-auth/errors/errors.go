package errors

import "errors"

// Erros comuns de autenticação
var (
	// ErrUnauthorized indica que a autenticação falhou
	ErrUnauthorized = errors.New("unauthorized")

	// ErrInvalidToken indica que o token JWT é inválido
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired indica que o token JWT expirou
	ErrTokenExpired = errors.New("token expired")

	// ErrMissingToken indica que o token não foi fornecido
	ErrMissingToken = errors.New("missing token")

	// ErrInsufficientPermissions indica que o usuário não tem permissões suficientes
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// ErrInvalidCredentials indica que as credenciais são inválidas
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUserNotFound indica que o usuário não foi encontrado
	ErrUserNotFound = errors.New("user not found")

	// ErrUserInactive indica que o usuário está inativo
	ErrUserInactive = errors.New("user is inactive")

	// ErrUserNotVerified indica que o usuário não verificou o email
	ErrUserNotVerified = errors.New("user email not verified")

	// ErrInvalidRole indica que a role é inválida
	ErrInvalidRole = errors.New("invalid role")
)

// AuthError representa um erro de autenticação com código e mensagem
type AuthError struct {
	Code    string
	Message string
	Err     error
}

// Error implementa a interface error
func (e *AuthError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap permite usar errors.Is e errors.As
func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError cria um novo erro de autenticação
func NewAuthError(code, message string, err error) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Códigos de erro padronizados
const (
	CodeUnauthorized            = "UNAUTHORIZED"
	CodeInvalidToken            = "INVALID_TOKEN"
	CodeTokenExpired            = "TOKEN_EXPIRED"
	CodeMissingToken            = "MISSING_TOKEN"
	CodeInsufficientPermissions = "INSUFFICIENT_PERMISSIONS"
	CodeInvalidCredentials      = "INVALID_CREDENTIALS"
	CodeUserNotFound            = "USER_NOT_FOUND"
	CodeUserInactive            = "USER_INACTIVE"
	CodeUserNotVerified         = "USER_NOT_VERIFIED"
	CodeInvalidRole             = "INVALID_ROLE"
)
