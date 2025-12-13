package main

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"
)


func websocketHandler(c *gin.Context) {
    user := getUser(c)
    roomId := c.Param("roomId")
    room := rooms.getOrCreate(roomId)

    // Upgrade HTTP → WebSocket
    unsafeConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Errorf("upgrade failed: %v", err)
        return
    }

    ws := &ThreadSafeWriter{unsafeConn, sync.Mutex{}}

    // --- register WS ---
    room.listLock.Lock()
    room.wsConnections[ws.Conn] = ws
    room.listLock.Unlock()

    var peerConnection *webrtc.PeerConnection

    // ---- clean up ----
    defer func() {
        room.listLock.Lock()
        // 複数のデバイスでロビーに入室し、
        // 1つだけルームにいる状態でロビーのどれかのデバイスがリロードすると
        // ルームの方のPeerConnectionも切れてしまうため、peerConnectionがある時のみクローズ
        if peerConnection != nil {
            peerConnection.Close()
            delete(room.participants, user.ID)
        }
        delete(room.wsConnections, ws.Conn)
        room.listLock.Unlock()

        ws.Close()
        rooms.cleanupEmptyRoom(roomId)
    }()

    msg := &WebsocketMessage{}

    for {
        // ---------- read WS message ----------
        _, raw, err := ws.ReadMessage()
        if err != nil {
            log.Errorf("WS read error: %v", err)
            return
        }

        if err := json.Unmarshal(raw, msg); err != nil {
            log.Errorf("json unmarshal error: %v", err)
            return
        }

        switch msg.Event {

        // =====================================================
        //                      OFFER
        // =====================================================
        case "offer":
            peerConnection, err = webrtc.NewPeerConnection(webrtc.Configuration{})
            if err != nil {
                log.Errorf("pc create error: %v", err)
                return
            }

            // Recvonly transceivers
            for _, typ := range []webrtc.RTPCodecType{
                webrtc.RTPCodecTypeVideo,
                webrtc.RTPCodecTypeAudio,
            } {
                if _, err := peerConnection.AddTransceiverFromKind(
                    typ,
                    webrtc.RTPTransceiverInit{Direction: webrtc.RTPTransceiverDirectionRecvonly},
                ); err != nil {
                    log.Errorf("add transceiver error: %v", err)
                    return
                }
            }

            // ----- register participant -----
            room.listLock.Lock()
            room.participants[user.ID] = &Participant{
                user,
                ws,
                peerConnection,
            }
            room.listLock.Unlock()

			peerConnection.OnConnectionStateChange(func(p webrtc.PeerConnectionState) {
				log.Info("Connection state change: %s", p)

				switch p {
				case webrtc.PeerConnectionStateConnected:
					room, ok := rooms.getRoom(roomId);if !ok {
						return
					}
					users := make([]UserInfo, 0)
					for _, participant := range room.participants {
						users = append(users, UserInfo{
							ID: participant.ID,
							Name: participant.Name,
							Email: participant.Email,
							Image: participant.Image,
						})
					}
					res, _ := json.Marshal(users)
					for _, conn := range room.wsConnections {
						conn.Send("access", string(res))
					}
				case webrtc.PeerConnectionStateFailed:
					_ = peerConnection.Close()
				case webrtc.PeerConnectionStateDisconnected:
					// 猶予を与える
					go func() {
						time.Sleep(20 * time.Second)
						if peerConnection.ConnectionState() == webrtc.PeerConnectionStateDisconnected {
							_ = peerConnection.Close()
						}
					}()
				case webrtc.PeerConnectionStateClosed:
					room, ok := rooms.getRoom(roomId);if !ok {
						return
					}
					room.listLock.Lock()
					delete(room.participants, user.ID)
					room.listLock.Unlock()
					users := make([]UserInfo, 0)
					for _, participant := range room.participants {
						users = append(users, UserInfo{
							ID: participant.ID,
							Name: participant.Name,
							Email: participant.Email,
							Image: participant.Image,
						})
					}
					res, _ := json.Marshal(users)
					for _, conn := range room.wsConnections {
						conn.Send("access", string(res))
					}
				}
			})

            // ----- track handler -----
            peerConnection.OnTrack(func(t *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
                log.Infof("Got remote track: %s %s", t.Kind(), t.ID())

                trackLocal := addTrack(room, t)
                if trackLocal == nil {
                    return
                }
                defer removeTrack(room, trackLocal)

                buf := make([]byte, 1500)
                pkt := &rtp.Packet{}

                for {
                    n, _, err := t.Read(buf)
                    if err != nil {
                        return
                    }

                    if pkt.Unmarshal(buf[:n]) != nil {
                        continue
                    }

                    pkt.Extension = false
                    pkt.Extensions = nil
                    trackLocal.WriteRTP(pkt)
                }
            })

            // ----- set remote offer -----
            var offer webrtc.SessionDescription
            json.Unmarshal([]byte(msg.Data), &offer)
            peerConnection.SetRemoteDescription(offer)

            // ----- create answer -----
            answer, _ := peerConnection.CreateAnswer(nil)
            peerConnection.SetLocalDescription(answer)

            res, _ := json.Marshal(answer)
            ws.WriteJSON(&WebsocketMessage{
                Event: "answer",
                Data:  string(res),
            })

            // renegotiate others
            signalPeerConnections(room)

        // =====================================================
        //                  CANDIDATE
        // =====================================================
        case "candidate":
            var cand webrtc.ICECandidateInit
            json.Unmarshal([]byte(msg.Data), &cand)
            if err := peerConnection.AddICECandidate(cand); err != nil {
                log.Errorf("ice add error: %v", err)
            }

        // =====================================================
        //                    ANSWER
        // =====================================================
        case "answer":
            var ans webrtc.SessionDescription
            json.Unmarshal([]byte(msg.Data), &ans)
            peerConnection.SetRemoteDescription(ans)
        }
    }
}