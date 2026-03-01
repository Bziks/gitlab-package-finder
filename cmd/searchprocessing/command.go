package searchprocessing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	cachepackagetype "github.com/bziks/gitlab-package-finder/internal/adapters/cache/packagetype"
	"github.com/bziks/gitlab-package-finder/internal/adapters/gitlab"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/packages"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/packagetype"
	"github.com/bziks/gitlab-package-finder/internal/adapters/mysql/project"
	"github.com/bziks/gitlab-package-finder/internal/adapters/redis/packagesearch"
	"github.com/bziks/gitlab-package-finder/internal/app"
	"github.com/bziks/gitlab-package-finder/internal/app/filefinder"
	"github.com/bziks/gitlab-package-finder/internal/config"
	"github.com/bziks/gitlab-package-finder/internal/domain/entity"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager/composer"
	golangpm "github.com/bziks/gitlab-package-finder/internal/domain/packagemanager/golang"
	"github.com/bziks/gitlab-package-finder/internal/domain/packagemanager/npm"
	"github.com/bziks/gitlab-package-finder/internal/jobs"
	"github.com/bziks/gitlab-package-finder/internal/services/search"
	"github.com/bziks/gitlab-package-finder/internal/telemetry"
	"github.com/bziks/gitlab-package-finder/internal/worker"
	gocache "github.com/patrickmn/go-cache"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "search-processing",
		Short: "Process search queue",
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context())
		},
	}
}

func run(ctx context.Context) error {
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
		stopCtx, stopCancel := context.WithTimeout(context.Background(), cfg.Worker.ShutdownDelay)
		defer stopCancel()
		tp.Stop(stopCtx)
	}()

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
		return fmt.Errorf("init gitlab client: %w", err)
	}

	memoryCache := gocache.New(cfg.Cache.TTL, cfg.Cache.Cleanup)

	// Adapters
	packageTypeRepo := cachepackagetype.NewCacheRepository(
		packagetype.NewRepository(db),
		memoryCache,
	)
	projectRepo := project.NewRepository(db)
	packageRepo := packages.NewRepository(db)
	packageSearchStorage := packagesearch.New(redisClient)

	// File finder factory
	fileFinderFactory := filefinder.NewFactory()
	gitlabFileFinder := gitlab.NewFileFinder(gitlabClient)
	fileFinderFactory.Register(entity.PackageTypeComposer, gitlabFileFinder)
	fileFinderFactory.Register(entity.PackageTypeGo, gitlabFileFinder)
	fileFinderFactory.Register(entity.PackageTypeNpm, gitlabFileFinder)

	// Package manager factory
	packageManagerFactory := packagemanager.NewFactory()
	packageManagerFactory.Register(entity.PackageTypeComposer, composer.NewPackageManager())
	packageManagerFactory.Register(entity.PackageTypeGo, golangpm.NewPackageManager())
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

	// Job
	job := jobs.NewSearchProcessingJob(packageSearchStorage, searchSvc)

	// Worker
	w := worker.NewWorker(worker.Config{
		Job:           job,
		ShutdownDelay: cfg.Worker.ShutdownDelay,
		Logger:        slog.Default(),
	})

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	err = w.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("worker: %w", err)
	}

	slog.InfoContext(ctx, "search processing shutdown complete")

	return nil
}
