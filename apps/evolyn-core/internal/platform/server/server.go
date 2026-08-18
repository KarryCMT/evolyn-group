package server

import (
	"context"
	"evolyn/internal/platform/httpx"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"evolyn/internal/config"
	"evolyn/internal/infrastructure"
	"evolyn/internal/platform/auth"
	authcontroller "evolyn/internal/platform/auth/controller"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/controller"
	"evolyn/internal/platform/iam/authorization"
	iamcontroller "evolyn/internal/platform/iam/controller"
	"evolyn/internal/platform/iam/repository"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/platform/middleware"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	"evolyn/internal/utils/request"
	"evolyn/internal/utils/set"
	"evolyn/internal/version"

	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func New(conf *config.Config, logger *logrus.Logger) (*Server, error) {
	rateLimitMiddleware, err := middleware.RateLimitMiddleware(conf.Server.LimitConfigs)
	if err != nil {
		return nil, err
	}

	db, err := infrastructure.NewPostgres(&conf.DB)
	if err != nil {
		return nil, errors.Wrap(err, "db init failed")
	}

	rdb, err := infrastructure.NewRedisClient(&conf.Redis)
	if err != nil {
		return nil, errors.Wrap(err, "redis client failed")
	}

	// 域仓储装配（ADR-007 域模块化）：tenant 先于 iam 迁移，
	// 保证业务模型加 tenant_id 列时默认租户已就绪
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	iamRepo := repository.NewRepositories(db, rdb)
	if conf.DB.Migrate {
		if err := tenantRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := iamRepo.Migrate(); err != nil {
			return nil, err
		}
	}

	// 种子：默认租户最先（单租户/存量数据归属兜底），再 iam 资源与系统分组
	if err := tenantRepo.SeedDefaultTenant(); err != nil {
		return nil, err
	}
	if err := iamRepo.Init(); err != nil {
		return nil, err
	}

	accountService := service.NewAccountService(iamRepo.Account(), iamRepo.User(), tenantRepo)
	userService := service.NewUserService(iamRepo.User())
	groupService := service.NewGroupService(iamRepo.Group(), iamRepo.User())
	jwtService := auth.NewJWTService(conf.Server.JWTSecret)
	rbacService := service.NewRBACService(iamRepo.RBAC())
	oauthManager := oauth.NewOAuthManager(conf.OAuthConfig)

	userController := iamcontroller.NewUserController(userService)
	groupController := iamcontroller.NewGroupController(groupService)
	authController := authcontroller.NewAuthController(accountService, jwtService, oauthManager)
	rbacController := iamcontroller.NewRbacController(rbacService)

	// 鉴权器显式注入 iam 仓储（P0-4：拆除全局单例）
	authorizer := authorization.NewAuthorizer(iamRepo.User(), iamRepo.Group())

	controllers := []controller.Controller{userController, groupController, authController, rbacController}

	gin.SetMode(conf.Server.ENV)

	e := gin.New()
	e.Use(
		gin.Recovery(),
		rateLimitMiddleware,
		middleware.MonitorMiddleware(),
		middleware.CORSMiddleware(),
		middleware.RequestInfoMiddleware(&request.RequestInfoFactory{APIPrefixes: set.NewString("api")}),
		middleware.LogMiddleware(logger, "/"),
		middleware.AuthenticationMiddleware(jwtService, iamRepo.User()),
		middleware.TenantMiddleware(),
		middleware.AuthorizationMiddleware(authorizer),
		middleware.TraceMiddleware(),
	)

	return &Server{
		engine:      e,
		config:      conf,
		logger:      logger,
		db:          db,
		rdb:         rdb,
		controllers: controllers,
	}, nil
}

type Server struct {
	engine *gin.Engine
	config *config.Config
	logger *logrus.Logger

	// 存活探测与关闭由 infrastructure 直接承担（域仓储不再聚合生命周期）
	db  *gorm.DB
	rdb *infrastructure.RedisDB

	controllers []controller.Controller
}

// graceful shutdown
func (s *Server) Run() error {
	defer s.Close()

	s.initRouter()

	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)
	s.logger.Infof("Start server on: %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start server, %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.config.Server.GracefulShutdownPeriod)*time.Second)
	defer cancel()

	ch := <-sig
	s.logger.Infof("Receive signal: %s", ch)

	return server.Shutdown(ctx)
}

func (s *Server) Close() {
	if err := infrastructure.Close(s.db, s.rdb); err != nil {
		s.logger.Warnf("failed to close db/redis, %v", err)
	}

}

func (s *Server) initRouter() {
	root := s.engine

	// register non-resource routers
	root.GET("/", httpx.WrapFunc(s.getRoutes))
	root.GET("/index", controller.Index)
	root.GET("/healthz", httpx.WrapFunc(s.Ping))
	root.GET("/version", httpx.WrapFunc(version.Get))
	root.GET("/metrics", gin.WrapH(promhttp.Handler()))
	root.Any("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
	if gin.Mode() != gin.ReleaseMode {
		root.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}

	api := root.Group("/api/v1")
	controllers := make([]string, 0, len(s.controllers))
	for _, router := range s.controllers {
		router.RegisterRoute(api)
		controllers = append(controllers, router.Name())
	}
	logrus.Infof("server enabled controllers: %v", controllers)
}

func (s *Server) getRoutes() []string {
	paths := set.NewString()
	for _, r := range s.engine.Routes() {
		if r.Path != "" {
			paths.Insert(r.Path)
		}
	}
	return paths.Slice()
}

type ServerStatus struct {
	Ping         bool `json:"ping"`
	DBRepository bool `json:"dbRepository"`
}

func (s *Server) Ping() *ServerStatus {
	status := &ServerStatus{Ping: true}

	ctx, cannel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cannel()

	if err := infrastructure.Ping(ctx, s.db, s.rdb); err == nil {
		status.DBRepository = true
	}

	return status
}
