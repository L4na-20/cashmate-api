package helpers

import "github.com/gin-gonic/gin"

// Success writes the response envelope shared by all JSON endpoints.
func Success(c *gin.Context, status int, message string, data any) {
	c.JSON(status, gin.H{"message": message, "data": data})
}

// SuccessWithMeta writes a list response while keeping pagination metadata
// separate from the data collection.
func SuccessWithMeta(c *gin.Context, status int, message string, data any, meta any) {
	c.JSON(status, gin.H{"message": message, "data": data, "meta": meta})
}

// Error writes the normalized API error envelope. errors is optional and is
// intended for field-level validation details.
func Error(c *gin.Context, status int, message string, errors any) {
	body := gin.H{"message": message}
	if errors != nil {
		body["errors"] = errors
	}
	c.AbortWithStatusJSON(status, body)
}
