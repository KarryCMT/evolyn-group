package server

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"evolyn/internal/authorization"
	"evolyn/internal/config"
	"evolyn/internal/controller"
	"evolyn/internal/database"
	"evolyn/internal/middleware"
	"evolyn/internal/repository"
	"evolyn/internal/service"
	"evolyn/pkg/authentication"
	"evolyn/pkg/authentication/oauth"
	"evolyn/pkg/common"
	"evolyn/pkg/utils/request"
	"evolyn/pkg/utils/set"
	"evolyn/pkg/version"

	"github.com/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(conf *config.Config, logger *logrus.Logger) (*Server, error) {
	rateLimitMiddleware, err := middleware.RateLimitMiddleware(conf.Server.LimitConfigs)
	if err != nil {
		return nil, err
	}

	db, err := database.NewPostgres(&conf.DB)
	if err != nil {
		return nil, errors.Wrap(err, "db init failed")
	}

	rdb, err := database.NewRedisClient(&conf.Redis)
	if err != nil {
		return nil, errors.Wrap(err, "redis client failed")
	}

	repository := repository.NewRepository(db, rdb)
	if conf.DB.Migrate {
		if err := repository.Migrate(); err != nil {
			return nil, err
		}
	}

	if err := repository.Init(); err != nil {
		return nil, err
	}

	userService := service.NewUserService(repository.User())
	groupService := service.NewGroupService(repository.Group(), repository.User())
	jwtService := authentication.NewJWTService(conf.Server.JWTSecret)
	rbacService := service.NewRBACService(repository.RBAC())
	oauthManager := oauth.NewOAuthManager(conf.OAuthConfig)

	userController := controller.NewUserController(userService)
	groupController := controller.NewGroupController(groupService)
	authController := controller.NewAuthController(userService, jwtService, oauthManager)
	rbacController := controller.NewRbacController(rbacService)

	if err := authorization.InitAuthorization(repository); err != nil {
		return nil, err
	}

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
		middleware.AuthenticationMiddleware(jwtService, repository.User()),
		middleware.TenantMiddleware(),
		middleware.AuthorizationMiddleware(),
		middleware.TraceMiddleware(),
	)

	return &Server{
		engine:      e,
		config:      conf,
		logger:      logger,
		repository:  repository,
		controllers: controllers,
	}, nil
}

type Server struct {
	engine *gin.Engine
	config *config.Config
	logger *logrus.Logger

	repository repository.Repository

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
	if err := s.repository.Close(); err != nil {
		s.logger.Warnf("failed to close repository, %v", err)
	}

}

func (s *Server) initRouter() {
	root := s.engine

	// register non-resource routers
	root.GET("/", common.WrapFunc(s.getRoutes))
	root.GET("/index", controller.Index)
	root.GET("/healthz", common.WrapFunc(s.Ping))
	root.GET("/version", common.WrapFunc(version.Get))
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

	if err := s.repository.Ping(ctx); err == nil {
		status.DBRepository = true
	}

	return status
}
