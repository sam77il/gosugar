package sugar

import "net/http"

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

func (s SugarRequest) GetParam(slug string) string {
	return s.req.PathValue(slug)
}
