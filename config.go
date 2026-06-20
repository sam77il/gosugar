package sugar

import (
	"time"
)

type Config struct {
	AppName             string
	Port                int
	Cors                CorsSettings
	Timeout             time.Duration
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	MaxHeaderBytes      int
	DisableDefaultContentType bool
	DisableDefaultDate        bool
	DefaultHeaders            map[string]string
	DefaultErrorHandler       sugarHandler
	DefaultNotFoundHandler    sugarHandler
	BeforeServerStart         func()
}
