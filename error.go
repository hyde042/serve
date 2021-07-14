package serve

import (
	"errors"
	"io/fs"
	"net/http"
)

var ErrMethodNotAllowed = errors.New("method not allowed")

func Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	http.Error(w, err.Error(), statusCode(err))
}

func statusCode(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, fs.ErrNotExist):
		return http.StatusNotFound
	case errors.Is(err, fs.ErrPermission):
		return http.StatusForbidden
	case errors.Is(err, fs.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed
	default:
		return http.StatusInternalServerError
	}
}
