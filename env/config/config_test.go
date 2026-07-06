package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
	}{
		{
			name: "Success load defaults",
			envVars: map[string]string{
				"PORT": "8000",
			},
			wantErr: false,
		},
		{
			name: "Invalid PORT",
			envVars: map[string]string{
				"PORT": "invalid",
			},
			wantErr: true,
		},
		{
			name: "Invalid MAX_POOL_CONNECTIONS",
			envVars: map[string]string{
				"PORT":                 "8000",
				"MAX_POOL_CONNECTIONS": "invalid",
			},
			wantErr: true,
		},
		{
			name: "Invalid MAX_IDLE_CONNECTIONS",
			envVars: map[string]string{
				"PORT":                 "8000",
				"MAX_POOL_CONNECTIONS": "10",
				"MAX_IDLE_CONNECTIONS": "invalid",
			},
			wantErr: true,
		},
		{
			name: "Invalid CONNECTION_MAX_LIFETIME",
			envVars: map[string]string{
				"PORT":                    "8000",
				"MAX_POOL_CONNECTIONS":    "10",
				"MAX_IDLE_CONNECTIONS":    "5",
				"CONNECTION_MAX_LIFETIME": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			cfg, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && cfg == nil {
				t.Errorf("Load() returned nil config")
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		fields  []string
		wantErr bool
	}{
		{
			name: "Success validate all fields exist",
			cfg: &Config{
				JWTSecret: "secret",
				RedisURL:  "localhost:6379",
			},
			fields:  []string{"JWTSecret", "RedisURL"},
			wantErr: false,
		},
		{
			name: "Missing field JWTSecret",
			cfg: &Config{
				RedisURL: "localhost:6379",
			},
			fields:  []string{"JWTSecret", "RedisURL"},
			wantErr: true,
		},
		{
			name: "Missing field DatabaseURL",
			cfg: &Config{
				JWTSecret: "secret",
			},
			fields:  []string{"DatabaseURL"},
			wantErr: true,
		},
		{
			name: "Field not supported in switch",
			cfg: &Config{
				JWTSecret: "secret",
			},
			fields:  []string{"UnknownField"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate(tt.fields...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
