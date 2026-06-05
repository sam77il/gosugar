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
	config *Config
}

func (res *Response) JSON(v any) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}

	if !res.config.DisableDefaultContentType {
		res.writer.Header().Set("Content-Type", "application/json")
	}
	if !res.config.DisableDefaultDate {
		res.writer.Header().Set("Date", http.TimeFormat)
	}
	res.setDefaultHeaders()	
	res.writer.WriteHeader(res.statusCode)

	enc := json.NewEncoder(res.writer)
	return enc.Encode(v)
}

func (res *Response) Text(v string) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	
	if !res.config.DisableDefaultContentType {
		res.writer.Header().Set("Content-Type", "text/plain")
	}
	if !res.config.DisableDefaultDate {
		res.writer.Header().Set("Date", http.TimeFormat)
	}
	res.setDefaultHeaders()
	res.writer.WriteHeader(res.statusCode)
	
	_, err := res.writer.Write([]byte(v))
	return err
}

func (res *Response) setDefaultHeaders() {
	if len(res.config.DefaultHeaders) > 0 {
		for header, value := range res.config.DefaultHeaders {
			res.writer.Header().Set(header, value)
		}
	}
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