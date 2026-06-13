// Package validator provides the Validator contract plus a go-playground/validator implementation that maps violations
// to apperror.
package validator

// Validator is an interface for validating structs, typically implemented by a struct validator library.
type Validator interface {
	Validate(s interface{}) error
}
