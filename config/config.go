package config

import (
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// InitConfig инициализирует конфиг из YAML через Viper.
// Если файл не найден/не читается — выставляются дефолты.
func InitConfig(pathConfig string) {
	dir, file := filepath.Split(pathConfig)
	fileName := strings.Split(file, ".")[0]

	viper.SetConfigName(fileName)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(dir)

	if err := viper.ReadInConfig(); err != nil {
		slog.Info("Default configuration is used")

		viper.SetDefault("server.Addr", "0.0.0.0:8080")
		viper.SetDefault("server.WriteTimeout", 15)
		viper.SetDefault("server.ReadTimeout", 15)
		viper.SetDefault("server.IdleTimeout", 60)
		viper.SetDefault("server.ShutdownTimeout", 15)

		viper.SetDefault("client.Timeout", 5)

		return
	}

	slog.Info("Configuration loaded from file", slog.String("path", pathConfig))
}

// NewConfiguredHTTPServer создает http.Server из конфига и обязательно
// прокидывает Handler (mux), чтобы использовались зарегистрированные роуты.
func NewConfiguredHTTPServer(mux *http.ServeMux) http.Server {
	return http.Server{
		Addr:         viper.GetString("server.Addr"),
		Handler:      mux,
		WriteTimeout: time.Second * viper.GetDuration("server.WriteTimeout"),
		ReadTimeout:  time.Second * viper.GetDuration("server.ReadTimeout"),
		IdleTimeout:  time.Second * viper.GetDuration("server.IdleTimeout"),
	}
}

// NewConfiguredHTTPClient создает http.Client из конфига.
func NewConfiguredHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * viper.GetDuration("client.Timeout"),
	}
}

func GetConfiguredShutdownTimeout() time.Duration {
	return time.Second * viper.GetDuration("server.ShutdownTimeout")
}
