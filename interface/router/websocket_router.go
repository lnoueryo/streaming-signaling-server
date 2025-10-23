package router

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	live_video_hub "streaming-server.com/application/ports/realtime/hubs"
)

type WSMsgHandler func(ctx context.Context, raw interface{}, c *live_video_hub.ThreadSafeWriter)

type WSRoute struct {
	handlers map[string]WSMsgHandler
	Params   map[string]string
	BeforeConnect func(w http.ResponseWriter, req *http.Request)
	OnDisconnect  func(ctx context.Context, c *live_video_hub.ThreadSafeWriter)
}

type compiledPath struct {
    any     http.Handler
    methods map[string]http.Handler
}

// イベント登録
func (w *WSRoute) On(event string, h WSMsgHandler) {
	if w.handlers == nil {
		w.handlers = make(map[string]WSMsgHandler)
	}
	w.handlers[event] = h
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (r *Router) WS(path string, setup func(ws *WSRoute)) {
	wsr := &WSRoute{
		handlers: make(map[string]WSMsgHandler),
	}
	setup(wsr)

	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var ctx context.Context
		if wsr.BeforeConnect != nil {
			wsr.BeforeConnect(w, req)
		} else {
			ctx = req.Context()
		}
		// --- 1️⃣ パスパラメータ抽出 ---
		pathParams := extractPathParams(path, req.URL.Path) // 例: {roomId: "1"}

		// --- 2️⃣ クエリパラメータ結合 ---
		params := map[string]string{}
		for k, v := range pathParams {
			params[k] = v
		}
		for k, v := range req.URL.Query() {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}

		// --- 3️⃣ Contextに格納 ---
		ctx = context.WithValue(req.Context(), "params", params)

		// --- 4️⃣ WebSocketアップグレード ---
		conn, err := wsUpgrader.Upgrade(w, req, nil)
		if err != nil {
			log.Error("❌ WebSocket upgrade failed: %v", err)
			http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
			return
		}
		c := &live_video_hub.ThreadSafeWriter{
			conn,
			sync.Mutex{},
		}

		defer func() {
			log.Info("🔌 Connection closed for %+v", params)
			if wsr.OnDisconnect != nil {
				wsr.OnDisconnect(ctx, c)
			}
			c.Close()
		}()

		// --- 5️⃣ メッセージループ ---
		for {
			_, data, err := c.ReadMessage()
			if err != nil {
				log.Warn("🔌 WS read error: %v", err)
				break
			}

			var env struct {
				Type string      `json:"type"`
				Data interface{} `json:"data"`
			}
			if err := json.Unmarshal(data, &env); err != nil {
				log.Warn("⚠️ Invalid WS message: %v", err)
				continue
			}

			h, ok := wsr.handlers[env.Type]
			if !ok {
				log.Warn("⚠️ No handler for message type: %s", env.Type)
				continue
			}

			// 🔹 handler呼び出し（ctx＋conn＋data）
			h(ctx, env.Data, c)
		}
	})

	r.addRoute(http.MethodGet, path, handler)
}

func (r *Router) addRoute(method, path string, h http.Handler) {
    log.Info("[router.go] Registered route: %s", path)

    // 動的パスを正規表現化
    regexPath := "^" + regexp.QuoteMeta(path) + "$"
    regexPath = strings.ReplaceAll(regexPath, `\{[a-zA-Z0-9_]+\}`, "([a-zA-Z0-9_-]+)")
    compiled := regexp.MustCompile(regexPath)

    r.routes = append(r.routes, Route{
        Method:  method,
        Path:    path,
        Regex:   compiled,
        Handler: h,
    })
}

func extractPathParams(routePattern, actualPath string) map[string]string {
    params := make(map[string]string)

    routeSeg := strings.Split(strings.Trim(routePattern, "/"), "/")
    pathSeg := strings.Split(strings.Trim(actualPath, "/"), "/")

    // 🧠 もし実際の方が長い場合、末尾合わせにする
    if len(pathSeg) > len(routeSeg) {
        diff := len(pathSeg) - len(routeSeg)
        pathSeg = pathSeg[diff:]
    }

    for i, seg := range routeSeg {
        if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
            key := seg[1 : len(seg)-1]
            if i < len(pathSeg) {
                params[key] = pathSeg[i]
            }
        }
    }
    return params
}