package repository_interfaces

import "streaming-signaling.jounetsism.biz/src/domain/entities"

type ILobbyRepository interface{
	GetOrCreate(id string) entities.Lobby
	DeleteIfEmptity(id string)
	Lock()
	Unlock()
}