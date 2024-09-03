package config

import "os"

type Config struct {
	Address       string
	JaegerAddress string
}

func GetConfig() Config {
	return Config{
		Address:       os.Getenv("NOTIFICATION_SERVICE_PORT"),
		JaegerAddress: os.Getenv("JAEGER_ADDRESS"),
	}
}
