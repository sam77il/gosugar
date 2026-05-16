package sugar

import (
	"context"
	"fmt"
	"net/http"
)

type Request struct {
	Method           string
	HTTPVersion      string
	HTTPVersionMajor int
	HTTPVersionMinor int
	Header           http.Header
	Body             []byte
	URL              string
	IP IP
	UserAgent string
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

func (s *Request) GetQuery(query string) string {
	queries := s.req.URL.Query()
	if queries[query][0] != "" {
		return  queries[query][0]
	}
	return ""
}

func (s *Request) AddCtx(key any, value any) {
	s.GoCtx = context.WithValue(s.GoCtx, key, value)
	s.req = s.req.WithContext(s.GoCtx)
}

func (s *Request) Next() error {
	if s.currentHandler >= len(s.extraHandlers) {
		fmt.Println("no handlers")
		return nil
	}

	next := s.extraHandlers[s.currentHandler]
	s.currentHandler++

	err := next(&Context{
		Request: s,
		Response: &Response{writer: s.writer},
	})
	return err
}