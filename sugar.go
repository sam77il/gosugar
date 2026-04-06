package sugar

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
)

type sugar struct {
	config *Config
	router *sugarRouter
}

type sugarRouter struct {
	routes []*route
	middlewares []*SugarMiddleware
}

type SugarContext struct {
	Request *SugarRequest
	Response *SugarResponse
}

type sugarHandler = func(*SugarContext)

type SugarMiddleware struct {
	URL string
	Handler func(*SugarContext, func())
}

type CorsSettings struct {
	Enabled bool
	Origins []string
	Methods []string
	Headers []string
	Credentials bool
}

type sugarMux struct {
	router *sugarRouter
	config *Config
}

type route struct {
	path string
	method string
	handler sugarHandler
	segments []string
}

func (s *sugarMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	requestSegments := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	path := strings.Trim(req.URL.Path, "/")
	fmt.Println(path)
	for _, route := range s.router.routes {
		if route.method != req.Method {
			continue
		}

		params, ok :=
			matchRoute(
				route.segments,
				requestSegments,
			)

		if !ok {
			continue
		}

		s.handleRoute(w, req, route, params)
		return
	}
	http.Error(w, "route not found", 404)
}

func (s *sugarMux) handleRoute(w http.ResponseWriter, req *http.Request, route *route, params map[string]string) {
	ctx, cancel := context.WithTimeout(req.Context(), s.config.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	resDone := make(chan struct{})
	go func() {
		defer close(resDone)

		w.Header().Add("X-Powered-By", "Sugar")

		if s.config.Cors.Enabled {
			origin := req.Header.Get("Origin")

			if origin != "" && slices.Contains(s.config.Cors.Origins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(s.config.Cors.Methods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(s.config.Cors.Headers, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%t", s.config.Cors.Credentials))
			}

			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		mwIndex := slices.IndexFunc(s.router.middlewares, func(m *SugarMiddleware) bool {
			requestSegments := strings.Split(req.URL.Path, "/")
			mwPathSegments := strings.Split(m.URL, "/")
			starIndex := slices.Index(mwPathSegments, "*")

			return slices.Equal(requestSegments[:starIndex], mwPathSegments[:starIndex])
		})

		handlerContext := &SugarContext{
			Request: &SugarRequest{
				Method: req.Method,
				Header: req.Header,
				URL:    req.URL.Path,
				req:    req,
				GoCtx: ctx,
				Params: params,
			},
			Response: &SugarResponse{
				res: w,
			},
		}

		// Checking method and adding body
		if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodTrace {
			bodyContent, err := io.ReadAll(req.Body)
			if err != nil {
				fmt.Println("Error parsing body")
				return
			}
			handlerContext.Request.Body = bodyContent
		}

		if mwIndex >= 0 {
			m := s.router.middlewares[mwIndex]
			m.Handler(handlerContext, func() {
				route.handler(handlerContext)
			})
		} else {
			route.handler(handlerContext)
		}
	}()

	select {
	case <- ctx.Done():
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	case <- resDone:
		fmt.Println("done")
	}
}

func (s *sugar) Listen() {
	server := &http.Server{
		Addr: s.config.Host,
		Handler: &sugarMux{
			router: s.router,
			config: s.config,
		},
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}

func (s *sugar) Middleware(url string, handler func(*SugarContext, func())) {
	s.router.middlewares = append(s.router.middlewares, &SugarMiddleware{
		URL: url,
		Handler: handler,
	})
}

func (s *sugar) Get(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodGet, path: path, handler: sh, segments: segments})
	// addRoute(http.MethodGet, path, sh, s.config)
}

func (s *sugar) Post(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodPost, path: path, handler: sh, segments: segments})
	// addRoute(http.MethodPost, path, sh, s.config)
}

func (s *sugar) Delete(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodDelete, path: path, handler: sh, segments: segments})
	// addRoute(http.MethodDelete, path, sh, s.config)
}

func (s *sugar) Patch(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodPatch, path: path, handler: sh, segments: segments})
	// addRoute(http.MethodPatch, path, sh, s.config)
}

func (s *sugar) Put(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodPut, path: path, handler: sh, segments: segments})
	// addRoute(http.MethodPut, path, sh, s.config)
}

func New(config *Config) *sugar {
	return &sugar{
		config: config,
		router: &sugarRouter{},
	}
}

func matchRoute(
	routeSegments []string,
	requestSegments []string,
) (map[string]string, bool) {

	if len(routeSegments) != len(requestSegments) {
		return nil, false
	}

	params := map[string]string{}

	for i := range routeSegments {

		rSeg := routeSegments[i]
		reqSeg := requestSegments[i]

		if strings.HasPrefix(rSeg, ":") {

			paramName := rSeg[1:]

			params[paramName] = reqSeg

			continue
		}

		if rSeg != reqSeg {
			return nil, false
		}
	}

	return params, true
}