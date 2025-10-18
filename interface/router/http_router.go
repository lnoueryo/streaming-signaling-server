package router

import (
	"net/http"
	"regexp"
	"strings"

	"streaming-server.com/infrastructure/logger"
)

var log = logger.Log

type Adapter func(http.Handler) http.Handler

type Route struct {
	Path        string
	Method      string            // "", "GET", "POST", ...
	Regex       *regexp.Regexp
	Handler     http.Handler
	Middlewares []Adapter         // そのルート専用のミドルウェア（Groupで付与）
}

type Router struct {
	mux            *http.ServeMux
	globalAdapters []Adapter     // 全ルート共通
	routes         []Route
    prefix   string
}

func NewRouter() *Router {
	return &Router{
		mux:            http.NewServeMux(),
		globalAdapters: make([]Adapter, 0),
		routes:         make([]Route, 0),
	}
}

// ===== ルート登録（全HTTPメソッド） =====

func (r *Router) Get(path string, fn func(http.ResponseWriter, *http.Request)) {
	r.routes = append(r.routes, Route{Path: path, Method: http.MethodGet, Handler: http.HandlerFunc(fn)})
}
func (r *Router) Post(path string, fn func(http.ResponseWriter, *http.Request)) {
	r.routes = append(r.routes, Route{Path: path, Method: http.MethodPost, Handler: http.HandlerFunc(fn)})
}
func (r *Router) Put(path string, fn func(http.ResponseWriter, *http.Request)) {
	r.routes = append(r.routes, Route{Path: path, Method: http.MethodPut, Handler: http.HandlerFunc(fn)})
}
func (r *Router) Delete(path string, fn func(http.ResponseWriter, *http.Request)) {
	r.routes = append(r.routes, Route{Path: path, Method: http.MethodDelete, Handler: http.HandlerFunc(fn)})
}
func (r *Router) Patch(path string, fn func(http.ResponseWriter, *http.Request)) {
	r.routes = append(r.routes, Route{Path: path, Method: http.MethodPatch, Handler: http.HandlerFunc(fn)})
}

func (r *Router) Handle(path string, h http.Handler) {
	r.routes = append(r.routes, Route{Path: path, Method: "", Handler: h})
}
func (r *Router) HandleFunc(path string, fn func(http.ResponseWriter, *http.Request)) {
	r.Handle(path, http.HandlerFunc(fn))
}

func (r *Router) SetMiddleware(mw Adapter) {
	r.globalAdapters = append(r.globalAdapters, mw)
}

func (r *Router) Group(mws []Adapter, cfg func(g *Router)) {
	sub := &Router{
		mux:            r.mux,
		globalAdapters: append([]Adapter(nil), r.globalAdapters...), // 継承
		routes:         make([]Route, 0),
	}
	// グループ内で登録されたルートに mws を付与
	wrapped := func(g *Router) {
		cfg(g)
		for i := range g.routes {
			g.routes[i].Middlewares = append(g.routes[i].Middlewares, mws...)
		}
	}
	wrapped(sub)
	// 親へ取り込み
	r.routes = append(r.routes, sub.routes...)
}

func (r *Router) GroupWithPrefix(prefix string, mws []Adapter, cfg func(g *Router)) {
	sub := &Router{
		mux:            r.mux,
		globalAdapters: append([]Adapter(nil), r.globalAdapters...),
		routes:         make([]Route, 0),
	}
	cfg(sub)
	for _, rt := range sub.routes {
		full := joinPaths(prefix, rt.Path)
		rt.Path = full
		rt.Middlewares = append(rt.Middlewares, mws...)
		r.routes = append(r.routes, rt)
	}
}

func (r *Router) Route(prefix string, cfg func(g *Router)) {
	r.GroupWithPrefix(prefix, nil, cfg)
}

func joinPaths(a, b string) string {
	a = strings.TrimSuffix(a, "/")
	if b == "" || b == "/" {
		if a == "" { return "/" }
		return a
	}
	if !strings.HasPrefix(b, "/") { b = "/" + b }
	if a == "" { return b }
	return a + b
}
