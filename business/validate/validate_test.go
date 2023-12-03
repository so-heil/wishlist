package validate_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/so-heil/wishlist/business/validate"
)

type user struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Username string `json:"username" validate:"required,lowercase,alphanum,max=20"`
	Password string `json:"password" validate:"required,password"`
}

func TestCheckStruct(t *testing.T) {
	err := validate.Init()
	if err != nil {
		t.Fatalf("create validator: %s", err)
	}

	validUser := user{
		Name:     "some name",
		Email:    "test@test.com",
		Username: "test",
		Password: "jiwulR18p",
	}
	if err := validate.Check(validUser); err != nil {
		t.Errorf("should not yield validUser as invalid, checkErr: %s", err)
	} else {
		t.Log("validated validUser")
	}

	emptyNameWeakPassword := user{
		Name:     "",
		Email:    "test@test.com",
		Username: "test",
		Password: "test1234",
	}
	if checkErr := validate.Check(emptyNameWeakPassword); checkErr != nil {
		var ferr validate.FieldErrors
		if !errors.As(checkErr, &ferr) {
			t.Errorf("should return error of type FieldErrors")
		} else {
			fields := ferr.Fields()
			_, hasName := fields["name"]
			_, hasPass := fields["password"]
			if !hasName && !hasPass {
				t.Errorf("should have name and password in fields, fields: %s", fields)
			} else {
				t.Log("validated emptyNameWeakPassword correctly")
			}
		}

		var jsFerr validate.FieldErrors
		if err := json.Unmarshal([]byte(checkErr.Error()), &jsFerr); err != nil {
			t.Errorf("FieldError.Error() should produce a valid json string that can be parsed back, got: %s, checkErr: %s", checkErr.Error(), err)
		}
		if !reflect.DeepEqual(jsFerr, ferr) {
			t.Errorf("checkErr parsed back should be equal to itself, is: %+v", jsFerr)
		} else {
			t.Log("tested FieldError.Error()")
		}
	} else {
		t.Error("should return error for emptyNameWeakPassword")
	}
}
