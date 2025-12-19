package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret		 string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return &Config{
		JWTSecret:      os.Getenv("JWT_SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURI:  os.Getenv("GOOGLE_REDIRECT_URI"),
	}
}