package live_video_controller

import (
	"context"

	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	close_connection_usecase "streaming-server.com/application/usecases/live_video/close_connection"
	create_viewer_peer_connection_usecase "streaming-server.com/application/usecases/live_video/create_viewer_peer_connection"
	get_offer_usecase "streaming-server.com/application/usecases/live_video/get_offer"
	set_answer_usecase "streaming-server.com/application/usecases/live_video/set_answer"
	set_candidate_usecase "streaming-server.com/application/usecases/live_video/set_candidate"
	"streaming-server.com/infrastructure/logger"
	live_video_request "streaming-server.com/interface/controllers/websocket/live_video/request"
)


var log = logger.Log
type Controller struct {
	GetOfferUsecase        *get_offer_usecase.GetOfferUsecase
	CreateViewerPeerConnectionUsecase *create_viewer_peer_connection_usecase.CreateViewerPeerConnectionUsecase
	SetAnswerUsecase *set_answer_usecase.SetAnswerUsecase
	SetCandidateUsecase *set_candidate_usecase.SetCandidateUsecase
	CloseConnectionUsecase *close_connection_usecase.CloseConnectionUsecase
}

func NewLiveVideoController(
	GetOfferUsecase *get_offer_usecase.GetOfferUsecase,
	createViewerPeerConnectionUsecase *create_viewer_peer_connection_usecase.CreateViewerPeerConnectionUsecase,
	setAnswerUsecase *set_answer_usecase.SetAnswerUsecase,
	setCandidateUsecase *set_candidate_usecase.SetCandidateUsecase,
	closeConnectionUsecase *close_connection_usecase.CloseConnectionUsecase,
) *Controller {
	return &Controller{
		GetOfferUsecase,
		createViewerPeerConnectionUsecase,
		setAnswerUsecase,
		setCandidateUsecase,
		closeConnectionUsecase,
	}
}

func (c *Controller) CreateViewPeerConnection(ctx context.Context, msg interface{},  conn *live_video_hub.ThreadSafeWriter) {
	params, err := live_video_request.CreateViewerPeerConnectionRequest(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	c.CreateViewerPeerConnectionUsecase.Do(params, conn,)
}

func (c *Controller) SetAnswer(ctx context.Context, msg interface{},  conn *live_video_hub.ThreadSafeWriter) {
	params, message, err := live_video_request.SetAnswerRequest(ctx, msg)
	if err != nil {
		log.Error(err)
		return
	}
	c.SetAnswerUsecase.Do(params, message, conn)
}

func (c *Controller) SetCandidate(ctx context.Context, msg interface{},  conn *live_video_hub.ThreadSafeWriter) {
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
	conn *live_video_hub.ThreadSafeWriter,
) {
	params, message, err := live_video_request.GetOfferRequest(ctx, msg)
	if err != nil {
		log.Error(err)
		return
	}
	c.GetOfferUsecase.Do(params, message, conn)
}

func (c *Controller) CloseConnection(
	ctx context.Context,
	conn *live_video_hub.ThreadSafeWriter,
) {
	params, err := live_video_request.CloseConnectionRequest(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	c.CloseConnectionUsecase.Do(params, conn)
}
