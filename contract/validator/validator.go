// Package validator defines the Validator contract.
package validator

// Validator is an interface for validating structs, typically implemented by a struct validator library.
type Validator interface {
	Validate(s interface{}) error
}
