package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Api          ApiConfig
	Worker       WorkerConfig
	Logging      LoggingConfig
	Tracing      TracingConfig
	Metrics      MetricsConfig
	Mysql        MysqlConfig
	Redis        RedisConfig
	Cache        CacheConfig
	Gitlab       GitlabConfig
	ProjectsSync ProjectsSyncConfig
}

type LoggingConfig struct {
	Level slog.Level `envconfig:"LOGGING_LEVEL" default:"info"`
}

type TracingConfig struct {
	Enabled    bool    `envconfig:"TRACING_ENABLED" default:"false"`
	SampleRate float64 `envconfig:"TRACING_SAMPLE_RATE" default:"0.0"`
}

type ProjectsSyncConfig struct {
	Interval time.Duration `envconfig:"PROJECTS_SYNC_INTERVAL" default:"1h"`
}

type ApiConfig struct {
	Port            uint          `envconfig:"HTTP_PORT" default:"58080"`
	ReadTimeout     time.Duration `envconfig:"HTTP_READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `envconfig:"HTTP_WRITE_TIMEOUT" default:"30s"`
	IdleTimeout     time.Duration `envconfig:"HTTP_IDLE_TIMEOUT" default:"60s"`
	ShutdownDelay   time.Duration `envconfig:"API_SHUTDOWN_DELAY" default:"10s"`
	CORSAllowOrigin string        `envconfig:"CORS_ALLOW_ORIGIN" default:"*"`
}

type WorkerConfig struct {
	ShutdownDelay time.Duration `envconfig:"WORKER_SHUTDOWN_DELAY" default:"10s"`
}

type MetricsConfig struct {
	Name      string `envconfig:"OTEL_SERVICE_NAME" default:"gitlab_package_finder"`
	Version   string `envconfig:"OTEL_VERSION" default:"0.1.0"`
	Namespace string `envconfig:"OTEL_NAMESPACE" default:"gpf"`
}

type MysqlConfig struct {
	Host           string `envconfig:"DB_HOST" required:"true"`
	User           string `envconfig:"DB_USER" required:"true"`
	Pass           string `envconfig:"DB_PASS" required:"true"`
	DbName         string `envconfig:"DB_NAME" required:"true"`
	MigrationsPath string `envconfig:"DB_MIGRATIONS_PATH" default:"/app/migrations"`
}

type RedisConfig struct {
	Hosts        string        `envconfig:"REDIS_HOSTS" required:"true"`
	PoolSize     int           `envconfig:"REDIS_POOL_SIZE" default:"10"`
	ReadTimeout  time.Duration `envconfig:"REDIS_READ_TIMEOUT" default:"3s"`
	WriteTimeout time.Duration `envconfig:"REDIS_WRITE_TIMEOUT" default:"3s"`
	ConnRetries  int           `envconfig:"REDIS_CONN_RETRIES" default:"5"`
}

type CacheConfig struct {
	TTL           time.Duration `envconfig:"CACHE_TTL" default:"10m"`
	SessionKeyTTL time.Duration `envconfig:"SESSION_KEY_TTL" default:"8m"`
	Cleanup       time.Duration `envconfig:"CACHE_CLEANUP" default:"10m"`
}

type GitlabConfig struct {
	Token       string        `envconfig:"GITLAB_TOKEN" required:"true"`
	BaseURL     string        `envconfig:"GITLAB_BASE_URL" required:"true"`
	HTTPTimeout time.Duration `envconfig:"GITLAB_HTTP_TIMEOUT" default:"10s"`
}

func New() (Config, error) {
	cfg := Config{}

	if err := envconfig.Process("", &cfg); err != nil {
		return cfg, fmt.Errorf("cannot process the config: %w", err)
	}

	return cfg, nil
}
