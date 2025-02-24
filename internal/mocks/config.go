package mocks

import "github.com/cradoe/morenee/internal/config"

var MockConfig = &config.Config{
	BaseURL:  "http://localhost",
	HttpPort: 8080,
	Db: struct {
		Dsn         string
		Automigrate bool
	}{
		Dsn:         "mock_dsn",
		Automigrate: false,
	},
	RedisServer: "localhost:6379",
	Jwt: struct {
		SecretKey string
	}{
		SecretKey: "test_secret",
	},
	Notifications: struct {
		Email string
	}{
		Email: "no-reply@example.com",
	},
	Smtp: struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user@example.com",
		Password: "password",
		From:     "no-reply@example.com",
	},
	KafkaServers: "localhost:9092",
}
