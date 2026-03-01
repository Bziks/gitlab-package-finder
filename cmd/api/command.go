package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"errors"
	"fmt"

	"github.com/spf13/cobra"

	cachepackagetype "github.com/bziks/gitlab-package-finder/internal/adapters/cache/packagetype"
	gitlabadapter "github.com/bziks/gitlab-package-finder/internal/adapters/gitlab"
	"github.com/bziks/gitlab-package-finder/internal/adapters/metrics"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/branch"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/packages"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/packagetype"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/project"
	"github.com/bziks/gitlab-package-finder/internal/adapters/redis/packagesearch"
	"github.com/bziks/gitlab-package-finder/internal/app"
	apihttp "github.com/bziks/gitlab-package-finder/internal/app/api/http"
	"github.com/bziks/gitlab-package-finder/internal/app/filefinder"
	"github.com/bziks/gitlab-package-finder/internal/config"
	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager/composer"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager/golang"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager/npm"
	"github.com/bziks/gitlab-package-finder/internal/services/packagesvc"
	projectservice "github.com/bziks/gitlab-package-finder/internal/services/project"
	"github.com/bziks/gitlab-package-finder/internal/services/search"
	"github.com/bziks/gitlab-package-finder/internal/telemetry"
	gocache "github.com/patrickmn/go-cache"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "api",
		Short: "api server",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			// Config
			cfg, err := config.New()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			// Logger
			slog.SetDefault(telemetry.NewLogger(cfg.Logging.Level))

			// Tracer
			tp, err := telemetry.NewTracerProvider(ctx, cfg.Tracing)
			if err != nil {
				return fmt.Errorf("init tracer: %w", err)
			}
			defer func() {
				stopCtx, stopCancel := context.WithTimeout(context.Background(), cfg.Api.ShutdownDelay)
				defer stopCancel()
				tp.Stop(stopCtx)
			}()

			// Infrastructure
			db, err := app.InitMySQL(cfg.Mysql)
			if err != nil {
				return fmt.Errorf("init mysql: %w", err)
			}
			defer db.Close()

			redisClient, err := app.InitRedis(ctx, cfg.Redis)
			if err != nil {
				return fmt.Errorf("init redis: %w", err)
			}
			defer redisClient.Close()

			gitlabClient, err := app.InitGitlab(cfg.Gitlab)
			if err != nil {
				return fmt.Errorf("init gitlab: %w", err)
			}

			memoryCache := gocache.New(cfg.Cache.TTL, cfg.Cache.Cleanup)

			m, err := metrics.NewPrometheusMetrics(cfg.Metrics)
			if err != nil {
				return fmt.Errorf("init metrics: %w", err)
			}

			// Adapters
			packageTypeRepo := cachepackagetype.NewCacheRepository(
				packagetype.NewRepository(db),
				memoryCache,
			)
			projectRepo := project.NewRepository(db)
			branchRepo := branch.NewRepository(db)
			packageRepo := packages.NewRepository(db)
			packageSearchStorage := packagesearch.New(redisClient)

			// Factories
			fileFinderFactory := filefinder.NewFactory()
			gitlabFileFinder := gitlabadapter.NewFileFinder(gitlabClient)
			fileFinderFactory.Register(entity.PackageTypeComposer, gitlabFileFinder)
			fileFinderFactory.Register(entity.PackageTypeGo, gitlabFileFinder)
			fileFinderFactory.Register(entity.PackageTypeNpm, gitlabFileFinder)

			packageManagerFactory := packagemanager.NewFactory()
			packageManagerFactory.Register(entity.PackageTypeComposer, composer.NewPackageManager())
			packageManagerFactory.Register(entity.PackageTypeGo, golang.NewPackageManager())
			packageManagerFactory.Register(entity.PackageTypeNpm, npm.NewPackageManager())

			// Services
			searchSvc := search.NewService(
				packageSearchStorage,
				projectRepo,
				packageTypeRepo,
				packageRepo,
				fileFinderFactory,
				packageManagerFactory,
			)
			packageSvc := packagesvc.NewService(packageRepo, packageSearchStorage)
			projectSvc := projectservice.NewService(projectRepo, branchRepo, packageTypeRepo)

			// API handler
			a := apihttp.NewAPI(
				packageTypeRepo,
				projectRepo,
				packageSearchStorage,
				searchSvc,
				packageSvc,
				projectSvc,
				m,
				cfg.Api.CORSAllowOrigin,
			)

			// HTTP server
			httpSrv := apihttp.NewHTTPServer(cfg.Api, a.NewRouter())

			go func() {
				if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.ErrorContext(ctx, "http server failed", "error", err)
				}
			}()

			slog.InfoContext(ctx, "API server started, press Ctrl+C to stop")

			// Graceful shutdown
			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()

			<-ctx.Done()

			slog.InfoContext(ctx, "Shutdown signal received, shutting down gracefully...")

			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Api.ShutdownDelay)
			defer cancel()

			if err := httpSrv.Shutdown(shutdownCtx); err != nil {
				slog.ErrorContext(ctx, "server forced to shutdown", "error", err)
			}

			slog.InfoContext(ctx, "Server stopped")

			return nil
		},
	}
}
