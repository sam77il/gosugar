package sugar

import "time"

type Config struct {
	Host    string
	Cors    CorsSettings
	Timeout time.Duration
}
