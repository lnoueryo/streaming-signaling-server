package main

import (
	"context"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	signaling "streaming-signaling.jounetsism.biz/proto/signaling"
)

type SignalingService struct {
	signaling.UnimplementedSignalingServiceServer
}

func (s *SignalingService) DecideRequest(
	ctx context.Context,
	req *signaling.SpaceMember,
) (*signaling.Void, error) {
	room, ok := rooms.getRoom(req.SpaceId)
	if !ok {
		md := metadata.Pairs(
			"error-code", "room-not-found",
		)
		grpc.SetTrailer(ctx, md)
		return nil, status.Error(codes.NotFound, "roomが存在しません")
	}

	for _, c := range room.wsConnections {
		if c.UserID == req.UserId {
			res, _ := json.Marshal(req)
			c.Send("request-decision", string(res))
		}
	}

	return &signaling.Void{}, nil
}

func (s *SignalingService) BroadcastToLobby(
    ctx context.Context,
    req *signaling.BroadcastRequest,
) (*signaling.Void, error) {
	room, ok := rooms.getRoom(req.SpaceId);if !ok {
		return nil, status.Error(codes.NotFound, "room not found")
	}

	for _, c := range room.wsConnections {
		c.Send(req.Event, string(req.Data))
	}

	return &signaling.Void{}, nil
}

func (s *SignalingService) Unicast(
	ctx context.Context,
	req *signaling.UnicastRequest,
) (*signaling.Void, error) {
	logrus.Error("Unicast %s to %s", req.Event, req.GetUserId())
	room, ok := rooms.getRoom(req.SpaceId);if !ok {
		return nil, status.Error(codes.NotFound, "room not found")
	}

	for _, p := range room.wsConnections {
		if p.UserID == req.GetUserId() {
			p.Send(req.Event, string(req.Data))
		}
	}
    return &signaling.Void{}, nil
}