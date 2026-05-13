package sugar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type sugar struct {
	config *Config
	routers []*sugarRouter
	middlewares []*SugarMiddleware
}

type sugarRouter struct {
	prefix string
	routes []*route
}

type SugarContext struct {
	Request *SugarRequest
	Response *SugarResponse
	Header http.Header
	Cookies Cookies
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

type route struct {
	path string
	method string
	handler sugarHandler
	extraHandlers []sugarHandler
	currentHandler int
	segments []string
}

type Cookies struct {
	req *http.Request
	writer http.ResponseWriter
}

func (c *Cookies) Set(cookieData J) {
	cookieBytes, err := json.Marshal(cookieData)
	if err != nil {
		fmt.Println(err)
		return
	}
	var cookie http.Cookie
	json.Unmarshal(cookieBytes, &cookie)
	fmt.Println(cookie)
	http.SetCookie(c.writer, &cookie)
}

func (c *Cookies) Get(name string) J {
	cookie, err := c.req.Cookie(name)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	cookieBytes, err := json.Marshal(cookie)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var cookieMap J
	err = json.Unmarshal(cookieBytes, &cookieMap)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return cookieMap
}

type J = map[string]any

func (s *sugar) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	requestSegments := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	w.Header().Add("X-Powered-By", "Sugar")
	for _, router := range s.routers {
		for _, route := range router.routes {
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
	}
	http.Error(w, "route not found", 404)
}

func (s *sugar) handleRoute(w http.ResponseWriter, req *http.Request, route *route, params map[string]string) {
	ctx, cancel := context.WithTimeout(req.Context(), s.config.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	mwIndex := slices.IndexFunc(s.middlewares, func(m *SugarMiddleware) bool {
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
				writer: w,
				req: req,
			},
			Header: w.Header(),
			Cookies: Cookies{
				req: req,
				writer: w,
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
			m := s.middlewares[mwIndex]
			handlerContext.Request.extraHandlers = append(handlerContext.Request.extraHandlers, route.handler)
			err := m.Handler(handlerContext)
			if err != nil {
				fmt.Println(err)
				http.Error(w, "error on route " + route.path, 500)
			}
		} else {
			err := route.handler(handlerContext)
			if err != nil {
				fmt.Println(err)
				http.Error(w, "error on route " + route.path, 500)
			}
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
		Handler: s,
	}
	fmt.Printf(">> Sugar started on port %d <<\n",s.config.Port)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}

func (s *sugar) Middleware(url string, handler func(*SugarContext) error) {
	s.middlewares = append(s.middlewares, &SugarMiddleware{
		URL: url,
		Handler: handler,
	})
}

func (router *sugarRouter) Get(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodGet, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *sugarRouter) Post(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodPost, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *sugarRouter) Delete(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodDelete, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *sugarRouter) Patch(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodPatch, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *sugarRouter) Put(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodPut, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (s *sugarRouter) Static(folderPath string, urlPath string) {
	staticFolderHandler := func (ctx *SugarContext) error {
		fileName := ctx.Request.Params["file"]
		cleanPath := filepath.Clean(fileName)
		
		if strings.Contains(cleanPath, "..") {
			return errors.New("invalid path")
		}

		fullPath := filepath.Join(folderPath, cleanPath)
		fmt.Println(fullPath)
		absBase, _ := filepath.Abs(folderPath)
		absFile, _ := filepath.Abs(fullPath)
		fmt.Println(absBase)
		fmt.Println(absFile)

		if !strings.HasPrefix(absFile, absBase) {
			return errors.New("access denied")
		}

		http.ServeFile(ctx.Request.writer, ctx.Request.req, absFile)

		return nil
	}

	p := urlPath + "/*file"
	segments := strings.Split(strings.Trim(p, "/"), "/")

	s.routes = append(s.routes, &route{method: http.MethodGet, path:p, handler: staticFolderHandler, segments: segments})
}

func New(config Config) *sugar {
	if config.Timeout == 0 {
		config.Timeout = time.Second * 30
	}

	return &sugar{
		config: &config,
		routers: []*sugarRouter{},
	}
}

func (s *sugar) Group(prefix string) *sugarRouter {
	router := &sugarRouter{prefix: prefix}
	s.routers = append(s.routers, router)
	return router
}

func (s *sugar) Router() *sugarRouter {
	router := &sugarRouter{}
	s.routers = append(s.routers, router)
	return router
}

func matchRoute(
	routeSegments []string,
	requestSegments []string,
) (map[string]string, bool) {

	params := map[string]string{}

	for i := range routeSegments {

		if i >= len(requestSegments) {
			return nil, false
		}

		rSeg := routeSegments[i]
		reqSeg := requestSegments[i]

		if strings.HasPrefix(rSeg, "*") {

			paramName := rSeg[1:]

			params[paramName] = strings.Join(requestSegments[i:], "/")

			return params, true
		}

		if strings.HasPrefix(rSeg, ":") {

			paramName := rSeg[1:]

			params[paramName] = reqSeg

			continue
		}

		if rSeg != reqSeg {
			return nil, false
		}
	}

	if len(requestSegments) != len(routeSegments) {
		return nil, false
	}

	return params, true
}