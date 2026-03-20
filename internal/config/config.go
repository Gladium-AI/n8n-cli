package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	KeyBaseURL = "base_url"
	KeyAPIKey  = "api_key"
	KeyOutput  = "output"
	KeyJSON    = "json"
	KeyYAML    = "yaml"
	KeyQuiet   = "quiet"
	KeyNoColor = "no_color"
	KeyDryRun  = "dry_run"

	DefaultBaseURL = "http://localhost:5678"
	DefaultOutput  = "summary"
)

func Init() {
	_ = godotenv.Load()

	viper.SetDefault(KeyBaseURL, DefaultBaseURL)
	viper.SetDefault(KeyOutput, DefaultOutput)
	viper.SetDefault(KeyJSON, false)
	viper.SetDefault(KeyYAML, false)
	viper.SetDefault(KeyQuiet, false)
	viper.SetDefault(KeyNoColor, false)
	viper.SetDefault(KeyDryRun, false)

	viper.SetEnvPrefix("N8N")
	viper.AutomaticEnv()
	viper.BindEnv(KeyBaseURL, "N8N_BASE_URL")
	viper.BindEnv(KeyAPIKey, "N8N_API_KEY")

	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(home)
		viper.AddConfigPath(filepath.Join(home, ".config", "n8n-cli"))
	}
	viper.AddConfigPath(".")
	viper.SetConfigName(".n8n-cli")
	viper.SetConfigType("yaml")

	_ = viper.ReadInConfig()
}

func BaseURL() string {
	return viper.GetString(KeyBaseURL)
}

func APIKey() string {
	return viper.GetString(KeyAPIKey)
}

func Output() string {
	return viper.GetString(KeyOutput)
}

func IsJSON() bool {
	return viper.GetBool(KeyJSON)
}

func IsYAML() bool {
	return viper.GetBool(KeyYAML)
}

func IsQuiet() bool {
	return viper.GetBool(KeyQuiet)
}

func IsDryRun() bool {
	return viper.GetBool(KeyDryRun)
}

func NoColor() bool {
	return viper.GetBool(KeyNoColor)
}

func Validate() error {
	if APIKey() == "" {
		return fmt.Errorf("API key is required: set N8N_API_KEY env var, --api-key flag, or api_key in config file")
	}
	if BaseURL() == "" {
		return fmt.Errorf("base URL is required: set N8N_BASE_URL env var, --base-url flag, or base_url in config file")
	}
	return nil
}
