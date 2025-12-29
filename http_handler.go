package main

import (
	"github.com/gin-gonic/gin"
)

func getUser(c *gin.Context) UserInfo {
	userVal, exists := c.Get("user")
	if !exists {
		return UserInfo{}
	}
	return userVal.(UserInfo)
}