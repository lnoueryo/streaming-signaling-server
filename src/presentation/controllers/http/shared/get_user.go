package shared

import (
	"github.com/gin-gonic/gin"
	application_shared "streaming-signaling.jounetsism.biz/src/application/shared"
)

func GetUser(c *gin.Context) application_shared.AuthUser {
	userVal, exists := c.Get("user")
	if !exists {
		return application_shared.AuthUser{}
	}
	return userVal.(application_shared.AuthUser)
}