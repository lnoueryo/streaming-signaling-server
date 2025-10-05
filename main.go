package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Room struct {
	mu     sync.Mutex
	tracks []*webrtc.TrackLocalStaticRTP
	peers  []*webrtc.PeerConnection
}

var rooms = map[string]*Room{}
var roomMu sync.Mutex

type Client struct {
    conn *websocket.Conn
    pc   *webrtc.PeerConnection
}

var clients = sync.Map{} // conn → Client を保存

func getRoom() *Room {
	roomMu.Lock()
	defer roomMu.Unlock()
	roomID := "room1"
	if _, ok := rooms[roomID]; !ok {
		rooms[roomID] = &Room{}
	}
	return rooms[roomID]
}

// WebSocket handler
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer conn.Close()

	var msg struct {
		Type          string `json:"type"`
		SDP           string `json:"sdp"`
		Candidate     string `json:"candidate"`
		SDPMid        *string `json:"sdpMid"`
		SDPMLineIndex *uint16 `json:"sdpMLineIndex"`
	}
	for {
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("read:", err)
			return
		}

		switch msg.Type {
		case "offer":
			pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{URLs: []string{"stun:stun.l.google.com:19302"}},
				},
			})
			if err != nil {
				log.Println("pc:", err)
				return
			}
			clients.Store(conn, &Client{conn: conn, pc: pc})
			room := getRoom()

			// 受信したトラックを他のピアに転送
			pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
				log.Println("📡 Track received:", track.Kind().String())

				// 新しい LocalTrack を作成して room に保存
				localTrack, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.Kind().String(), "pion")
				if err != nil {
					log.Println("NewTrackLocalStaticRTP error:", err)
					return
				}

				room.mu.Lock()
				room.tracks = append(room.tracks, localTrack)
				room.mu.Unlock()

				// リモートから読み取って localTrack に書き込む
				go func() {
					buf := make([]byte, 1500)
					for {
						n, _, err := track.Read(buf)
						if err != nil {
							break
						}
						if _, err = localTrack.Write(buf[:n]); err != nil {
							break
						}
					}
				}()
			})

			pc.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c == nil {
					return
				}
				candidate := map[string]interface{}{
					"type":      "candidate",
					"candidate": c.ToJSON().Candidate,
					"sdpMid":    c.ToJSON().SDPMid,
					"sdpMLineIndex": c.ToJSON().SDPMLineIndex,
				}
				if err := conn.WriteJSON(candidate); err != nil {
					log.Println("send candidate:", err)
				}
			})

			offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: msg.SDP}
			if err := pc.SetRemoteDescription(offer); err != nil {
				log.Println("setRemote:", err)
				return
			}
			answer, err := pc.CreateAnswer(nil)
			if err != nil {
				log.Println("createAnswer:", err)
				return
			}
			g := webrtc.GatheringCompletePromise(pc)
			pc.SetLocalDescription(answer)
			<-g // ★ ICE候補の埋め込み完了待ち

			room.mu.Lock()
			room.peers = append(room.peers, pc)
			room.mu.Unlock()

			resp := map[string]string{"type": "answer", "sdp": answer.SDP}
			conn.WriteJSON(resp)

		case "answer":
			val, ok := clients.Load(conn)
			if !ok {
				log.Println("no pc for this conn")
				return
			}
			client := val.(*Client)

			if err := client.pc.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  msg.SDP,
			}); err != nil {
				log.Println("setRemote answer:", err)
			}

		case "candidate":
			val, ok := clients.Load(conn)
			if !ok { log.Println("no pc for this conn"); break }
			client := val.(*Client)

			cand := webrtc.ICECandidateInit{
				Candidate:     msg.Candidate,
				SDPMid:        msg.SDPMid,
				SDPMLineIndex: msg.SDPMLineIndex,
			}
			if err := client.pc.AddICECandidate(cand); err != nil {
				log.Println("AddICECandidate:", err)
			}

		case "viewer":
			pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
				ICEServers: []webrtc.ICEServer{
					{URLs: []string{"stun:stun.l.google.com:19302"}},
				},
			})
			if err != nil {
				log.Println("pc:", err)
				return
			}

			// Viewerのpcを保存
			clients.Store(conn, &Client{conn: conn, pc: pc})

			room := getRoom()

			// ICE candidateを送信
			pc.OnICECandidate(func(c *webrtc.ICECandidate) {
				if c != nil {
					conn.WriteJSON(c.ToJSON())
				}
			})
			pc.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo)
			pc.AddTransceiverFromKind(webrtc.RTPCodecTypeAudio)

			// ルーム内のトラックをこの視聴者に追加
			room.mu.Lock()
			for _, t := range room.tracks {
				_, err := pc.AddTrack(t)
				if err != nil {
					log.Println("AddTrack:", err)
				}
			}
			room.mu.Unlock()

			// --- サーバーがOfferを生成してViewerに送る ---
			offer, err := pc.CreateOffer(nil)
			if err != nil {
				log.Println("createOffer viewer:", err)
				return
			}
			g := webrtc.GatheringCompletePromise(pc)
			if err := pc.SetLocalDescription(offer); err != nil {
				log.Println("setLocal viewer:", err)
				return
			}
			<-g
			// PeerConnectionを保存
			room.mu.Lock()
			room.peers = append(room.peers, pc)
			room.mu.Unlock()

			// Viewerに送信
			resp := map[string]string{"type": "offer", "sdp": offer.SDP}
			conn.WriteJSON(resp)
		}
	}
}

func main() {
	http.HandleFunc("/ws", handleWS)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	fmt.Println("✅ WebRTC SFU server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
