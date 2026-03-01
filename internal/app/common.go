package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	gitlab "gitlab.com/gitlab-org/api/client-go"

	"github.com/bziks/gitlab-package-finder/internal/config"
)

func InitMySQL(cfg config.MysqlConfig) (*sqlx.DB, error) {
	mysqlCfg := mysql.Config{
		User:            cfg.User,
		Passwd:          cfg.Pass,
		Addr:            cfg.Host,
		DBName:          cfg.DbName,
		Net:             "tcp",
		ParseTime:       true,
		MultiStatements: true,
	}

	db, err := sqlx.Open("mysql", mysqlCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

func InitRedis(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Hosts,
		PoolSize:     cfg.PoolSize,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		MaxRetries:   cfg.ConnRetries,
	})

	pingCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return client, nil
}

func InitGitlab(cfg config.GitlabConfig) (*gitlab.Client, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("gitlab token is empty")
	}

	client, err := gitlab.NewClient(
		cfg.Token,
		gitlab.WithBaseURL(cfg.BaseURL),
		gitlab.WithHTTPClient(&http.Client{
			Timeout: cfg.HTTPTimeout,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create gitlab client: %w", err)
	}

	return client, nil
}
