package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	systemV1 "spark/api/system/v1"
	"spark/internal/conf"
	internalMiddleware "spark/internal/middleware"
	"spark/internal/service/system"
	"spark/pkg/clog"
	"spark/pkg/kvstore"
	"spark/pkg/middleware"
	"spark/pkg/utils/oss"

	"github.com/casbin/casbin/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware/validate"
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"gorm.io/gorm"
)

// NewHTTPServer creates a new HTTP server with example endpoints
func NewHTTPServer(
	c *conf.Bootstrap,
	logger clog.Logger,
	db *gorm.DB,
	kv kvstore.Store,
	casbinEnforcer *casbin.Enforcer,

	// System services
	auth *system.AuthService,
	configService *system.ConfigService,
	dept *system.DeptService,
	log *system.LogService,
	menu *system.MenuService,
	post *system.PostService,
	user *system.UserService,
	dict *system.DictService,
	role *system.RoleService,
	ossProvider oss.OssProvider,
	file *system.FileService,
	announcement *system.AnnouncementService,
	message *system.MessageService,
	oauth *system.OAuthService,
	packageService *system.PackageService,
	userOnline *system.UserOnlineService,

) *kratosHttp.Server {

	// Define paths that don't require authentication
	unAuthPaths := []string{
		"/api.system.v1.Auth/Login",
		"/api.system.v1.OAuth/Login",
		"/api.biz.v1.Client/ClientLogin", // Desktop client login
		"/openapi.yaml",                  // OpenAPI documentation endpoint (dev mode)
		"/healthz",
		"/readyz",
		"/metrics",
		// Add other public endpoints here
	}

	// Create auth middleware options
	authOpts := middleware.AuthOption{
		AccessSecret: c.Auth.AccessSecret,
		AccessExpire: c.Auth.AccessExpire.AsDuration(),
		UserService:  user,
		RoleService:  role,
		KV:           kv,
		Casbin:       casbinEnforcer,
		Logger:       logger,
	}

	var opts = []kratosHttp.ServerOption{
		kratosHttp.Middleware(
			// 1. No auth middleware chain (for login and public endpoints)
			selector.Server(
				recovery.Recovery(),
				tracing.Server(),
				internalMiddleware.Locale(),
				middleware.TraceHeader(),
				middleware.MetricsServer(),
				middleware.Server(logger),
				validate.Validator(),
			).
				Path(unAuthPaths...).
				Build(),

			// 2. Standard auth middleware chain (for all other endpoints requiring auth)
			selector.Server(
				recovery.Recovery(),
				tracing.Server(),
				internalMiddleware.Locale(),
				middleware.TraceHeader(),
				middleware.MetricsServer(),
				middleware.Server(logger),
				middleware.AuthInterceptorMiddleware(authOpts),
				internalMiddleware.OperationLog(log),
				validate.Validator(),
			).
				Match(func(ctx context.Context, operation string) bool {
					// Exclude paths that don't require auth
					for _, path := range unAuthPaths {
						if operation == path {
							return false
						}
					}
					return true
				}).
				Build(),
		),
		kratosHttp.ErrorEncoder(DefaultErrorEncoder),
		kratosHttp.ResponseEncoder(DefaultResponseEncoder),
	}

	// Configure server options from config
	if c.Server != nil && c.Server.Http != nil {
		if c.Server.Http.Network != "" {
			opts = append(opts, kratosHttp.Network(c.Server.Http.Network))
		}
		if c.Server.Http.Addr != "" {
			opts = append(opts, kratosHttp.Address(c.Server.Http.Addr))
		}
		if c.Server.Http.Timeout != nil {
			opts = append(opts, kratosHttp.Timeout(c.Server.Http.Timeout.AsDuration()))
		}
	} else {
		// Default configuration
		opts = append(opts, kratosHttp.Address(":8000"))
	}

	// Initialize OSS service
	if c.Oss != nil && c.Oss.Service != "" {
		ctx := context.Background()
		err := ossProvider.Update(ctx, c.Oss)
		if err != nil {
			logger.Warn(context.Background(), "OSS service initialization failed, file upload will return error", clog.Err(err))
		} else {
			logger.Info(context.Background(), "OSS service initialized", clog.String("service", c.Oss.Service))
		}
	} else {
		logger.Info(context.Background(), "OSS service not configured, file upload disabled")
	}

	srv := kratosHttp.NewServer(opts...)

	registerHealthEndpoints(srv, c, db, kv, logger)
	registerMetricsEndpoint(srv)

	// Register system services
	systemV1.RegisterAuthHTTPServer(srv, auth)
	systemV1.RegisterConfigHTTPServer(srv, configService)
	systemV1.RegisterDeptHTTPServer(srv, dept)
	systemV1.RegisterLogHTTPServer(srv, log)
	systemV1.RegisterMenuHTTPServer(srv, menu)
	systemV1.RegisterPostHTTPServer(srv, post)
	systemV1.RegisterUserHTTPServer(srv, user)
	systemV1.RegisterDictHTTPServer(srv, dict)
	systemV1.RegisterRoleHTTPServer(srv, role)
	systemV1.RegisterFileHTTPServer(srv, file)
	systemV1.RegisterAnnouncementHTTPServer(srv, announcement)
	systemV1.RegisterMessageHTTPServer(srv, message)
	systemV1.RegisterOAuthHTTPServer(srv, oauth)
	systemV1.RegisterPackageHTTPServer(srv, packageService)
	systemV1.RegisterUserOnlineHTTPServer(srv, userOnline)

	// Register OpenAPI endpoint if dev mode is enabled
	if c.Dev != nil && c.Dev.EnableOpenapi {
		registerOpenAPIEndpoint(srv, c, logger)
	}

	return srv
}

