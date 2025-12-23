package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "streaming-signaling.jounetsism.biz/proto"
)

func GetTargetSpaceMember(roomId string, userId string) (*pb.GetTargetSpaceMemberResponse, error) {
	// ① gRPC サーバーへ接続
	conn, err := grpc.NewClient(
		"dns:///streaming-backend:50051",                     // 推奨は DNS スキーマ付き
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer conn.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

	client := pb.NewSpaceServiceClient(conn)

	// 実際のリクエスト
	resp, err := client.GetTargetSpaceMember(
		ctx,
		&pb.GetTargetSpaceMemberRequest{
			SpaceId: roomId,
			UserId:  userId,
		},
	)
	if err != nil {
		log.Fatalf("GetTargetSpaceMember failed: %v", err)
		return nil, err
	}

	// 結果を表示
	log.Printf(
		"SpaceMember: ID=%d, SpaceID=%s, UserID=%s, Email=%s, Role=%s",
		resp.GetId(),
		resp.GetSpaceId(),
		resp.GetUserId(),
		resp.GetEmail(),
		resp.GetRole(),
	)

	return resp, nil
}