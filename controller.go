package sugar

import (
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

func addRoute(method string, path string, sh sugarHandler, cors CorsSettings) {
	sugarMux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("X-Powered-By", "Sugar")

		if cors.Enabled {
			origin := r.Header.Get("Origin")

			if origin != "" && slices.Contains(cors.Origins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cors.Methods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cors.Headers, ", "))
				w.Header().Set("Access-Control-Allow-Credentials", fmt.Sprintf("%t", cors.Credentials))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}
		}

		mwIndex := slices.IndexFunc(sugarMiddlewares, func(m SugarMiddleware) bool {
			requestSegments := strings.Split(r.URL.Path, "/")
			mwPathSegments := strings.Split(m.URL, "/")
			starIndex := slices.Index(mwPathSegments, "*")

			return slices.Equal(requestSegments[:starIndex], mwPathSegments[:starIndex])
		})

		handlerContext := &SugarContext{
			Request: &SugarRequest{
				Method: r.Method,
				Header: r.Header,
				URL:    r.URL.Path,
				req:    r,
			},
			Response: &SugarResponse{
				res: w,
			},
		}

		// Checking method and adding body
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodTrace {
			bodyContent, err := io.ReadAll(r.Body)
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
		} else if r.Method == method {
			sh(handlerContext)
		}
	})
}