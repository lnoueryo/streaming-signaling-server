package main

import (
	"encoding/json"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)


func websocketHandler(c *gin.Context) {
    user := getUser(c)
    roomId := c.Param("roomId")
    room := rooms.getOrCreate(roomId)

    // Upgrade HTTP → WebSocket
    unsafeConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Errorf("upgrade failed: %v", err)
        return
    }

    ws := &ThreadSafeWriter{UserID: user.ID, Conn: unsafeConn, Mutex: sync.Mutex{}}

    // --- register WS ---
    room.listLock.Lock()
    room.wsConnections[ws.Conn] = ws
    room.listLock.Unlock()

    // ---- clean up ----
    defer func() {
        room.listLock.Lock()
        // 複数のデバイスでロビーに入室し、
        // 1つだけルームにいる状態でロビーのどれかのデバイスがリロードすると
        // ルームの方のPeerConnectionも切れてしまうため、peerConnectionがある時のみクローズ
        delete(room.wsConnections, ws.Conn)
        room.listLock.Unlock()

        ws.Close()
        rooms.cleanupEmptyRoom(roomId)
    }()

    msg := &WebsocketMessage{}

    for {
        // ---------- read WS message ----------
        _, raw, err := ws.ReadMessage()
        if err != nil {
            log.Errorf("WS read error: %v", err)
            return
        }

        if err := json.Unmarshal(raw, msg); err != nil {
            log.Errorf("json unmarshal error: %v", err)
            return
        }

        switch msg.Event {
        case "candidate":
            if err := AddCandidate(roomId, user, msg.Data); err != nil {
                log.Errorf("add candidate error: %v", err)
                return
            }
        case "answer":
            if err := SetAnswer(roomId, user, msg.Data); err != nil {
                log.Errorf("set answer error: %v", err)
                return
            }
        }
    }
}

func websocketViewerHandler(c *gin.Context) {
    user := getUser(c)
	uid, err := uuid.NewV7()
	if err != nil {
		log.Fatal(err)
	}
    user.ID = uid.String()
    roomId := c.Param("roomId")
    room := rooms.getOrCreate(roomId)

    unsafeConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Errorf("upgrade failed: %v", err)
        return
    }

    ws := &ThreadSafeWriter{UserID: user.ID, Conn: unsafeConn, Mutex: sync.Mutex{}}

    // --- register WS ---
    room.listLock.Lock()
    room.wsConnections[ws.Conn] = ws
    room.listLock.Unlock()

    // ---- clean up ----
    defer func() {
        room.listLock.Lock()
        delete(room.wsConnections, ws.Conn)
        room.listLock.Unlock()
        ws.Close()
        rooms.cleanupEmptyRoom(roomId)
    }()

    msg := &WebsocketMessage{}

    for {
        // ---------- read WS message ----------
        _, raw, err := ws.ReadMessage()
        if err != nil {
            log.Errorf("WS read error: %v", err)
            return
        }

        if err := json.Unmarshal(raw, msg); err != nil {
            log.Errorf("json unmarshal error: %v", err)
            return
        }

        switch msg.Event {
        case "offer":
            if err := CreateViewerPeer(roomId, user); err != nil {
                log.Errorf("create peer error: %v", err)
                return
            }
        case "candidate":
            if err := AddViewerCandidate(roomId, user, msg.Data); err != nil {
                log.Errorf("add candidate error: %v", err)
                return
            }
        case "answer":
            if err := SetViewerAnswer(roomId, user, msg.Data); err != nil {
                log.Errorf("set answer error: %v", err)
                return
            }
        }
    }
}