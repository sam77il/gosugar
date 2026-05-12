package sugar

import (
	"context"
	"fmt"
	"net/http"
)

type SugarRequest struct {
	Method           string
	HTTPVersion      string
	HTTPVersionMajor int
	HTTPVersionMinor int
	Header           http.Header
	Body             []byte
	URL              string
	IP IP
	GoCtx         context.Context
	req              *http.Request
	writer http.ResponseWriter
	Params map[string]string
	next sugarHandler
	extraHandlers []sugarHandler
	currentHandler int
}

type IP struct {
	Adress string
	Extended string
	Port string
}

func (s *SugarRequest) GetQuery(query string) string {
	queries := s.req.URL.Query()
	if queries[query][0] != "" {
		return  queries[query][0]
	}
	return ""
}

func (s *SugarRequest) AddCtx(key any, value any) {
	s.GoCtx = context.WithValue(s.GoCtx, key, value)
	s.req = s.req.WithContext(s.GoCtx)
}

func (s *SugarRequest) Next() {
	if s.currentHandler >= len(s.extraHandlers) {
		fmt.Println("no handlers")
		return
	}

	next := s.extraHandlers[s.currentHandler]
	s.currentHandler++

	err := next(&SugarContext{
		Request: s,
		Response: &SugarResponse{res: s.writer},
	})
	if err != nil {
		http.Error(s.writer, "error on route " + s.URL, 500)
	}
}