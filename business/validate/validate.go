// Package validate provides functions to validate structs with validate tag
// and defines FieldErrors as result of failed validations
package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type validate struct {
	v     *validator.Validate
	trans ut.Translator
}

var once sync.Once
var v *validate

func Init() error {
	var rterr error
	once.Do(func() {
		instance := validator.New()

		enLang := en.New()
		uni := ut.New(enLang, enLang)
		trans, found := uni.GetTranslator("en")
		if !found {
			rterr = errors.New("cannot find enLang translation")
			return
		}

		instance.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		if err := en_translations.RegisterDefaultTranslations(instance, trans); err != nil {
			rterr = err
			return
		}

		if err := addPassword(instance, trans); err != nil {
			rterr = fmt.Errorf("register password validator: %w", err)
			return
		}

		v = &validate{
			v:     instance,
			trans: trans,
		}
	})
	return rterr
}

func Check(val any) error {
	if err := v.v.Struct(val); err != nil {
		verrors, ok := err.(validator.ValidationErrors)
		if !ok {
			return err
		}

		var fields FieldErrors
		for _, verror := range verrors {
			field := FieldError{
				Field: verror.Field(),
				Err:   verror.Translate(v.trans),
			}
			fields = append(fields, field)
		}

		return fields
	}

	return nil
}
