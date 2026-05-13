package sugar

import (
	"encoding/json"
	"errors"
	"net/http"
)

type SugarResponse struct {
	res http.ResponseWriter
	req *http.Request
	statusCode int
}

func (res *SugarResponse) JSON(v any) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	res.res.WriteHeader(res.statusCode)
	res.res.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(res.res)
	err := enc.Encode(v)
	
	return err
}

func (res *SugarResponse) Text(v string) error {
	if res.statusCode == 0 {
		return errors.New("no status code given")
	}
	res.res.WriteHeader(res.statusCode)
	res.res.Header().Set("Content-Type", "text/plain")
	_, err := res.res.Write([]byte(v))

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

	http.Redirect(res.res, res.req, url, res.statusCode)
	return nil
}