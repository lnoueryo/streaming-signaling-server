package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pion/webrtc/v4"
)

func removeParticipant(c *gin.Context) {
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
	participant, ok := room.participants[userId]; if !ok {
		err := &ErrorResponse{
			"ユーザーは既にトークルームから退出しています",
			http.StatusNotFound,
			"no-target-user",
		}
		err.response(c)
		return
	}
	room.listLock.Lock()
	var data string
	participant.WS.Send("close", data)
	participant.PC.Close()
	delete(room.participants, userId)
	participant.WS.Close()
	delete(room.wsConnections, participant.WS.Conn)
	room.listLock.Unlock()
	participants := make([]gin.H, 0, len(room.participants))
	for _, participant := range room.participants {
		if participant.PC.ConnectionState() == webrtc.PeerConnectionStateConnected {
			participants = append(participants, gin.H{
				"id": participant.ID,
				"name": participant.Name,
				"email": participant.Email,
				"image": participant.Image,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id": room.ID,
		"participants": participants,
	})
}

func getRoom(c *gin.Context) {
	roomId := c.Param("roomId")
	room, ok := rooms.getRoom(roomId); if !ok {
		err := &ErrorResponse{
			"roomが存在しません",
			404,
			"not-found",
		}
		err.response(c)
		return
	}
	participants := make([]gin.H, 0, len(room.participants))

	for _, participant := range room.participants {
		if participant.PC.ConnectionState() == webrtc.PeerConnectionStateConnected {
			participants = append(participants, gin.H{
				"id": participant.ID,
				"name": participant.Name,
				"email": participant.Email,
				"image": participant.Image,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id": room.ID,
		"participants": participants,
	})
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