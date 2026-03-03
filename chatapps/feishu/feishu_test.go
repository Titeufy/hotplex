package feishu

import (
	"log/slog"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid_config",
			config: &Config{
				AppID:             "test_app_id",
				AppSecret:         "test_app_secret",
				VerificationToken: "test_token",
				EncryptKey:        "test_encrypt_key",
			},
			wantErr: false,
		},
		{
			name: "missing_app_id",
			config: &Config{
				AppSecret:         "test_app_secret",
				VerificationToken: "test_token",
				EncryptKey:        "test_encrypt_key",
			},
			wantErr: true,
		},
		{
			name: "missing_app_secret",
			config: &Config{
				AppID:             "test_app_id",
				VerificationToken: "test_token",
				EncryptKey:        "test_encrypt_key",
			},
			wantErr: true,
		},
		{
			name: "missing_verification_token",
			config: &Config{
				AppID:      "test_app_id",
				AppSecret:  "test_app_secret",
				EncryptKey: "test_encrypt_key",
			},
			wantErr: true,
		},
		{
			name: "missing_encrypt_key",
			config: &Config{
				AppID:             "test_app_id",
				AppSecret:         "test_app_secret",
				VerificationToken: "test_token",
			},
			wantErr: true,
		},
		{
			name: "default_server_addr",
			config: &Config{
				AppID:             "test_app_id",
				AppSecret:         "test_app_secret",
				VerificationToken: "test_token",
				EncryptKey:        "test_encrypt_key",
				ServerAddr:        "",
			},
			wantErr: false,
		},
		{
			name: "default_max_message_len",
			config: &Config{
				AppID:             "test_app_id",
				AppSecret:         "test_app_secret",
				VerificationToken: "test_token",
				EncryptKey:        "test_encrypt_key",
				MaxMessageLen:     0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.config.ServerAddr == "" {
				t.Error("Config.Validate() should set default ServerAddr")
			}
			if !tt.wantErr && tt.config.MaxMessageLen == 0 {
				t.Error("Config.Validate() should set default MaxMessageLen")
			}
		})
	}
}

func TestNewAdapter(t *testing.T) {
	logger := slog.Default()

	// Test with valid config
	config := &Config{
		AppID:             "test_app_id",
		AppSecret:         "test_app_secret",
		VerificationToken: "test_token",
		EncryptKey:        "test_encrypt_key",
	}

	adapter, err := NewAdapter(config, logger)
	if err != nil {
		t.Fatalf("NewAdapter() error = %v", err)
	}

	if adapter == nil {
		t.Fatal("NewAdapter() returned nil adapter")
	}

	if adapter.Platform() != "feishu" {
		t.Errorf("Adapter.Platform() = %v, want 'feishu'", adapter.Platform())
	}

	if adapter.client == nil {
		t.Error("Adapter.client should not be nil")
	}
}

func TestNewAdapter_InvalidConfig(t *testing.T) {
	logger := slog.Default()

	// Test with invalid config (missing AppID)
	config := &Config{
		AppSecret:         "test_app_secret",
		VerificationToken: "test_token",
		EncryptKey:        "test_encrypt_key",
	}

	adapter, err := NewAdapter(config, logger)
	if err == nil {
		t.Error("NewAdapter() should return error for invalid config")
	}
	if adapter != nil {
		t.Error("NewAdapter() should return nil adapter for invalid config")
	}
}
