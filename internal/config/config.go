package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Log      LogConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type AuthConfig struct {
	JWTSecret  string
	TokenTTL   time.Duration
	RefreshTTL time.Duration
}

type LogConfig struct {
	Level  string
	Format string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.SSLMode,
	)
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")

	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("database.host", "127.0.0.1")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 5)
	viper.SetDefault("database.conn_max_lifetime", "5m")
	viper.SetDefault("auth.token_ttl", "24h")
	viper.SetDefault("auth.refresh_ttl", "168h")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	connLifetime, err := time.ParseDuration(viper.GetString("database.conn_max_lifetime"))
	if err != nil {
		connLifetime = 5 * time.Minute
	}
	tokenTTL, err := time.ParseDuration(viper.GetString("auth.token_ttl"))
	if err != nil {
		tokenTTL = 24 * time.Hour
	}
	refreshTTL, err := time.ParseDuration(viper.GetString("auth.refresh_ttl"))
	if err != nil {
		refreshTTL = 168 * time.Hour
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: viper.GetString("server.host"),
			Port: viper.GetInt("server.port"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("database.host"),
			Port:            viper.GetInt("database.port"),
			User:            viper.GetString("database.user"),
			Password:        viper.GetString("database.password"),
			DBName:          viper.GetString("database.dbname"),
			SSLMode:         viper.GetString("database.sslmode"),
			MaxOpenConns:    viper.GetInt("database.max_open_conns"),
			MaxIdleConns:    viper.GetInt("database.max_idle_conns"),
			ConnMaxLifetime: connLifetime,
		},
		Auth: AuthConfig{
			JWTSecret:  viper.GetString("auth.jwt_secret"),
			TokenTTL:   tokenTTL,
			RefreshTTL: refreshTTL,
		},
		Log: LogConfig{
			Level:  viper.GetString("log.level"),
			Format: viper.GetString("log.format"),
		},
	}

	return cfg, nil
}
