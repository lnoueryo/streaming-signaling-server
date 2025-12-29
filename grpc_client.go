package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	media "streaming-signaling.jounetsism.biz/proto/media"
)
var (
    mediaServiceClient media.MediaServiceClient
    grpcConn           *grpc.ClientConn
    onceInitRoom           sync.Once
)

func initRoomClient() {
    conn, err := grpc.Dial(
        mediaServerOrigin,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatalf("failed to dial media server: %v", err)
    }
    mediaServiceClient = media.NewMediaServiceClient(conn)
}

func CreateViewerPeer(spaceId string, user UserInfo) error {
    onceInitRoom.Do(initRoomClient)
    token := createServiceJWT()
    md := metadata.New(map[string]string{
        "authorization": "Bearer " + token,
    })
    ctx := metadata.NewOutgoingContext(context.Background(), md)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := mediaServiceClient.CreateViewerPeer(ctx, &media.CreateViewerPeerRequest{
        SpaceId: spaceId,
        User: &media.User{
            Id:    user.ID,
            Email: user.Email,
            Name:  user.Name,
            Image: user.Image,
        },
    })
    if err != nil {
        return err
    }
    return nil
}

func AddCandidate(spaceId string, user UserInfo, candidate string) error {
    onceInitRoom.Do(initRoomClient)
    token := createServiceJWT()
    md := metadata.New(map[string]string{
        "authorization": "Bearer " + token,
    })
    ctx := metadata.NewOutgoingContext(context.Background(), md)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := mediaServiceClient.AddCandidate(ctx, &media.AddCandidateRequest{
        SpaceId: spaceId,
        User: &media.User{
			Id:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Image: user.Image,
		},
		Candidate: candidate,
    })
    if err != nil {
        return err
    }

    return nil
}

func AddViewerCandidate(spaceId string, user UserInfo, candidate string) error {
    onceInitRoom.Do(initRoomClient)
    token := createServiceJWT()
    md := metadata.New(map[string]string{
        "authorization": "Bearer " + token,
    })
    ctx := metadata.NewOutgoingContext(context.Background(), md)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := mediaServiceClient.AddViewerCandidate(ctx, &media.AddCandidateRequest{
        SpaceId: spaceId,
        User: &media.User{
			Id:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Image: user.Image,
		},
		Candidate: candidate,
    })
    if err != nil {
        return err
    }

    return nil
}

func SetAnswer(spaceId string, user UserInfo, answer string) error {
    onceInitRoom.Do(initRoomClient)
    token := createServiceJWT()
    md := metadata.New(map[string]string{
        "authorization": "Bearer " + token,
    })
    ctx := metadata.NewOutgoingContext(context.Background(), md)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := mediaServiceClient.SetAnswer(ctx, &media.SetAnswerRequest{
        SpaceId: spaceId,
        User: &media.User{
			Id:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Image: user.Image,
		},
		Answer: answer,
    })
    if err != nil {
        return err
    }

    return nil
}

func SetViewerAnswer(spaceId string, user UserInfo, answer string) error {
    onceInitRoom.Do(initRoomClient)
    token := createServiceJWT()
    md := metadata.New(map[string]string{
        "authorization": "Bearer " + token,
    })
    ctx := metadata.NewOutgoingContext(context.Background(), md)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    _, err := mediaServiceClient.SetViewerAnswer(ctx, &media.SetAnswerRequest{
        SpaceId: spaceId,
        User: &media.User{
			Id:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Image: user.Image,
		},
		Answer: answer,
    })
    if err != nil {
        return err
    }

    return nil
}

func createServiceJWT() string {
	secret := []byte(os.Getenv("SERVICE_JWT_SECRET"))

	claims := jwt.RegisteredClaims{
		Issuer:    "app-server",
		Audience:  []string{"signaling-server"},
		Subject:   "media-service", // 任意（識別用）
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Minute)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(secret)
	if err != nil {
		panic(err)
	}

	return signed
}