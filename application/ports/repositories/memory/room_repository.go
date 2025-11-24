package room_memory_repository

import (
    "streaming-server.com/domain/entities/room"
)

type IRoomRepository interface {
    GetOrCreate(string) *room_entity.RuntimeRoom
    GetRoom(string) (*room_entity.RuntimeRoom, error)
    DeleteRoom(string)
}