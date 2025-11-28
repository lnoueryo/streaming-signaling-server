package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
)

type Room struct {
	ID string
	listLock sync.RWMutex
	clients map[int]*RTCSession
	trackLocals map[string]*webrtc.TrackLocalStaticRTP
	cancelFunc context.CancelFunc
}

func addTrack(id string, t *webrtc.TrackRemote) *webrtc.TrackLocalStaticRTP {
	room, err := rooms.getRoom(id);if err != nil {
		return nil
	}
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		signalPeerConnections(id)
	}()
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(t.Codec().RTPCodecCapability, t.ID(), t.StreamID())
	if err != nil {
		panic(err)
	}
	room.trackLocals[t.ID()] = trackLocal
	return trackLocal
}

// Remove from list of tracks and fire renegotation for all PeerConnections.
func removeTrack(id string, t *webrtc.TrackLocalStaticRTP) {
	room, err := rooms.getRoom(id);if err != nil {
		return
	}
	room.listLock.Lock()
	defer func() {
		room.listLock.Unlock()
		signalPeerConnections(id)
	}()

	delete(room.trackLocals, t.ID())
}

// dispatchKeyFrame sends a keyframe to all PeerConnections, used everytime a new user joins the call.
func dispatchKeyFrame(id string) {
    room, err := rooms.getRoom(id)
    if err != nil { return }

    // 収集だけロック下で
    type target struct{ pc *webrtc.PeerConnection; ssrc uint32 }
    var targets []target

    room.listLock.RLock()
    for _, c := range room.clients {
        for _, recv := range c.Peer.GetReceivers() {
            if tr := recv.Track(); tr != nil {
                targets = append(targets, target{pc: c.Peer, ssrc: uint32(tr.SSRC())})
            }
        }
    }
    room.listLock.RUnlock()

    // ロック外でRTCP送信
    for _, t := range targets {
        _ = t.pc.WriteRTCP([]rtcp.Packet{
            &rtcp.PictureLossIndication{MediaSSRC: t.ssrc},
        })
    }
}

// signalPeerConnections updates each PeerConnection so that it is getting all the expected media tracks.
func signalPeerConnections(id string) {
    room, err := rooms.getRoom(id)
    if err != nil { return }

    type applyFn func() bool // true=要リトライ
    var ops []applyFn
    var emptyAfter bool

    // 1) ロック内で差分計算のみ
    room.listLock.Lock()

    for uid, cli := range room.clients {
        if cli.Peer.ConnectionState() == webrtc.PeerConnectionStateClosed {
            // mapから削除（ロック下でOK）
            delete(room.clients, uid)
        }
    }
    emptyAfter = (len(room.clients) == 0)

    // 送受信の差分を計算して ops に積む
    for _, cli := range room.clients {
        pc := cli.Peer

        // 既存 sender/receiver をスキャン（IDだけ集める）
        existing := map[string]bool{}
        for _, s := range pc.GetSenders() {
            if tr := s.Track(); tr != nil { existing[tr.ID()] = true }
        }
        for _, rcv := range pc.GetReceivers() {
            if tr := rcv.Track(); tr != nil { existing[tr.ID()] = true }
        }

        // Remove 対象
        var toRemove []*webrtc.RTPSender
        for _, s := range pc.GetSenders() {
            if tr := s.Track(); tr != nil {
                if _, ok := room.trackLocals[tr.ID()]; !ok {
                    toRemove = append(toRemove, s)
                }
            }
        }
        // Add 対象
        var toAdd []*webrtc.TrackLocalStaticRTP
        for id, tl := range room.trackLocals {
            if !existing[id] {
                toAdd = append(toAdd, tl)
            }
        }

        ws := cli.WS // WS参照

        // 2) PC/WSの操作はロック外で走らせる関数として用意
		ops = append(ops, func() bool {
			cli.sigMu.Lock()
			defer cli.sigMu.Unlock()

			if cli.makingOffer {
				// いま別のトリガで offer 中 → 後で一度だけやり直す印を付けて終わり
				cli.needRenego = true
				return false
			}
			cli.makingOffer = true

			// Remove / Add はそのまま
			for _, s := range toRemove {
				if err := pc.RemoveTrack(s); err != nil { /* ログ */ }
			}
			for _, tl := range toAdd {
				if _, err := pc.AddTrack(tl); err != nil { /* ログ */ }
			}

			offer, err := pc.CreateOffer(nil)
			if err != nil { cli.makingOffer = false; return true }
			if err = pc.SetLocalDescription(offer); err != nil { cli.makingOffer = false; return true }
			if err = ws.Send("offer", offer); err != nil { cli.makingOffer = false; return true }

			// ここでは makingOffer=true のまま。answer 受領で解除する
			return false
		})
    }

    room.listLock.Unlock()

    // 3) ルームが空になったら、rooms をロックして削除（mapは常にrooms.lockで）
    if emptyAfter {
        rooms.deleteRoom(id)
        return
    }

    // 4) ロック外でPC/WS操作を適用。失敗したら短時間後に再試行
    retry := false
    for _, f := range ops {
        if f() { retry = true }
    }
    if retry {
        time.AfterFunc(3*time.Second, func(){ signalPeerConnections(id) })
    }

    // 5) 最後にロック外で KeyFrame 要求
    dispatchKeyFrame(id)
}

type Rooms struct {
	item map[string]*Room
	lock sync.RWMutex
}

func (r *Rooms) getOrCreate(id string) *Room {
    r.lock.Lock()
    defer r.lock.Unlock()

    room := r.item[id]
    if room == nil {
        room = &Room{
            ID:          id,
            listLock:    sync.RWMutex{},
            clients:     make(map[int]*RTCSession),
            trackLocals: make(map[string]*webrtc.TrackLocalStaticRTP),
        }
        r.item[id] = room

        // ticker は作成時に一度だけ
        ctx, cancel := context.WithCancel(context.Background())
        room.cancelFunc = cancel
        go func() {
            t := time.NewTicker(3 * time.Second)
            defer t.Stop()
            for {
                select {
                case <-ctx.Done():
                    return
                case <-t.C:
                    dispatchKeyFrame(id) // ← この中も lock 外でPC操作するよう修正(後述)
                }
            }
        }()
    }
    return room
}

func (r *Rooms) getRoom(id string) (*Room, error) {
    r.lock.RLock()
    room, ok := r.item[id]
    r.lock.RUnlock()
    if !ok {
        return nil, errors.New("room not found")
    }
    return room, nil
}

func (r *Rooms) deleteRoom(id string) {
    r.lock.Lock()
    room, ok := r.item[id]
    if ok {
        if room.cancelFunc != nil { room.cancelFunc() }
        delete(r.item, id)
    }
    r.lock.Unlock()
}