package room_memory_repository

import (
	"context"
	"errors"
	"sync"
	"time"
	room_memory_repository "streaming-server.com/application/ports/repositories/memory"
	room_entity "streaming-server.com/domain/entities/room"
	"streaming-server.com/infrastructure/logger"
)

type RoomRepository struct {
	rooms map[string]*room_entity.RuntimeRoom // roomId -> runtime
	mu    sync.RWMutex
}

var (
	_ room_memory_repository.IRoomRepository = (*RoomRepository)(nil)
	log = *logger.Log
)

func NewRoomRepository() *RoomRepository {
	rooms := make(map[string]*room_entity.RuntimeRoom)
	return &RoomRepository{ rooms, sync.RWMutex{} }
}

func (r *RoomRepository) GetOrCreate(roomId string) *room_entity.RuntimeRoom {
	room := r.rooms[roomId]
	if room == nil {
		room = room_entity.NewRuntimeRoom(roomId)
		r.rooms[roomId] = room
		ctx, cancel := context.WithCancel(context.Background())
		room.AddCancelFunc(cancel)
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					room.DispatchKeyFrame()
				}
			}
		}()
	}
	return room
}

func (r *RoomRepository) GetRoom(roomId string) (*room_entity.RuntimeRoom, error) {
	room, ok := r.rooms[roomId];if !ok {
		return nil, errors.New("room not found")
	}
	// request a keyframe every 3 seconds
	go func() {
		for range time.NewTicker(time.Second * 3).C {
			room.DispatchKeyFrame()
		}
	}()

	return room, nil
}

func (h *RoomRepository) DeleteRoom(roomId string) {
	delete(h.rooms, roomId)
}