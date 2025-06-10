package common

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type AppError struct {
	StatusCode int    `json:"status_code"`
	RootErr    error  `json:"root_error"`
	Message    string `json:"message"`
	Log        string `json:"log"`
	Key        string `json:"error_key"`
}

func (e *AppError) Error() string {
	return e.RootError().Error()
}

func (e *AppError) RootError() error {
	if err, ok := e.RootErr.(*AppError); ok {
		return err.RootError()
	}
	return e.RootErr
}
func NewFullErrorResponse(statusCode int, root error, message, log, key string) *AppError {
	return &AppError{
		StatusCode: statusCode,
		RootErr:    root,
		Message:    message,
		Log:        log,
		Key:        key,
	}
}

func NewErrorResponse(root error, message, log, key string) *AppError {
	return &AppError{
		StatusCode: http.StatusBadRequest,
		RootErr:    root,
		Message:    message,
		Log:        log,
		Key:        key,
	}
}

func NewCustomError(err error, message, key string) *AppError {
	return NewErrorResponse(err, message, err.Error(), key)
}

func NewUnauthorizedErrorResponse(root error, message, key string) *AppError {
	return &AppError{
		StatusCode: http.StatusUnauthorized,
		RootErr:    root,
		Message:    message,
		Key:        key,
	}
}

func ErrDB(err error) *AppError {
	return NewFullErrorResponse(http.StatusInternalServerError, err, "Database error", err.Error(), ErrorDatabase)
}

func ErrInvalidRequest(err error) *AppError {
	return NewErrorResponse(err, "Invalid request", err.Error(), ErrorInvalidRequest)
}

func ErrInternal(err error) *AppError {
	return NewFullErrorResponse(http.StatusInternalServerError, err, "Something went wrong with the server", err.Error(), ErrorInternalServer)
}

func ErrCannotCreateEntity(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("Can't create %s", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorCannotCreate, entity))
}

func ErrCannotGetEntity(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("Can't get %s", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorCannotGet, entity))
}

func ErrCannotUpdateEntity(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("Can't update %s", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorCannotUpdate, entity))
}

func ErrCannotDeleteEntity(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("Can't delete %s", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorCannotDelete, entity))
}

func ErrCannotGetListEntity(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("Can't get list %s", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorCannotGetList, entity))
}

func ErrEntityExisted(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("%s is existed", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorExistedEntity, entity))
}

func ErrEntityNotFoundEntity(entity string, err error) *AppError {
	return NewCustomError(err, fmt.Sprintf("Can't found %s", strings.ToLower(entity)), fmt.Sprintf("%s:%s", ErrorExistedEntity, entity))
}

func ErrNoPermission(err error) *AppError {
	return NewCustomError(err, "You have no permission", ErrorNoPermission)
}

func ErrorSimpleMessage(message string) error {
	return errors.New(message)
}
