package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	Driver   string
	SSLMode  string
}

type AppConfig struct {
	Name        string
	Port        string
	Version     string
	Environment string
}

type JWTConfig struct {
	PrivateKeyPath   string
	PublicKeyPath    string
	Issuer           string
	KeyID            string
	AllowedAudiences []string
}

type GoogleOIDCConfig struct {
	ClientID            string
	ClientSecret        string
	RedirectURL         string
	FrontendCallbackURL string
}

type Config struct {
	AppConfig
	DBConfig
	JWTConfig
	GoogleOIDCConfig
}

func LoadConfig() error {
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

func (c *Config) setConfig() error {
	if err := LoadConfig(); err != nil {
		return err
	}

	// AppConfig
	c.AppConfig.Name = viper.GetString("APP_NAME")
	c.AppConfig.Port = viper.GetString("APP_PORT")
	c.AppConfig.Version = viper.GetString("APP_VERSION")
	c.AppConfig.Environment = viper.GetString("APP_ENV")
	if c.AppConfig.Environment == "" {
		c.AppConfig.Environment = "development"
	}

	// DBConfig
	c.DBConfig.Host = viper.GetString("DB_HOST")
	c.DBConfig.Port = viper.GetString("DB_PORT")
	c.DBConfig.User = viper.GetString("DB_USER")
	c.DBConfig.Password = viper.GetString("DB_PASSWORD")
	c.DBConfig.Name = viper.GetString("DB_NAME")
	c.DBConfig.SSLMode = viper.GetString("DB_SSLMODE")
	c.DBConfig.Driver = viper.GetString("DB_DRIVER")
	// JWTConfig
	c.JWTConfig.PrivateKeyPath = viper.GetString("JWT_PRIVATE_KEY_PATH")
	c.JWTConfig.PublicKeyPath = viper.GetString("JWT_PUBLIC_KEY_PATH")
	c.JWTConfig.Issuer = viper.GetString("JWT_ISSUER")
	c.JWTConfig.KeyID = viper.GetString("JWT_KEY_ID")
	c.JWTConfig.AllowedAudiences = splitList(viper.GetString("JWT_ALLOWED_AUDIENCES"))

	// Google OpenID Connect
	c.GoogleOIDCConfig.ClientID = viper.GetString("GOOGLE_CLIENT_ID")
	c.GoogleOIDCConfig.ClientSecret = viper.GetString("GOOGLE_CLIENT_SECRET")
	c.GoogleOIDCConfig.RedirectURL = viper.GetString("GOOGLE_REDIRECT_URL")
	c.GoogleOIDCConfig.FrontendCallbackURL = viper.GetString("GOOGLE_FRONTEND_CALLBACK_URL")
	if c.GoogleOIDCConfig.RedirectURL == "" {
		return fmt.Errorf("required GOOGLE_FRONTEND_CALLBACK_URL value is missing")
	}

	if c.AppConfig.Name == "" || c.AppConfig.Port == "" || c.AppConfig.Version == "" {
		return fmt.Errorf("required application configuration is missing")
	}

	if c.DBConfig.Host == "" || c.DBConfig.Port == "" || c.DBConfig.User == "" || c.DBConfig.Password == "" {
		return fmt.Errorf("required database configuration is missing")
	}
	issuer, err := url.Parse(c.JWTConfig.Issuer)
	if err != nil || issuer.Scheme == "" || issuer.Host == "" || issuer.User != nil {
		return fmt.Errorf("JWT_ISSUER must be an absolute URL")
	}
	if c.JWTConfig.PrivateKeyPath == "" || c.JWTConfig.PublicKeyPath == "" || c.JWTConfig.KeyID == "" {
		return fmt.Errorf("JWT key paths and JWT_KEY_ID are required")
	}
	if len(c.JWTConfig.AllowedAudiences) == 0 {
		return fmt.Errorf("JWT_ALLOWED_AUDIENCES must contain at least one audience")
	}

	if (c.GoogleOIDCConfig.ClientID == "") != (c.GoogleOIDCConfig.ClientSecret == "") {
		return fmt.Errorf("Google OIDC client ID and client secret must both be configured")
	}

	return nil
}

func splitList(value string) []string {
	items := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func GetConfig() (*Config, error) {
	cfg := &Config{}
	err := cfg.setConfig()
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
