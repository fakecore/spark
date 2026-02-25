package validator

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/go-playground/validator/v10"
)

func init() {
	GetValidator()
}

var (
	once     sync.Once
	instance *validator.Validate
)

// GetValidator returns the singleton instance of the validator
func GetValidator() *validator.Validate {
	once.Do(func() {
		instance = validator.New()
		registerCustomValidations(instance)
	})
	return instance
}

// registerCustomValidations registers all custom validations
func registerCustomValidations(v *validator.Validate) {
	err := v.RegisterValidation("numberscomma", validateNumbersComma)
	if err != nil {
		fmt.Println("Error registering validation:", err)
	}
}

// validateNumbersComma validates a string of comma-separated numbers
func validateNumbersComma(fl validator.FieldLevel) bool {
	return len(fl.Field().String()) == 0 || regexp.MustCompile(`^\S+(,\S+)*$`).MatchString(fl.Field().String())
}

// QuickValidate provides a quick way to validate a struct
func QuickValidate(s interface{}) error {
	return GetValidator().Struct(s)
}

// CommaSeparatedNumbers is a struct that contains a field for comma-separated numbers
type CommaSeparatedNumbers struct {
	Numbers string `validate:"numberscomma"`
}
