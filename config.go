package sugar

import "time"

type Config struct {
	Host    string
	Port    int
	Cors    CorsSettings
	Timeout time.Duration
}
