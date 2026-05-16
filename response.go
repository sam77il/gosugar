package sugar

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Response struct {
	writer http.ResponseWriter
	req *http.Request
	statusCode int
}

func (res *Response) JSON(v any) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	res.writer.WriteHeader(res.statusCode)
	res.writer.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res.writer)
	err := enc.Encode(v)
	
	return err
}

func (res *Response) Text(v string) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	res.writer.WriteHeader(res.statusCode)
	res.writer.Header().Set("Content-Type", "text/plain")
	_, err := res.writer.Write([]byte(v))

	return err
}

func (res *Response) Status(statusCode int) *Response {
	res.statusCode = statusCode
	return res
}

func (res *Response) Redirect(url string) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}

	http.Redirect(res.writer, res.req, url, res.statusCode)
	return nil
}