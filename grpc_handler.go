package main

import (
	"context"
	"encoding/json"

	"github.com/pion/webrtc/v4"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"streaming-signaling.jounetsism.biz/proto"
)

type RoomService struct {
	signaling.UnimplementedRoomServiceServer
}

func (s *RoomService) GetRoom(
	ctx context.Context,
	req *signaling.GetRoomRequest,
) (*signaling.GetRoomResponse, error) {
	var participants []*signaling.Participant
	room, ok := rooms.getRoom(req.SpaceId)
	logrus.Infof("GetRoom gRPC called: spaceId=%s, found=%v", req.SpaceId, ok)
	if !ok {
		md := metadata.Pairs(
			"error-code", "room-not-found",
		)
		grpc.SetTrailer(ctx, md)
		return nil, status.Error(codes.NotFound, "roomが存在しません")
	}

	for _, p := range room.participants {
		if p.PC.ConnectionState().String() == "connected" {
			participants = append(participants, &signaling.Participant{
				Id:    p.ID,
				Name:  p.Name,
				Email: p.Email,
				Image: p.Image,
			})
		}
	}

	return &signaling.GetRoomResponse{
		Id:           room.ID,
		Participants: participants,
	}, nil
}

func (s *RoomService) RemoveParticipant(
	ctx context.Context,
	req *signaling.RemoveParticipantRequest,
) (*signaling.GetRoomResponse, error) {
	roomId := req.GetSpaceId()
	userId := req.GetUserId()

	// ルームが存在しない場合
	room, ok := rooms.getRoom(roomId)
	if !ok {
		return nil, status.Error(codes.NotFound, "no-target-room")
	}

	// 参加者が存在しない場合
	participant, ok := room.participants[userId]
	if !ok {
		return nil, status.Error(codes.NotFound, "no-target-user")
	}

	// 切断処理
	room.listLock.Lock()
	delete(room.participants, userId)
	delete(room.wsConnections, participant.WS.Conn)

	// WebRTC のクローズ
	participant.PC.Close()
	participant.WS.Send("close", "")

	room.listLock.Unlock()

	// 残った参加者を整形
	responseParticipants := make([]*signaling.Participant, 0, len(room.participants))
	for _, p := range room.participants {
		if p.PC.ConnectionState() == webrtc.PeerConnectionStateConnected {
			responseParticipants = append(responseParticipants, &signaling.Participant{
				Id:    p.ID,
				Name:  p.Name,
				Email: p.Email,
				Image: p.Image,
			})
		}
	}

	return &signaling.GetRoomResponse{
		Id:           room.ID,
		Participants: responseParticipants,
	}, nil
}

func (s *RoomService) RequestEntry(
	ctx context.Context,
	req *signaling.SpaceMemberRequest,
) (*signaling.Void, error) {
	room, ok := rooms.getRoom(req.SpaceId)
	if !ok {
		md := metadata.Pairs(
			"error-code", "room-not-found",
		)
		grpc.SetTrailer(ctx, md)
		return nil, status.Error(codes.NotFound, "roomが存在しません")
	}

	for _, p := range room.participants {
		if p.Role == "owner" {
			res, _ := json.Marshal(req.SpaceMember)
			p.WS.Send("participant-request", string(res))
		}
	}

	return &signaling.Void{}, nil
}

func (s *RoomService) DecideRequest(
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

func (s *RoomService) AcceptInvitation(
	ctx context.Context,
	req *signaling.SpaceMemberRequest,
) (*signaling.Void, error) {
	room, ok := rooms.getRoom(req.SpaceId)
	if !ok {
		md := metadata.Pairs(
			"error-code", "room-not-found",
		)
		grpc.SetTrailer(ctx, md)
		return nil, status.Error(codes.NotFound, "roomが存在しません")
	}

	for _, p := range room.participants {
		if p.Role == "owner" {
			res, _ := json.Marshal(req.SpaceMember)
			p.WS.Send("accept-invitation", string(res))
		}
	}

	return &signaling.Void{}, nil
}