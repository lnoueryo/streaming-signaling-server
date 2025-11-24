package main

import (
	close_connection_usecase "streaming-server.com/application/usecases/live_video/close_connection"
	create_live_video_usecase "streaming-server.com/application/usecases/live_video/create_live_video"
	create_viewer_peer_connection_usecase "streaming-server.com/application/usecases/live_video/create_viewer_peer_connection"
	get_offer_usecase "streaming-server.com/application/usecases/live_video/get_offer"
	set_answer_usecase "streaming-server.com/application/usecases/live_video/set_answer"
	set_candidate_usecase "streaming-server.com/application/usecases/live_video/set_candidate"
	"streaming-server.com/infrastructure/logger"
	"streaming-server.com/infrastructure/server"

	"streaming-server.com/infrastructure/router"
	"streaming-server.com/interface/controllers"
	websocket_controller "streaming-server.com/interface/controllers/http/websocket"
	live_video_controller "streaming-server.com/interface/controllers/websocket/live_video"
	room_memory_repository "streaming-server.com/interface/repositories/memory"
)

func main() {
	roomRepository := room_memory_repository.NewRoomRepository()
	createLiveVideoUseccase := create_live_video_usecase.NewCreateLiveVideo()
	getOfferUsecase := get_offer_usecase.NewGetOffer(roomRepository)
	createViewerPeerConnectionUsecase := create_viewer_peer_connection_usecase.NewCreateViewerPeerConnection(roomRepository)
	setAnswerUsecase := set_answer_usecase.NewSetAnswer(roomRepository)
	setCandidateUsecase := set_candidate_usecase.NewSetCandidate(roomRepository)
	closeConnectionUsecase := close_connection_usecase.NewCloseConnection(roomRepository)
	liveVideoController := live_video_controller.NewLiveVideoController(
		getOfferUsecase,
		createViewerPeerConnectionUsecase,
		setAnswerUsecase,
		setCandidateUsecase,
		closeConnectionUsecase,
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
