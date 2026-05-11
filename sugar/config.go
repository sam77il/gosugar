package sugar

import (
	"time"
)

type Config struct {
	Port    	int
	Cors    	CorsSettings
	Timeout 	time.Duration
	Static string
}
