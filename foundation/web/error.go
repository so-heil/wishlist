package web

import (
	"encoding/json"
	"errors"
)

// EndUserError is an error that is sent to the end-user
type EndUserError struct {
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Status  int               `json:"-"`
}

func EUEFromError(err error, status int) EndUserError {
	return EndUserError{
		Message: err.Error(),
		Status:  status,
	}
}

func IsEndUserError(err error) bool {
	var eue EndUserError
	return errors.As(err, &eue)
}

// ExternalError represents errors that happen because an external service is not accessible
type ExternalError struct {
	Err error
}

func (ee ExternalError) Error() string {
	return ee.Err.Error()
}

func (eue EndUserError) Error() string {
	jsn, err := json.Marshal(eue)
	if err != nil {
		return err.Error()
	}

	return string(jsn)
}

func IsExternalError(err error) bool {
	var ee ExternalError
	return errors.As(err, &ee)
}
