package config

import (
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	ChatGptToken string `mapstructure:"CHAT_GPT_TOKEN"`
	IgToken      string `mapstructure:"IG_TOKEN"`
	IgID         string `mapstructure:"IG_ID"`
}

func LoadConfig(path string) (config Config, err error) {

	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}

func LoadFromEnvVars() Config {
	return Config{
		ChatGptToken: os.Getenv("CHAT_GPT_TOKEN"),
		IgToken:      os.Getenv("IG_TOKEN"),
		IgID:         os.Getenv("IG_ID"),
	}
}
