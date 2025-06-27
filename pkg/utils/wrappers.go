package utils

import (
	"github.com/gin-gonic/gin"
)

// ErrorJSONGin sends a JSON error response using Gin's context.
// It sets the appropriate HTTP status code and sends a JSON body
// in the format: {"error": "message"}.
func ErrorJSONGin(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{"error": msg})
}

// Note: The previous Chi-specific helpers (MethodGuard, WriteJSON, ErrorJSON)
// and ResponseWriterWrapper have been removed as they are superseded by
// Gin-specific mechanisms or no longer needed with Gin's context.
// For example, Gin handlers directly use c.JSON(), c.String(), etc., for responses,
// and c.Writer.Status() provides the status code for metrics.
// Method guarding is handled by Gin middleware defined in main.go.
