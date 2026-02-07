package entities

import (
	"github.com/google/uuid"
)

type WSClient struct {
    ID      string
    Email string
    Name  string
    Image  string
    LobbyID string
    ConnID string
}

func NewWSClient(id string, email string, name string, image string, lobbyId string) *WSClient {
	uid, err := uuid.NewV7();if err != nil {
		return nil
	}
    return &WSClient{
        ID:      id,
        Email: email,
        Name:  name,
        Image:  image,
        LobbyID: lobbyId,
        ConnID: uid.String(),
    }
}   