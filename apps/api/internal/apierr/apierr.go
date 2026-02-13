package apierr

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

type response struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Message)
}

func (e *APIError) Respond(c *gin.Context) {
	c.JSON(e.Status, response{Code: e.Code, Message: e.Message})
}

func (e *APIError) Abort(c *gin.Context) {
	e.Respond(c)
	c.Abort()
}

func New(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}
