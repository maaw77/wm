// Package config отвечает за загрузку конфигурации приложения и создание настроенных клиентов.
//
// Конфигурация читается из YAML-файла через Viper и используется для:
// - параметров HTTP-сервера (addr, таймауты);
// - параметров HTTP-клиента (timeout);
// - параметров storage (директория и имя файла состояния);
// - таймаута graceful shutdown.
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
//
// pathConfig — путь до YAML-файла конфигурации.
// Если файл не найден или не читается, используются значения по умолчанию.
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

		viper.SetDefault("storage.DataDir", "data")
		viper.SetDefault("storage.FileName", "links.json")

		return
	}

	slog.Info("Configuration loaded from file", slog.String("path", pathConfig))
}

// NewConfiguredHTTPServer создает http.Server из значений конфига.
//
// Важно: функция принимает mux и устанавливает его в поле Handler,
// чтобы сервер использовал зарегистрированные роуты.
func NewConfiguredHTTPServer(mux *http.ServeMux) http.Server {
	return http.Server{
		Addr:         viper.GetString("server.Addr"),
		Handler:      mux,
		WriteTimeout: time.Second * viper.GetDuration("server.WriteTimeout"),
		ReadTimeout:  time.Second * viper.GetDuration("server.ReadTimeout"),
		IdleTimeout:  time.Second * viper.GetDuration("server.IdleTimeout"),
	}
}

// NewConfiguredHTTPClient создает http.Client с таймаутом из конфига.
func NewConfiguredHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * viper.GetDuration("client.Timeout"),
	}
}

// GetConfiguredShutdownTimeout возвращает timeout graceful shutdown из конфига.
func GetConfiguredShutdownTimeout() time.Duration {
	return time.Second * viper.GetDuration("server.ShutdownTimeout")
}

// GetConfiguredStorageDataDir возвращает директорию для файла состояния storage из конфига.
func GetConfiguredStorageDataDir() string {
	return viper.GetString("storage.DataDir")
}

// GetConfiguredStorageFileName возвращает имя файла состояния storage из конфига.
func GetConfiguredStorageFileName() string {
	return viper.GetString("storage.FileName")
}
