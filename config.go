package sugar

import (
	"time"
)

type Config struct {
	AppName 	string
	Port    	int
	Cors    	CorsSettings
	Timeout 	time.Duration
	DisableDefaultContentType bool
	DisableDefaultDate bool
	DefaultHeaders map[string]string
	DefaultErrorHandler sugarHandler
	DefaultNotFoundHandler sugarHandler
	BeforeServerStart func()
}
