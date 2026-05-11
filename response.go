package sugar

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type SugarResponse struct {
	res http.ResponseWriter
}

func (s *SugarResponse) JSON(v any) {
	s.res.Header().Add("Content-Type", "application/json")
	enc := json.NewEncoder(s.res)
	err := enc.Encode(v)
	if err != nil {
		fmt.Println(err)
	}
}

func (s *SugarResponse) Status(statusCode int) *SugarResponse {
	s.res.WriteHeader(statusCode)
	return s
}