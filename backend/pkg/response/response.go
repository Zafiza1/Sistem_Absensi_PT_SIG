// Package response defines the single JSON response envelope used by every
// API endpoint, so Flutter and Next.js clients only ever need to parse one
// shape.
package response

import "github.com/gin-gonic/gin"

// Envelope is the standard response body for every API endpoint.
//
//	{
//	  "success": true,
//	  "message": "Absensi berhasil",
//	  "data": { ... }
//	}
type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// OK writes a 2xx success response with the given HTTP status code.
func OK(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Envelope{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// Fail writes an error response with the given HTTP status code.
// errs may be nil, a validation error map, or any JSON-serializable detail.
func Fail(c *gin.Context, status int, message string, errs any) {
	c.JSON(status, Envelope{
		Success: false,
		Message: message,
		Errors:  errs,
	})
}
