package sugar

import (
	"encoding/json"
	"errors"
	"net/http"
)

type SugarResponse struct {
	writer http.ResponseWriter
	req *http.Request
	statusCode int
}

func (res *SugarResponse) JSON(v any) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	res.writer.WriteHeader(res.statusCode)
	res.writer.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res.writer)
	err := enc.Encode(v)
	
	return err
}

func (res *SugarResponse) Text(v string) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	res.writer.WriteHeader(res.statusCode)
	res.writer.Header().Set("Content-Type", "text/plain")
	_, err := res.writer.Write([]byte(v))

	return err
}

func (res *SugarResponse) Status(statusCode int) *SugarResponse {
	res.statusCode = statusCode
	return res
}

func (res *SugarResponse) Redirect(url string) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}

	http.Redirect(res.writer, res.req, url, res.statusCode)
	return nil
}