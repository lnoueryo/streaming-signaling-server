package create_viewer_peer_connection_usecase

import (
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
	live_video_dto "streaming-server.com/application/usecases/live_video/dto"
	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type CreateViewerPeerConnectionUsecase struct {
	roomRepository live_video_hub.Interface
}

func NewCreateViewerPeerConnection(roomRepo live_video_hub.Interface) *CreateViewerPeerConnectionUsecase {
	return &CreateViewerPeerConnectionUsecase{
		roomRepo,
	}
}

func (u *CreateViewerPeerConnectionUsecase) Do(
	params *live_video_dto.Params,
	conn *websocket.Conn,
) error {
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		log.Error("pc:", err)
		return err
	}
	u.roomRepository.AddPeerConnection(params.RoomID, params.UserID, pc)

	// 受信専用の transceiver を先に用意（あとから track が来ても受けられる）
	_, _ = pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	)
	_, _ = pc.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
	)

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		log.Info("📥 Request: OnICECandidate")
		if c == nil {
			log.Info("ICE gathering complete")
			return
		}
		message := struct {
			Type string                   `json:"type"`
			Data webrtc.ICECandidateInit `json:"data"`
		}{
			Type: "candidate",
			Data: c.ToJSON(),
		}
		if err := conn.WriteJSON(message); err != nil {
			log.Error("send candidate error:", err)
		}
		log.Info("👌 Send: Candidate to Viewer")
	})
	u.roomRepository.AddPublisherTracks(params.RoomID, params.UserID, pc)

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		log.Error("createOffer viewer:", err)
		return err
	}
	g := webrtc.GatheringCompletePromise(pc)
	if err := pc.SetLocalDescription(offer); err != nil {
		log.Error("setLocal viewer:", err)
		return err
	}
	<-g
	message := struct {
		Type string `json:"type"`
		Data struct {
			RoomID int    `json:"roomId"`
			SDP    string `json:"sdp"`
		} `json:"data"`
	}{
		Type: "offer",
		Data: struct {
			RoomID int    `json:"roomId"`
			SDP    string `json:"sdp"`
		}{
			RoomID: params.RoomID,
			SDP:    offer.SDP,
		},
	}
	if err := conn.WriteJSON(message); err != nil {
		log.Error("send offer to viewer:", err)
	}
	log.Info("👌 Send: Offer To Viewer")
	return nil
}

//　Androidリセット→送信→ブラウザリロード
//　現状表示される
// websocketのメッセージをドメインに記述
//　ログを注入
