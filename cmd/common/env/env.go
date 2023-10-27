package env

import "os"

var (
	SecretKey = os.Getenv("SECRET_KEY")
)

const (
	DefaultModel = "mistral"
	DBName       = "chatbot_db"
	DBUserName   = "amenabshir"
)
