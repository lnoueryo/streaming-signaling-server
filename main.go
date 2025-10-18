package main

import (
	create_live_video_usecase "streaming-server.com/application/usecases/live_video/create_live_video"
	create_viewer_peer_connection_usecase "streaming-server.com/application/usecases/live_video/create_viewer_peer_connection"
	get_offer_usecase "streaming-server.com/application/usecases/live_video/get_offer"
	join_room_usecase "streaming-server.com/application/usecases/live_video/join_room"
	set_answer_usecase "streaming-server.com/application/usecases/live_video/set_answer"
	set_candidate_usecase "streaming-server.com/application/usecases/live_video/set_candidate"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/repository/memory/rooms"
	"streaming-server.com/infrastructure/server"

	"streaming-server.com/interface/controllers"
	websocket_controller "streaming-server.com/interface/controllers/http/websocket"
	live_video_controller "streaming-server.com/interface/controllers/websocket/live_video"
	"streaming-server.com/interface/router"
)

func main() {
	roomRepository := rooms_hub.New()
	createLiveVideoUseccase := create_live_video_usecase.NewCreateLiveVideo()
	joinRoomUsecase := join_room_usecase.NewJoinRoom(roomRepository)
	getOfferUsecase := get_offer_usecase.NewGetOffer(roomRepository)
	createViewerPeerConnectionUsecase := create_viewer_peer_connection_usecase.NewCreateViewerPeerConnection(roomRepository)
	setAnswerUsecase := set_answer_usecase.NewSetAnswer(roomRepository)
	setCandidateUsecase := set_candidate_usecase.NewSetCandidate(roomRepository)
	liveVideoController := live_video_controller.NewLiveVideoController(
		getOfferUsecase,
		joinRoomUsecase,
		createViewerPeerConnectionUsecase,
		setAnswerUsecase,
		setCandidateUsecase,
	)
	websocketController := websocket_controller.NewController(createLiveVideoUseccase)
	controllers := controllers.NewControllers(
		liveVideoController,
		websocketController,
	)
	muxOrHandler := router.CreateHandler(controllers) // ← ここだけ変更（*ServeMux でなく http.Handler）
	srv := server.NewHTTPServer(muxOrHandler)

	logger.Log.Info("✅ Server listening on :8080")
	logger.Log.Error("%v", srv.ListenAndServe())
}
