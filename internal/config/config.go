package config

type Config struct {
	BaseURL  string
	HttpPort int
	Db       struct {
		Dsn         string
		Automigrate bool
	}
	Jwt struct {
		SecretKey string
	}
	Notifications struct {
		Email string
	}
	Smtp struct {
		Host     string
		Port     int
		Username string
		Password string
		From     string
	}
	FileUploader struct {
		CloudName string
		ApiKey    string
		ApiSecret string
	}
	KafkaServers string
}
