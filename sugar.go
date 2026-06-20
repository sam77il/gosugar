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
	"sync"
	"time"
)

type sugar struct {
	config               *Config
	routers              []*Router
	middlewares          []*Middleware
	corsAllowMethods     string
	corsAllowHeaders     string
	corsAllowCredentials string
	tree                 *routeTree
	treeOnce             sync.Once
}

type Router struct {
	prefix string
	routes []*route
}

type Context struct {
	Request *Request
	Response *Response
	Header http.Header
	Cookies Cookies
}

type sugarHandler = func(*Context) error

type Middleware struct {
	URL      string
	Handler  sugarHandler
	segments []string
	starIdx  int
}

type CorsSettings struct {
	Enabled bool
	Origins []string
	Methods []string
	Headers []string
	Credentials bool
}

type route struct {
	path          string
	method        string
	handler       sugarHandler
	extraHandlers []sugarHandler
	segments      []string
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
	s.buildTree()
	requestSegments := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	w.Header().Add("X-Powered-By", "Sugar")

	if r, params, ok := s.tree.match(req.Method, requestSegments); ok {
		s.handleRoute(w, req, r, params)
		return
	}

	if s.config.DefaultNotFoundHandler != nil {
		ctx := acquireContext()
		defer releaseContext(ctx)
		*ctx.Request = Request{
			Method:    req.Method,
			Header:    req.Header,
			URL:       req.URL.Path,
			UserAgent: req.UserAgent(),
			req:       req,
			writer:    w,
		}
		*ctx.Response = Response{
			writer: w,
			req:    req,
			config: s.config,
		}
		ctx.Header = w.Header()
		ctx.Cookies = Cookies{req: req, writer: w}
		if err := s.config.DefaultNotFoundHandler(ctx); err != nil {
			fmt.Println(err)
		}
		return
	}
	http.Error(w, "route not found", 404)
}

func (s *sugar) handleRoute(w http.ResponseWriter, req *http.Request, route *route, params map[string]string) {
	ctx, cancel := context.WithTimeout(req.Context(), s.config.Timeout)
	defer cancel()
	req = req.WithContext(ctx)

	reqSegs := strings.Split(req.URL.Path, "/")
	mwIndex := slices.IndexFunc(s.middlewares, func(m *Middleware) bool {
		if m.starIdx >= 0 {
			if len(reqSegs) < m.starIdx {
				return false
			}
			return slices.Equal(reqSegs[:m.starIdx], m.segments[:m.starIdx])
		}
		return slices.Equal(reqSegs, m.segments)
	})

	if s.config.Cors.Enabled {
		origin := req.Header.Get("Origin")

		if origin != "" && slices.Contains(s.config.Cors.Origins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", s.corsAllowMethods)
			w.Header().Set("Access-Control-Allow-Headers", s.corsAllowHeaders)
			w.Header().Set("Access-Control-Allow-Credentials", s.corsAllowCredentials)
		}

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
	}

	ip, port, _ := net.SplitHostPort(req.RemoteAddr)

	handlerContext := acquireContext()
	defer releaseContext(handlerContext)
	*handlerContext.Request = Request{
		Method:        req.Method,
		Header:        req.Header,
		URL:           req.URL.Path,
		IP:            IP{Adress: ip, Port: port, Extended: req.RemoteAddr},
		UserAgent:     req.UserAgent(),
		req:           req,
		Context:       ctx,
		config:        s.config,
		Params:        params,
		writer:        w,
		extraHandlers: route.extraHandlers,
	}
	*handlerContext.Response = Response{
		writer: w,
		req:    req,
		config: s.config,
	}
	handlerContext.Header = w.Header()
	handlerContext.Cookies = Cookies{req: req, writer: w}

	var (
		bodyContent []byte
		err         error
	)
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		bodyContent, err = io.ReadAll(req.Body)
		if err != nil {
			fmt.Println("Error parsing body")
			return
		}
	}
	handlerContext.Request.Body = bodyContent

	if mwIndex >= 0 {
		m := s.middlewares[mwIndex]
		handlerContext.Request.extraHandlers = append(handlerContext.Request.extraHandlers, route.handler)
		err = m.Handler(handlerContext)
	} else {
		err = route.handler(handlerContext)
	}

	if err != nil {
		if s.config.DefaultErrorHandler != nil {
			if err := s.config.DefaultErrorHandler(handlerContext); err != nil {
				fmt.Println(err)
				http.Error(w, "error on route "+route.path, 500)
			}
			return
		}
		fmt.Println(err)
		http.Error(w, "error on route "+route.path, 500)
	}
}

func (s *sugar) buildTree() {
	s.treeOnce.Do(func() {
		s.tree = buildRouteTree(s.routers)
	})
}

func (s *sugar) Listen() {
	s.buildTree()
	address := fmt.Sprintf(":%d", s.config.Port)
	server := &http.Server{
		Addr:              address,
		Handler:           s,
		ReadHeaderTimeout: s.config.ReadHeaderTimeout,
		ReadTimeout:       s.config.ReadTimeout,
		WriteTimeout:      s.config.WriteTimeout,
		IdleTimeout:       s.config.IdleTimeout,
		MaxHeaderBytes:    s.config.MaxHeaderBytes,
	}
	fmt.Printf(">> Sugar started on port %d <<\n", s.config.Port)
	if s.config.BeforeServerStart != nil {
		s.config.BeforeServerStart()
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(0)
	}
}

func (s *sugar) Middleware(url string, handler func(*Context) error) {
	segs := strings.Split(url, "/")
	s.middlewares = append(s.middlewares, &Middleware{
		URL:      url,
		Handler:  handler,
		segments: segs,
		starIdx:  slices.Index(segs, "*"),
	})
}

func (router *Router) Get(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodGet, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *Router) Post(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodPost, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *Router) Delete(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodDelete, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *Router) Patch(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodPatch, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (router *Router) Put(path string, sh sugarHandler, extrahandlers ...sugarHandler) {
	segments := strings.Split(strings.Trim(router.prefix + path, "/"), "/")
	router.routes = append(router.routes, &route{method: http.MethodPut, path: router.prefix + path, handler: sh, segments: segments, extraHandlers: extrahandlers})
}

func (s *Router) Static(folderPath string, urlPath string) {
	staticFolderHandler := func (ctx *Context) error {
		fileName := ctx.Request.Params["file"]
		cleanPath := filepath.Clean(fileName)
		
		if strings.Contains(cleanPath, "..") {
			return errors.New("invalid path")
		}

		fullPath := filepath.Join(folderPath, cleanPath)
		absBase, _ := filepath.Abs(folderPath)
		absFile, _ := filepath.Abs(fullPath)

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
	if config.ReadHeaderTimeout == 0 {
		config.ReadHeaderTimeout = time.Second * 5
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = time.Second * 30
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = time.Second * 30
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = time.Second * 120
	}
	if config.MaxHeaderBytes == 0 {
		config.MaxHeaderBytes = 1 << 20 // 1 MB
	}

	s := &sugar{
		config:  &config,
		routers: []*Router{},
	}
	if config.Cors.Enabled {
		s.corsAllowMethods = strings.Join(config.Cors.Methods, ", ")
		s.corsAllowHeaders = strings.Join(config.Cors.Headers, ", ")
		s.corsAllowCredentials = fmt.Sprintf("%t", config.Cors.Credentials)
	}
	return s
}

func (s *sugar) Group(prefix string) *Router {
	router := &Router{prefix: prefix}
	s.routers = append(s.routers, router)
	return router
}

func (s *sugar) Router() *Router {
	router := &Router{}
	s.routers = append(s.routers, router)
	return router
}

