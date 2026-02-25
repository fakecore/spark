package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"spark/internal/biz"
	"spark/internal/conf"
	"spark/internal/data"
	"spark/internal/i18n"
	"spark/internal/server"
	"spark/internal/service"
	"spark/pkg/clog"
	configpkg "spark/pkg/config"
	"spark/pkg/config/watcher"
	"spark/pkg/trace"
	"spark/pkg/utils"
	"spark/pkg/utils/oss"
	"spark/pkg/utils/provider"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "spark"
	// Version is the version of the compiled software.
	Version string = "v1.0.0"
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../docker/backend/config/config.yaml", "config path, eg: -conf config.yaml")
}

// convertLogConfig converts conf.Server_Log to clog.Config
func convertLogConfig(logCfg *conf.Server_Log) *clog.Config {
	if logCfg == nil {
		return &clog.Config{
			Level:             "info",
			Format:            "json",
			DisableCaller:     false,
			DisableStacktrace: false,
			OutputPaths:       []string{"stdout"},
			ErrorOutputPaths:  []string{"stderr"},
		}
	}
	rotation := &clog.RotationConfig{}
	if logCfg.Rotation != nil {
		rotation = &clog.RotationConfig{
			MaxSize:    int(logCfg.Rotation.MaxSize),
			MaxBackups: int(logCfg.Rotation.MaxBackups),
			MaxAge:     int(logCfg.Rotation.MaxAge),
			Compress:   logCfg.Rotation.Compress,
		}
	}
	return &clog.Config{
		Level:             logCfg.Level,
		Format:            logCfg.Format,
		DisableCaller:     logCfg.DisableCaller,
		DisableStacktrace: logCfg.DisableStacktrace,
		OutputPaths:       logCfg.OutputPaths,
		ErrorOutputPaths:  logCfg.ErrorOutputPaths,
		Rotation:          rotation,
	}
}

func main() {
	flag.Parse()

	// Bootstrap phase - no logger needed here

	// Load .env file if it exists
	if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			// Log warning but continue - .env file is optional
			log.Infof("Warning: could not load env file %s: %v", envFile, err)
		}
	} else {
		// Try to load default .env files
		_ = godotenv.Load(".env.local")      // Highest priority
		_ = godotenv.Load(".env.production") // Production specific
		_ = godotenv.Load(".env")            // Default env file
	}

	// Load configuration from file
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	if err := configpkg.ValidateBootstrap(&bc); err != nil {
		panic(fmt.Errorf("invalid bootstrap config: %w", err))
	}

	// Initialize i18n translator
	if err := i18n.InitGlobalTranslator(); err != nil {
		panic(fmt.Errorf("failed to init i18n translator: %w", err))
	}

	// Start application with fx
	app := fx.New(
		fx.WithLogger(func(config *conf.Bootstrap) fxevent.Logger {
			clogCfg := convertLogConfig(config.Server.Log)
			l, err := clog.NewFxLoggerAdapterWithCallerSkip(clogCfg, 3)
			if err != nil {
				panic(err)
			}
			return l
		}),

		fx.Provide(
			func() *conf.Bootstrap { return &bc },
			// Initialize tracer
			func(bc *conf.Bootstrap) func() {
				tr := &trace.Config{
					ServiceName:    "spark",
					ServiceVersion: Version,
					SamplingRate:   1.0,
					Enabled:        true,
				}
				if bc != nil && bc.Server != nil && bc.Server.Tracing != nil {
					tr.Enabled = bc.Server.Tracing.Enabled
					if bc.Server.Tracing.Otel != nil {
						if bc.Server.Tracing.Otel.ServiceName != "" {
							tr.ServiceName = bc.Server.Tracing.Otel.ServiceName
						} else if bc.Server.Name != "" {
							tr.ServiceName = bc.Server.Name
						}
						if bc.Server.Tracing.Otel.ServiceVersion != "" {
							tr.ServiceVersion = bc.Server.Tracing.Otel.ServiceVersion
						}
						tr.Endpoint = bc.Server.Tracing.Otel.Endpoint
						tr.ResourceAttributes = bc.Server.Tracing.Otel.ResourceAttributes
					}
				}
				shutdown, err := trace.InitTracer(tr)
				if err != nil {
					panic(fmt.Errorf("failed to init tracer: %w", err))
				}
				return shutdown
			},
		),

		// Include all modules
		data.Module,
		biz.Module,
		service.Module,
		server.Module,
		utils.Module,

		// Lifecycle management for server and tracer
		fx.Invoke(func(
			lc fx.Lifecycle,
			srv *server.AppServer,
			tracerShutdown func(),
			logger clog.Logger,
			bc *conf.Bootstrap,
			ossProv oss.OssProvider,
			dbProv provider.DatabaseProvider,
			kvProv provider.KVStoreProvider,
		) {
			watcher := watcher.NewConfigWatcher(
				logger,
				flagconf,
				bc,
				ossProv,
				dbProv,
				kvProv,
			)

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					logger.Info(ctx, "Starting servers...")
					go func() {
						if err := srv.Start(); err != nil {
							panic(fmt.Errorf("failed to start servers: %w", err))
						}
					}()

					if err := watcher.Start(ctx); err != nil {
						panic(fmt.Errorf("failed to start config watcher: %w", err))
					}
					logger.Info(ctx, "config watcher started")
					return nil
				},
				OnStop: func(ctx context.Context) error {
					logger.Info(ctx, "Stopping HTTP server...")
					if err := srv.Stop(ctx); err != nil {
						logger.Error(ctx, "Failed to stop HTTP server", clog.Err(err))
					}
					logger.Info(ctx, "Stopping config watcher...")
					if err := watcher.Stop(ctx); err != nil {
						logger.Error(ctx, "Failed to stop config watcher", clog.Err(err))
					}
					logger.Info(ctx, "Stopping tracer...")
					tracerShutdown()
					return nil
				},
			})
		}),
	)

	// Create context for graceful shutdown
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		log.Fatalf("Failed to start application: %v", err)
	}

	// Wait for termination signal
	<-app.Done()

	// Stop the application
	if err := app.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop application: %v", err)
	}
}