type readyzResult struct {
	Ok     bool              `json:"ok"`
	Checks map[string]string `json:"checks"`
}

func registerHealthEndpoints(
	srv *kratosHttp.Server,
	c *conf.Bootstrap,
	db *gorm.DB,
	kv kvstore.Store,
	logger clog.Logger,
) {
	srv.HandleFunc("/healthz", func(w kratosHttp.ResponseWriter, r *kratosHttp.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv.HandleFunc("/readyz", func(w kratosHttp.ResponseWriter, r *kratosHttp.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		res := readyzResult{
			Ok:     true,
			Checks: map[string]string{},
		}

		// DB
		if db == nil {
			res.Ok = false
			res.Checks["postgres"] = "db is nil"
		} else if sqlDB, err := db.DB(); err != nil {
			res.Ok = false
			res.Checks["postgres"] = fmtS("get sql db: %v", err)
		} else if err := pingSQL(ctx, sqlDB); err != nil {
			res.Ok = false
			res.Checks["postgres"] = fmtS("ping: %v", err)
		} else {
			res.Checks["postgres"] = "ok"
		}

		// Redis (KV store)
		if kv == nil {
			res.Ok = false
			res.Checks["redis"] = "kv store is nil"
		} else if ok, detail := kv.Health(ctx); !ok {
			res.Ok = false
			res.Checks["redis"] = detail
		} else {
			res.Checks["redis"] = fmtS("ok (%s)", kv.Backend())
		}

		// Response
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if !res.Ok {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		b, _ := json.Marshal(res)
		_, _ = w.Write(b)

		if !res.Ok {
			logger.Warn(ctx, "readyz failed", clog.Any("checks", res.Checks))
		}
	})
}

func pingSQL(ctx context.Context, db *sql.DB) error {
	return db.PingContext(ctx)
}

func fmtS(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}

// registerOpenAPIEndpoint registers the OpenAPI documentation endpoint
func registerOpenAPIEndpoint(srv *kratosHttp.Server, c *conf.Bootstrap, logger clog.Logger) {
	openapiPath := c.Dev.OpenapiPath
	if openapiPath == "" {
		openapiPath = "./openapi.yaml"
	}

	srv.HandleFunc("/openapi.yaml", func(w kratosHttp.ResponseWriter, r *kratosHttp.Request) {
		data, err := os.ReadFile(openapiPath)
		if err != nil {
			logger.Error(r.Context(), "Failed to read OpenAPI file", clog.Err(err), clog.String("path", openapiPath))
			w.WriteHeader(404)
			w.Write([]byte("OpenAPI file not found"))
			return
		}

		w.Header().Set("Content-Type", "application/x-yaml")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(200)
		w.Write(data)
	})

	logger.Info(context.Background(), "OpenAPI endpoint registered", clog.String("path", "/openapi.yaml"), clog.String("file", openapiPath))
}
