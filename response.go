package sugar

import (
	"encoding/json"
	"net/http"
)

type SugarResponse struct {
	res http.ResponseWriter
}

func (s *SugarResponse) JSON(statusCode int, v any) {
	s.res.Header().Add("Content-Type", "application/json")
	s.res.WriteHeader(statusCode)
	t := json.NewEncoder(s.res)
	t.Encode(v)
}
