package router

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"streaming-server.com/interface/controllers"
	http_controllers "streaming-server.com/interface/controllers/http"
	middleware "streaming-server.com/interface/middlewares"
)

func (r *Router) registerHttpRoutes(ctrls *controllers.Controllers) {
	baseDir, _ := os.Getwd()
	staticDir := filepath.Join(baseDir, "interface", "static")

	// 静的配信 /static/*
	r.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// トップ
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
	})

	r.Group([]Adapter{middleware.AuthMiddleware}, func(g *Router) {
		g.Get("/dashboard", func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("Dashboard (auth)"))
		})
		g.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte("Settings (auth)"))
		})
	})

	r.GroupWithPrefix("/auth", []Adapter{middleware.AuthMiddleware}, func(g *Router) {
		g.Get("/broadcast", func(w http.ResponseWriter, req *http.Request) {
			http.ServeFile(w, req, filepath.Join(staticDir, "broadcast.html"))
		})
	})

	// 動作確認
	r.Get("/hello", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello world"))
	})

	// API メソッド例
	r.Post("/api/echo", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("posted"))
	})
	r.Put("/api/item", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("put"))
	})
	r.Delete("/api/item", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("deleted"))
	})
	r.Patch("/api/item", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("patched"))
	})


    r.GroupWithPrefix("/ws", []Adapter{middleware.AuthMiddleware}, func(g *Router) {
        g.WS("/live/{roomId}/{userId}", func(ws *WSRoute) {
            ws.BeforeConnect = ctrls.WebsocketController.CreateLiveVideo
            ws.OnDisconnect  = ctrls.LiveVideoController.CloseConnection
            ws.On("offer", ctrls.LiveVideoController.GetOffer)
            ws.On("candidate", ctrls.LiveVideoController.SetCandidate)
            ws.On("answer", ctrls.LiveVideoController.SetAnswer)
            ws.On("viewer", ctrls.LiveVideoController.CreateViewPeerConnection)
        })
    })
}

func CreateHandler(ctrls *controllers.Controllers) http.Handler {
    r := NewRouter()
    r.registerHttpRoutes(ctrls)

    // グローバルMW（ログなど）
    r.SetMiddleware(middleware.LoggingMiddleware)
	return r.setRoutes()
}

func (r *Router) setRoutes() http.Handler {
    // 1) まずパスごとにメソッド別ハンドラを集約
    paths := make(map[string]*compiledPath)
    for _, rt := range r.routes {
        // ルート固有ミドルウェアをハンドラに適用
        h := rt.Handler
        for i := len(rt.Middlewares) - 1; i >= 0; i-- {
            h = rt.Middlewares[i](h)
        }

        cp := paths[rt.Path]
        if cp == nil {
            cp = &compiledPath{methods: map[string]http.Handler{}}
            paths[rt.Path] = cp
        }
        if rt.Method == "" {
            cp.any = h
        } else {
            cp.methods[rt.Method] = h
        }
    }

    // Strict 用のルート一覧
    exacts := make(map[string]struct{})
    prefixes := make([]string, 0)

    // 2) パスごとに1回だけ ServeMux へ登録（中でメソッドディスパッチ）
    for path, cp := range paths {
        // メソッドディスパッチャ
        dispatcher := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
            ctx := context.WithValue(req.Context(), "params", pathParams)
            req = req.WithContext(ctx)
            ctx = context.WithValue(req.Context(), "request", http_controllers.NewRequest(req))
            req = req.WithContext(ctx)
            // 明示メソッド優先
            if h, ok := cp.methods[req.Method]; ok {
                h.ServeHTTP(w, req)
                return
            }
            // HEAD は GET にフォールバック
            if req.Method == http.MethodHead {
                if h, ok := cp.methods[http.MethodGet]; ok {
                    h.ServeHTTP(w, req)
                    return
                }
            }
            // 汎用ハンドラ（Handle/HandleFunc）にフォールバック
            if cp.any != nil {
                cp.any.ServeHTTP(w, req)
                return
            }

            // 405 Method Not Allowed
            allow := make([]string, 0, len(cp.methods))
            for m := range cp.methods {
                allow = append(allow, m)
            }
            if _, hasGet := cp.methods[http.MethodGet]; hasGet {
                // GET があれば HEAD も許可
                foundHead := false
                for _, m := range allow {
                    if m == http.MethodHead {
                        foundHead = true
                        break
                    }
                }
                if !foundHead {
                    allow = append(allow, http.MethodHead)
                }
            }
            sort.Strings(allow)
            w.Header().Set("Allow", strings.Join(allow, ", "))
            http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
        })

        // グローバルMWはパス単位で適用
        var h http.Handler = dispatcher
        for i := len(r.globalAdapters) - 1; i >= 0; i-- {
            h = r.globalAdapters[i](h)
        }
        log.Info(path)
        r.mux.Handle(path, h)

        // Strict 404 用
        if strings.HasSuffix(path, "/") && path != "/" {
            prefixes = append(prefixes, path)
        } else {
            exacts[path] = struct{}{}
        }
    }

    return NewPathGuard(r.mux, exacts, prefixes, true)
}