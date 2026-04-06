package sugar

import (
	"fmt"
	"net/http"
)

type sugar struct {
	config *Config
}

type SugarContext struct {
	Request *SugarRequest
	Response *SugarResponse
}

type sugarHandler = func(*SugarContext)

type SugarMux struct {
	*http.ServeMux
}

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

var sugarMux *SugarMux
var sugarMiddlewares []SugarMiddleware

func (s *sugar) Listen() {
	http.ListenAndServe(fmt.Sprintf("%s:%d", s.config.Host, s.config.Port), *sugarMux)
}

func (s *sugar) Middleware(url string, handler func(*SugarContext, func())) {
	sugarMiddlewares = append(sugarMiddlewares, SugarMiddleware{
		URL: url,
		Handler: handler,
	})
}

func (s *sugar) Get(path string, sh sugarHandler) {
	addRoute(http.MethodGet, path, sh, s.config)
}

func (s *sugar) Post(path string, sh sugarHandler) {
	addRoute(http.MethodPost, path, sh, s.config)
}

func (s *sugar) Delete(path string, sh sugarHandler) {
	addRoute(http.MethodDelete, path, sh, s.config)
}

func (s *sugar) Patch(path string, sh sugarHandler) {
	addRoute(http.MethodPatch, path, sh, s.config)
}

func (s *sugar) Put(path string, sh sugarHandler) {
	addRoute(http.MethodPut, path, sh, s.config)
}

func New(config *Config) *sugar {
	sugarMux = &SugarMux{
		ServeMux: http.NewServeMux(),
	}

	return &sugar{
		config: config,
	}
}