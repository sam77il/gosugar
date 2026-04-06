package sugar

import (
	"encoding/json"
	"net/http"
)

type SugarResponse struct {
	res http.ResponseWriter
}

func (s *SugarResponse) JSON(v any) {
	s.res.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(s.res)
	enc.Encode(v)
}

func (s *SugarResponse) Status(statusCode int) *SugarResponse {
	s.res.WriteHeader(statusCode)
	return s
}