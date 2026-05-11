package sugar

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
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

type sugarHandler = func(*SugarContext) error

type SugarMiddleware struct {
	URL string
	Handler sugarHandler
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
	extraHandlers []sugarHandler
	currentHandler int
	segments []string
}

type J = map[string]any

func (s *sugarMux) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	requestSegments := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	w.Header().Add("X-Powered-By", "Sugar")
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

	mwIndex := slices.IndexFunc(s.router.middlewares, func(m *SugarMiddleware) bool {
		requestSegments := strings.Split(req.URL.Path, "/")
		mwPathSegments := strings.Split(m.URL, "/")
		starIndex := slices.Index(mwPathSegments, "*")
		if starIndex >= 0 {
			return slices.Equal(requestSegments[:starIndex], mwPathSegments[:starIndex])
		}
		return slices.Equal(requestSegments, mwPathSegments)
	})

	resDone := make(chan int)
	go func() {
		defer close(resDone)

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
		ip, port, err := net.SplitHostPort(req.RemoteAddr)
		handlerContext := &SugarContext{
			Request: &SugarRequest{
				Method: req.Method,
				Header: req.Header,
				URL:    req.URL.Path,
				IP: IP{
					Adress: ip,
					Port: port,
					Extended: req.RemoteAddr,
				},
				req:    req,
				GoCtx: ctx,
				Params: params,
				writer: w,
				extraHandlers: route.extraHandlers,
			},
			Response: &SugarResponse{
				res: w,
			},
		}

		// Checking method and adding body
		bodyContent, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Println("Error parsing body")
			return
		}
		handlerContext.Request.Body = bodyContent

		if mwIndex >= 0 {
			m := s.router.middlewares[mwIndex]
			handlerContext.Request.next = route.handler
			m.Handler(handlerContext)
		} else {
			route.handler(handlerContext)
		}
	}()

	select {
	case <- ctx.Done():
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
	case <- resDone:
		
	}
}

func (s *sugar) Listen() {
	address := fmt.Sprintf(":%d", s.config.Port)
	server := &http.Server{
		Addr: address,
		Handler: &sugarMux{
			router: s.router,
			config: s.config,
		},
	}
	fmt.Printf(">> Sugar started on port %d <<\n",s.config.Port)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}

func (s *sugar) Middleware(url string, handler func(*SugarContext) error) {
	s.router.middlewares = append(s.router.middlewares, &SugarMiddleware{
		URL: url,
		Handler: handler,
	})
}

func (s *sugar) Get(path string, sh sugarHandler, shs ...sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodGet, path: path, handler: sh, extraHandlers: shs, segments: segments})
}

func (s *sugar) Post(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodPost, path: path, handler: sh, segments: segments})
}

func (s *sugar) Delete(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodDelete, path: path, handler: sh, segments: segments})
}

func (s *sugar) Patch(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodPatch, path: path, handler: sh, segments: segments})
}

func (s *sugar) Put(path string, sh sugarHandler) {
	segments := strings.Split(strings.Trim(path, "/"), "/")
	s.router.routes = append(s.router.routes, &route{method: http.MethodPut, path: path, handler: sh, segments: segments})
}

func New(config Config) *sugar {
	if config.Timeout == 0 {
		config.Timeout = time.Second * 30
	}

	return &sugar{
		config: &config,
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