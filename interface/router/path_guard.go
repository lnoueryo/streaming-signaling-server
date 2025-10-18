package router

import (
	"context"
	"net/http"
	"regexp"
)

type PathGuard struct {
	http.Handler
	exacts        map[string]struct{}
	prefixes      []string
	redirectSlash bool
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    path := req.URL.Path

    for _, route := range r.routes {
        if route.Method != req.Method {
            continue
        }

        if matches := route.Regex.FindStringSubmatch(path); matches != nil {
            // パラメータを抽出
            paramNames := extractParamNames(route.Path)
            params := map[string]string{}
            for i, name := range paramNames {
                if i+1 < len(matches) {
                    params[name] = matches[i+1]
                }
            }

            // コンテキストに埋め込む
            ctx := context.WithValue(req.Context(), "params", params)
            req = req.WithContext(ctx)

            route.Handler.ServeHTTP(w, req)
            return
        }
    }

    http.NotFound(w, req)
}

func NewPathGuard(next http.Handler, exacts map[string]struct{}, prefixes []string, redirectSlash bool) http.Handler {
	return &PathGuard{next, exacts, prefixes, redirectSlash}
}

func extractParamNames(pattern string) []string {
    re := regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)
    matches := re.FindAllStringSubmatch(pattern, -1)
    names := []string{}
    for _, m := range matches {
        names = append(names, m[1])
    }
    return names
}