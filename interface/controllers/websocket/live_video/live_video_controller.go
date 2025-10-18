package live_video_controller

import (
	"context"
	"github.com/gorilla/websocket"
	create_viewer_peer_connection_usecase "streaming-server.com/application/usecases/live_video/create_viewer_peer_connection"
	get_offer_usecase "streaming-server.com/application/usecases/live_video/get_offer"
	join_room_usecase "streaming-server.com/application/usecases/live_video/join_room"
	set_answer_usecase "streaming-server.com/application/usecases/live_video/set_answer"
	set_candidate_usecase "streaming-server.com/application/usecases/live_video/set_candidate"
	"streaming-server.com/infrastructure/logger"
	live_video_request "streaming-server.com/interface/controllers/websocket/live_video/request"
)


var log = logger.Log
type Controller struct {
	GetOfferUsecase        *get_offer_usecase.GetOfferUsecase
	JoinRoomUsecase        *join_room_usecase.JoinRoomUsecase
	CreateViewerPeerConnectionUsecase *create_viewer_peer_connection_usecase.CreateViewerPeerConnectionUsecase
	SetAnswerUsecase *set_answer_usecase.SetAnswerUsecase
	SetCandidateUsecase *set_candidate_usecase.SetCandidateUsecase
}

func NewLiveVideoController(
	GetOfferUsecase *get_offer_usecase.GetOfferUsecase,
	joinRoomUsecase *join_room_usecase.JoinRoomUsecase,
	createViewerPeerConnectionUsecase *create_viewer_peer_connection_usecase.CreateViewerPeerConnectionUsecase,
	setAnswerUsecase *set_answer_usecase.SetAnswerUsecase,
	setCandidateUsecase *set_candidate_usecase.SetCandidateUsecase,
) *Controller {
	return &Controller{
		GetOfferUsecase,
		joinRoomUsecase,
		createViewerPeerConnectionUsecase,
		setAnswerUsecase,
		setCandidateUsecase,
	}
}

func (c *Controller) JoinRoom(ctx context.Context, msg interface{}, conn *websocket.Conn) {
	params, err := live_video_request.JoinRoomRequest(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	c.JoinRoomUsecase.Do(params, conn)
}

func (c *Controller) CreateViewPeerConnection(ctx context.Context, msg interface{},  conn *websocket.Conn) {
	params, err := live_video_request.CreateViewerPeerConnectionRequest(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	c.CreateViewerPeerConnectionUsecase.Do(params, conn,)
}

func (c *Controller) SetAnswer(ctx context.Context, msg interface{},  conn *websocket.Conn) {
	params, message, err := live_video_request.SetAnswerRequest(ctx, msg)
	if err != nil {
		log.Error(err)
		return
	}
	c.SetAnswerUsecase.Do(params, message, conn)
}

func (c *Controller) SetCandidate(ctx context.Context, msg interface{},  conn *websocket.Conn) {
	params, message, err := live_video_request.SetCandidateRequest(ctx, msg)
	if err != nil {
		log.Error(err)
		return
	}
	c.SetCandidateUsecase.Do(params, message, conn)
}

func (c *Controller) GetOffer(
	ctx context.Context,
	msg interface{},
	conn *websocket.Conn,
) {
	params, message, err := live_video_request.GetOfferRequest(ctx, msg)
	if err != nil {
		log.Error(err)
		return
	}
	c.GetOfferUsecase.Do(params, message, conn)
}
