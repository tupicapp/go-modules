// Package apperror defines the platform-wide application error type used to
// map domain and application failures to transport-level responses.
package apperror

import (
	"github.com/cockroachdb/errors"
)

type Type string

const (
	TypeLogic          Type = "logic"
	TypeValidation     Type = "validation"
	TypeNotFound       Type = "not_found"
	TypeAuthentication Type = "authentication"
	TypeAuthorization  Type = "authorization"
)

type Code string

const (
	CodeValidation     = Code(TypeValidation)
	CodeNotFound       = Code(TypeNotFound)
	CodeAuthentication = Code(TypeAuthentication)
	CodeAuthorization  = Code(TypeAuthorization)
)

type AppError struct {
	Type    Type          `json:"type"`
	Code    Code          `json:"code"`
	Message string        `json:"message"`
	Details []interface{} `json:"details"`
}

func (e *AppError) Error() string { return e.Message }

func (e *AppError) IsLogic() bool          { return e.Type == TypeLogic }
func (e *AppError) IsValidation() bool     { return e.Type == TypeValidation }
func (e *AppError) IsNotFound() bool       { return e.Type == TypeNotFound }
func (e *AppError) IsAuthentication() bool { return e.Type == TypeAuthentication }
func (e *AppError) IsAuthorization() bool  { return e.Type == TypeAuthorization }

func Logic(message string, code Code) *AppError {
	return &AppError{Type: TypeLogic, Code: code, Message: message}
}

func Validation(validationErrors map[string]string) *AppError {
	var metadata []interface{}
	for k, v := range validationErrors {
		metadata = append(metadata, map[string]interface{}{"field": k, "message": v})
	}
	return &AppError{Type: TypeValidation, Code: CodeValidation, Message: "Validation failed.", Details: metadata}
}

func NotFound(message string) *AppError {
	return &AppError{Type: TypeNotFound, Code: CodeNotFound, Message: message}
}

func NotFoundEntity(entity string) *AppError {
	return &AppError{Type: TypeNotFound, Code: CodeNotFound, Message: entity + " not found."}
}

func Authentication(message string) *AppError {
	return &AppError{Type: TypeAuthentication, Code: CodeAuthentication, Message: message}
}

func Authorization(message string) *AppError {
	return &AppError{Type: TypeAuthorization, Code: CodeAuthorization, Message: message}
}

func IsAppError(err error) bool {
	var appError *AppError
	return errors.As(err, &appError)
}
