package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func deleteRtcClient(c *gin.Context) {
	user := getUser(c)
	roomId := c.Param("roomId")
	userId := user.ID
	room, ok := rooms.getRoom(roomId); if !ok {
		err := &ErrorResponse{
			"トークルームが既に存在しません",
			http.StatusNotFound,
			"no-target-room",
		}
		err.response(c)
		return
	}
	client, ok := room.clients[userId]; if !ok {
		err := &ErrorResponse{
			"ユーザーは既にトークルームから退出しています",
			http.StatusNotFound,
			"no-target-user",
		}
		err.response(c)
		return
	}
	room.listLock.Lock()
	var data interface{}
	client.WS.Send("close", data)
	client.Peer.Close()
	client.WS.Close()
	delete(room.clients, userId)
	room.listLock.Unlock()
	c.JSON(http.StatusNoContent, gin.H{})
}

func checkIfCanJoin(c *gin.Context) {
	user := getUser(c)
	roomId := c.Param("roomId")
	userId := user.ID
	room, ok := rooms.getRoom(roomId); if !ok {
		c.JSON(http.StatusOK, room)
		return
	}
	_, ok = room.clients[userId]; if !ok {
		c.JSON(http.StatusOK, room)
		return
	}

	err := &ErrorResponse{
		"ユーザーは既に別の端末で参加しています",
		http.StatusConflict,
		"already-join",
	}
	err.response(c)
}

func getUser(c *gin.Context) UserInfo {
	userVal, exists := c.Get("user")
	if !exists {
		return UserInfo{}
	}
	return userVal.(UserInfo)
}

type ErrorResponse struct {
	message string
	statusCode int
	errorCode string
}

func (er *ErrorResponse)response(c *gin.Context) {
	c.JSON(er.statusCode, gin.H{
		"message": er.message,
		"statusCode": er.statusCode,
		"errorCode": er.errorCode,
	})
}