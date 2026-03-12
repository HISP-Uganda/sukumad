// package config
package config

import (
	goflag "flag"
	"runtime"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	DBURL                   string        `mapstructure:"dburl"`
	Port                    int           `mapstructure:"port"`
	MigrationsDirectory     string        `mapstructure:"migrations_directory"`
	InTestMode              bool          `mapstructure:"in_test_mode"`
	Debug                   bool          `mapstructure:"debug"`
	LogLevel                string        `mapstructure:"log_level"`
	LogFormat               string        `mapstructure:"log_format"`
	RedisAddress            string        `mapstructure:"redis_address"`
	MaxConcurrent           int           `mapstructure:"max_concurrent"`
	DownstreamClientTimeout time.Duration `mapstructure:"downstream_client_timeout"`
	SeedSampleData          bool          `mapstructure:"seed_sample_data" env:"SEED_SAMPLE_DATA" env-default:"false"`

	// Claim & worker pool
	ClaimBatch       int           `mapstructure:"claim_batch"`
	ClaimInterval    time.Duration `mapstructure:"claim_interval"`
	SendWorkers      int           `mapstructure:"send_workers"`
	PollWorkers      int           `mapstructure:"poll_workers"`
	GlobalRPS        float64       `mapstructure:"global_rps"`
	GlobalBurst      int           `mapstructure:"global_burst"`
	PollAgainSeconds int           `mapstructure:"poll_again_seconds"`

	JWTSecret string        `mapstructure:"jwt_secret" env:"JWT_SECRET" env-description:"JWT HMAC secret"`
	JWTIssuer string        `mapstructure:"jwt_issuer" env:"JWT_ISSUER" env-default:"sukumad"`
	JWTExpiry time.Duration `mapstructure:"jwt_expiry" env:"JWT_EXPIRY" env-default:"24h"`

	// Account lockout
	LockoutThreshold int           `mapstructure:"lockout_threshold" env:"LOCKOUT_THRESHOLD" env-default:"5"`
	LockoutWindow    time.Duration `mapstructure:"lockout_window"    env:"LOCKOUT_WINDOW"    env-default:"15m"`
	LockoutDuration  time.Duration `mapstructure:"lockout_duration"  env:"LOCKOUT_DURATION"  env-default:"30m"`

	// Password reset
	ResetTTL time.Duration `mapstructure:"reset_ttl"         env:"RESET_TTL"         env-default:"1h"`

	// CORS
	AllowedOrigins []string `mapstructure:"allowed_origins"   env:"ALLOWED_ORIGINS"   env-separator:","`
}

var AppConfig Config

func init() {
	// ----- Choose default config file path by OS
	var configFilePath string
	switch runtime.GOOS {
	case "windows":
		configFilePath = `C:\ProgramData\sukumad\sukumad.yml`
	case "darwin", "linux":
		configFilePath = `/etc/sukumad/sukumad.yml`
	default:
		configFilePath = `./sukumad.yml`
	}

	// ----- CLI flag for config file
	configFile := flag.String("config-file", configFilePath, "Path to application YAML config file")
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	flag.Parse()

	// ----- Viper setup: YAML + ENV (SUKUMAD_*), live reload
	viper.SetConfigType("yaml")
	if len(*configFile) > 0 {
		viper.SetConfigFile(*configFile)
	} else {
		viper.SetConfigName("sukumad")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/sukumad")
	}

	// Defaults (used if not provided in YAML or env)
	viper.SetDefault("dburl", "postgres://user:password@localhost:5432/sukumad?sslmode=disable")
	viper.SetDefault("port", 8383)
	viper.SetDefault("in_test_mode", false)
	viper.SetDefault("debug", false)
	viper.SetDefault("redis_address", "127.0.0.1:6379")
	viper.SetDefault("max_concurrent", 5)
	viper.SetDefault("downstream_client_timeout", "10s")

	viper.SetDefault("claim_batch", 100)
	viper.SetDefault("claim_interval", "500ms")
	viper.SetDefault("send_workers", 8)
	viper.SetDefault("poll_workers", 4)
	viper.SetDefault("global_rps", 50.0)
	viper.SetDefault("global_burst", 50)
	viper.SetDefault("poll_again_seconds", 60)
	viper.SetDefault("jwt_secret", "dev-secret-change-me")
	viper.SetDefault("jwt_issuer", "sukumad")
	viper.SetDefault("jwt_expiry", "24h")

	// ENV OVERRIDES: SUKUMAD_DBURL, SUKUMAD_GLOBAL_RPS, etc.
	viper.SetEnvPrefix("SUKUMAD")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Read config file (if present)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// optional file; continue with defaults + env
			log.Warnf("Config file not found at %s; using env/defaults", *configFile)
		} else {
			log.Fatalf("Error reading config file %s: %v", *configFile, err)
		}
	}

	// Unmarshal to AppConfig
	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("unable to decode config into struct: %v", err)
	}

	// Watch for changes (hot reload)
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Infof("Config file changed: %s", e.Name)
		if err := viper.ReadInConfig(); err != nil {
			log.Errorf("re-read config failed: %v", err)
			return
		}
		if err := viper.Unmarshal(&AppConfig); err != nil {
			log.Errorf("unmarshal on reload failed: %v", err)
		}
	})
	viper.WatchConfig()
}
