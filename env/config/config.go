package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config centraliza toda la configuración del ecosistema bank-api.
// Cada servicio (gateway, users, transactions, crons) carga este Config
// y usa solo los campos que necesita.
type Config struct {
	// --- Comunes ---
	Port     int
	Env      string // development | staging | production
	LogLevel string
	Service  string // nombre del servicio para logging y sigil (ej: "users")

	// --- Base de datos ---
	DatabaseURL string // postgres DSN

	// --- Auth ---
	JWTSecret    string
	AuthRequired bool // false → JWT opcional (útil para evaluadores sin el srv users)

	// --- Cifrado de datos en reposo ---
	EncryptionKey string // Hex-encoded 32 bytes key
	HMACKey       string // Hex-encoded HMAC key

	// --- Redis ---
	RedisURL string // ej: localhost:6379 o redis:6379

	// --- Kafka ---
	KafkaBrokers string // ej: localhost:9092 o kafka:29092
	KafkaTopic   string // ej: transaction-completed

	// --- Sigil (autenticación S2S) ---
	SigilSecret     string // secreto HMAC compartido
	AllowedServices string // servicios permitidos, separados por coma (ej: "gateway,crons")

	// --- URLs de servicios ---
	// Gateway (proxy externo): solo la base, sin path
	UsersURL        string // ej: http://users-svc:8081
	TransactionsURL string // ej: http://transactions-svc:8082

	// S2S interno (path completo incluido en la variable): cambia sin tocar código
	UsersInternalURL        string // ej: http://users-svc:8081/internal/users
	TransactionsInternalURL string // ej: http://transactions-svc:8082/internal/transactions
	ProviderExecuteURL      string // ej: http://provider-svc:8083/provider/v1/execute

	// --- Gateway ---
	AllowedOrigins string // orígenes CORS permitidos, separados por coma

	// --- Pool de conexiones (opt/db) ---
	MaxPoolConnections    int
	MaxIdleConnections    int
	ConnectionMaxLifetime int // segundos

	// --- Multi-tenant (opt/db pool manager) ---
	TenantDatabaseURITemplate string // Template: postgres://.../{slug}?...
}

// Load lee la configuración desde variables de entorno.
// Llama a godotenv.Load() para soportar archivos .env en desarrollo.
// No valida campos requeridos: cada servicio valida lo que necesita con Validate().
func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("PORT inválido: %w", err)
	}

	maxPoolConns, err := strconv.Atoi(getEnv("MAX_POOL_CONNECTIONS", "10"))
	if err != nil {
		return nil, fmt.Errorf("MAX_POOL_CONNECTIONS inválido: %w", err)
	}

	maxIdleConns, err := strconv.Atoi(getEnv("MAX_IDLE_CONNECTIONS", "5"))
	if err != nil {
		return nil, fmt.Errorf("MAX_IDLE_CONNECTIONS inválido: %w", err)
	}

	connMaxLifetime, err := strconv.Atoi(getEnv("CONNECTION_MAX_LIFETIME", "3600"))
	if err != nil {
		return nil, fmt.Errorf("CONNECTION_MAX_LIFETIME inválido: %w", err)
	}

	return &Config{
		Port:     port,
		Env:      getEnv("ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Service:  getEnv("SERVICE_NAME", "bank-api"),

		DatabaseURL: getEnv("DATABASE_URL", ""),

		JWTSecret:    getEnv("JWT_SECRET", ""),
		AuthRequired: getEnv("AUTH_REQUIRED", "true") == "true",

		EncryptionKey: getEnv("ENCRYPTION_KEY", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"), // Default 32 bytes hex key
		HMACKey:       getEnv("HMAC_KEY", "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"),       // Default 32 bytes hex key

		RedisURL: getEnv("REDIS_URL", "localhost:6379"),

		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:   getEnv("KAFKA_TOPIC", "transaction-completed"),

		SigilSecret:     getEnv("SIGIL_SECRET", ""),
		AllowedServices: getEnv("ALLOWED_SERVICES", "gateway"),

		UsersURL:        getEnv("USERS_URL", "http://localhost:8081"),
		TransactionsURL: getEnv("TRANSACTIONS_URL", "http://localhost:8082"),

		UsersInternalURL:        getEnv("USERS_INTERNAL_URL", "http://localhost:8081/internal/users"),
		TransactionsInternalURL: getEnv("TRANSACTIONS_INTERNAL_URL", "http://localhost:8082/internal/transactions"),
		ProviderExecuteURL:      getEnv("PROVIDER_EXECUTE_URL", "http://localhost:8083/provider/v1/execute"),

		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),

		MaxPoolConnections:        maxPoolConns,
		MaxIdleConnections:        maxIdleConns,
		ConnectionMaxLifetime:     connMaxLifetime,
		TenantDatabaseURITemplate: getEnv("TENANT_DATABASE_URI_TEMPLATE", ""),
	}, nil
}

// Validate verifica que los campos requeridos no estén vacíos.
// Cada servicio llama a Validate con los campos que necesita.
// Ejemplo: cfg.Validate("DatabaseURL", "JWTSecret", "SigilSecret")
func (c *Config) Validate(fields ...string) error {
	values := map[string]string{
		"DatabaseURL":             c.DatabaseURL,
		"JWTSecret":               c.JWTSecret,
		"SigilSecret":             c.SigilSecret,
		"UsersURL":                c.UsersURL,
		"TransactionsURL":         c.TransactionsURL,
		"UsersInternalURL":        c.UsersInternalURL,
		"TransactionsInternalURL": c.TransactionsInternalURL,
		"ProviderExecuteURL":      c.ProviderExecuteURL,
		"AllowedOrigins":          c.AllowedOrigins,
		"EncryptionKey":           c.EncryptionKey,
		"HMACKey":                 c.HMACKey,
		"RedisURL":                c.RedisURL,
		"KafkaBrokers":            c.KafkaBrokers,
		"KafkaTopic":              c.KafkaTopic,
	}

	for _, field := range fields {
		if val, ok := values[field]; !ok || val == "" {
			return fmt.Errorf("campo requerido faltante o vacío: %s", field)
		}
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
