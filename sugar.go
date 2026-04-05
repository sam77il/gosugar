package sugar

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

type sugar struct {
	config *Config
	cors CorsSettings
}

type Config struct {
	Host string
	Port int
	Cors CorsSettings
}

type SugarRequest struct {
	Method string
	HTTPVersion string
	HTTPVersionMajor int
	HTTPVersionMinor int
	Header http.Header
	Body []byte
	URL string
	Metadata []byte
	req *http.Request
}

func (s SugarRequest) GetParam(slug string) string {
	return s.req.PathValue(slug)
}

type SugarResponse struct {
	res http.ResponseWriter
}

func (s SugarResponse) JSON(statusCode int, v any) {
	s.res.Header().Add("Content-Type", "application/json")
	s.res.WriteHeader(statusCode)
	t := json.NewEncoder(s.res)
	t.Encode(v)
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

func (s sugar) Listen() {
	http.ListenAndServe(fmt.Sprintf("%s:%d", s.config.Host, s.config.Port), *sugarMux)
}

func (s sugar) Middleware(url string, handler func(*SugarContext, func())) {
	sugarMiddlewares = append(sugarMiddlewares, SugarMiddleware{
		URL: url,
		Handler: handler,
	})
}

func (s *sugar) Get(path string, sh sugarHandler) {
	addRoute(http.MethodGet, path, sh, s.cors)
}

func addRoute(method string, path string, sh sugarHandler, cors CorsSettings) {
	sugarMux.HandleFunc(method + " " + path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Powered-By", "Sugar")

		if cors.Enabled {
			origin := r.Header.Get("Origin")

			if origin != "" && slices.Contains(cors.Origins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.Methods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cors.Headers, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%t", cors.Credentials))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		mwIndex := slices.IndexFunc(sugarMiddlewares, func(m SugarMiddleware) bool {
			requestSegments := strings.Split(r.URL.Path, "/")
			mwPathSegments := strings.Split(m.URL, "/")
			starIndex := slices.Index(mwPathSegments, "*")

			return slices.Equal(requestSegments[:starIndex], mwPathSegments[:starIndex])		
		})

		handlerContext := &SugarContext{
			Request: &SugarRequest{
				Method: r.Method,
				Header: r.Header,
				URL: r.URL.Path,
				req: r,
			},
			Response: &SugarResponse{
				res: w,
			},
		}

		// Checking method and adding body
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodTrace {
			bodyContent, err := io.ReadAll(r.Body)
			if err != nil {
				fmt.Println("Error parsing body")
				return 
			}
			handlerContext.Request.Body = bodyContent
		} 

		if mwIndex >= 0 {
			m := sugarMiddlewares[mwIndex]
			m.Handler(handlerContext, func() {
				sh(handlerContext)
			})
		} else if r.Method == method {
			sh(handlerContext)
		}
	})
}

func New(config *Config) *sugar {
	sugarMux = &SugarMux{
		ServeMux: http.NewServeMux(),
	}

	return &sugar{
		config: config,
		cors: config.Cors,
	}
}