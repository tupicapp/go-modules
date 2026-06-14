package validator

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/cockroachdb/errors"
	playgroundLib "github.com/go-playground/validator/v10"
	"github.com/tupicapp/go-modules/apperror"
)

// localeRegexp matches locale codes in the form en-US (ISO 639-1 + ISO 3166-1 alpha-2).
var localeRegexp = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)

// Playground validates structs using github.com/go-playground/validator.
type Playground struct {
	validator *playgroundLib.Validate
}

// Option extends the underlying validator with service-specific aliases or custom validation functions.
type Option func(v *playgroundLib.Validate)

// WithAlias registers a tag alias (e.g. "asset_category" → "required,oneof=...").
func WithAlias(alias, tags string) Option {
	return func(v *playgroundLib.Validate) { v.RegisterAlias(alias, tags) }
}

// WithValidation registers a custom validation function under the given tag.
func WithValidation(tag string, fn playgroundLib.Func) Option {
	return func(v *playgroundLib.Validate) {
		if err := v.RegisterValidation(tag, fn); err != nil {
			panic(fmt.Sprintf("failed to register %q validator: %v", tag, err))
		}
	}
}

func NewPlayground(opts ...Option) *Playground {
	v := playgroundLib.New()
	if err := v.RegisterValidation("locale", func(fl playgroundLib.FieldLevel) bool {
		return localeRegexp.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("failed to register locale validator: %v", err))
	}
	for _, opt := range opts {
		opt(v)
	}
	return &Playground{validator: v}
}

func (v *Playground) Validate(i interface{}) error {
	if err := v.validator.Struct(i); err != nil {
		var validationErrors playgroundLib.ValidationErrors
		if errors.As(err, &validationErrors) {
			messages := make(map[string]string, len(validationErrors))
			for _, ve := range validationErrors {
				messages[ve.Field()] = v.makeMessage(ve)
			}
			return apperror.Validation(messages)
		}
		return errors.WithStack(err)
	}
	return nil
}

func (v *Playground) makeMessage(err playgroundLib.FieldError) string {
	field := err.Field()
	tag := err.ActualTag()
	if tag == "" {
		tag = err.Tag()
	}

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required.", field)
	case "email":
		return fmt.Sprintf("%s must be a valid email address.", field)
	case "min":
		if isNumericKind(err.Kind()) {
			return fmt.Sprintf("%s must be at least %s.", field, err.Param())
		}
		return fmt.Sprintf("%s must be at least %s characters long.", field, err.Param())
	case "max":
		if isNumericKind(err.Kind()) {
			return fmt.Sprintf("%s must be at most %s.", field, err.Param())
		}
		return fmt.Sprintf("%s must be at most %s characters long.", field, err.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long.", field, err.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s.", field, err.Param())
	case "required_without":
		return fmt.Sprintf("%s is required when %s is not provided.", field, err.Param())
	case "excluded_with":
		return fmt.Sprintf("%s must not be provided together with %s.", field, err.Param())
	case "locale":
		return fmt.Sprintf("%s must be a valid locale code (e.g. en-US).", field)
	case "uuid", "uuid3", "uuid4", "uuid5",
		"uuid_rfc4122", "uuid3_rfc4122", "uuid4_rfc4122", "uuid5_rfc4122":
		return fmt.Sprintf("%s must be a valid UUID.", field)
	default:
		return fmt.Sprintf("%s is invalid (%s).", field, tag)
	}
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

var _ Validator = (*Playground)(nil)
