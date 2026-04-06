package sugar

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

func addRoute(method string, path string, sh sugarHandler, cfg *Config) {
	sugarMux.HandleFunc(method+" "+path, func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), cfg.Timeout)
		defer cancel()
		req = req.WithContext(ctx)

		resDone := make(chan struct{})
		go func() {
			defer close(resDone)

			w.Header().Add("X-Powered-By", "Sugar")

			if cfg.Cors.Enabled {
				origin := req.Header.Get("Origin")

				if origin != "" && slices.Contains(cfg.Cors.Origins, origin) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.Cors.Methods, ", "))
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.Cors.Headers, ", "))
					w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%t", cfg.Cors.Credentials))
				}

				if req.Method == http.MethodOptions {
					w.WriteHeader(http.StatusOK)
					return
				}
			}

			mwIndex := slices.IndexFunc(sugarMiddlewares, func(m SugarMiddleware) bool {
				requestSegments := strings.Split(req.URL.Path, "/")
				mwPathSegments := strings.Split(m.URL, "/")
				starIndex := slices.Index(mwPathSegments, "*")

				return slices.Equal(requestSegments[:starIndex], mwPathSegments[:starIndex])
			})

			handlerContext := &SugarContext{
				Request: &SugarRequest{
					Method: req.Method,
					Header: req.Header,
					URL:    req.URL.Path,
					req:    req,
					GoCtx: ctx,
				},
				Response: &SugarResponse{
					res: w,
				},
			}

			// Checking method and adding body
			if req.Method != http.MethodGet && req.Method != http.MethodHead && req.Method != http.MethodTrace {
				bodyContent, err := io.ReadAll(req.Body)
				if err != nil {
					fmt.Println("Error parsing body")
					return
				}
				handlerContext.Request.Body = bodyContent
			}

			if mwIndex >= 0 {
				m := sugarMiddlewares[mwIndex]
				m.Handler(handlerContext, func() {
					sh(handlerContext)
				})
			} else if strings.EqualFold(req.Method, method) {
				sh(handlerContext)
			}
		}()

		select {
		case <- ctx.Done():
			http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		case <- resDone:
			fmt.Println("done")
		}
	})
}