package sugar

import (
	"encoding/json"
	"net/http"
)

type SugarResponse struct {
	res http.ResponseWriter
}

func (s *SugarResponse) JSON(v any) error {
	s.res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(s.res)
	err := enc.Encode(v)
	
	return err
}

func (s *SugarResponse) Text(v string) error {
	s.res.Header().Set("Content-Type", "text/plain")
	_, err := s.res.Write([]byte(v))

	return err
}

func (s *SugarResponse) Status(statusCode int) *SugarResponse {
	s.res.WriteHeader(statusCode)
	return s
}