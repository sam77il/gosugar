package sugar

import (
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
	Metadata         []byte
	req              *http.Request
}

func (s *SugarRequest) GetQuery(query string) string {
	queries := s.req.URL.Query()
	if queries[query][0] != "" {
		return  queries[query][0]
	}
	return ""
}

func (s *SugarRequest) GetParam(slug string) string {
	return s.req.PathValue(slug)
}
