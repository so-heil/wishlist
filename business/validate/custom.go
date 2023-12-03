package validate

import (
	"fmt"
	"unicode"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

const minLength = 8

func passv(pass string) bool {
	if len(pass) < minLength {
		return false
	}

	var (
		upper int
		num   int
	)

	for _, char := range pass {
		switch {
		case unicode.IsUpper(char):
			upper++
		case unicode.IsNumber(char):
			num++
		}
	}

	return upper >= 1 && num >= 1
}

func addPassword(instance *validator.Validate, trans ut.Translator) error {
	const tag = "password"

	err := instance.RegisterTranslation(tag, trans, func(ut ut.Translator) error {
		return ut.Add(tag, "{0} should be at least 8 characters long containing at least one upper-case letter, and one number", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T(tag, fe.Field())
		return t
	})
	if err != nil {
		return fmt.Errorf("add password validation translation: %w", err)
	}

	if err := instance.RegisterValidation(
		"password",
		func(fl validator.FieldLevel) bool { return passv(fl.Field().String()) },
	); err != nil {
		return fmt.Errorf("register password validator: %w", err)
	}

	return nil
}
