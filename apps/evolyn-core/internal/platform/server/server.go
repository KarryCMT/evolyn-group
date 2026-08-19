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

	"evolyn/internal/config"
	"evolyn/internal/infrastructure"
	// swagger spec 注册：swag init 生成的 docs 包经 init() 把接口定义
	// 注册给 gin-swagger，不引入则 /swagger/doc.json 500（页面能开但无内容）
	_ "evolyn/docs"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/auth"
	authcontroller "evolyn/internal/platform/auth/controller"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/auth/sms"
	"evolyn/internal/platform/controller"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"
	iamcontroller "evolyn/internal/platform/iam/controller"
	"evolyn/internal/platform/iam/repository"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/platform/middleware"
	tenantcontroller "evolyn/internal/platform/tenant/controller"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
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
	// FIX-009：两种 Schema 管理策略互斥，配错即拒绝启动
	if conf.DB.Migrate && conf.DB.Migrations {
		return nil, fmt.Errorf("db.migrate (AutoMigrate) and db.migrations (SQL migrations) are mutually exclusive")
	}

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

	// Schema 管理（FIX-009）：SQL Migration 是唯一生产路径；
	// AutoMigrate 仅供开发/测试实验。种子/回填两条路径共享（幂等）
	if conf.DB.Migrations {
		if err := infrastructure.NewMigrator(db).Up(); err != nil {
			return nil, errors.Wrap(err, "sql migrations failed")
		}
	}

	// 域仓储装配（ADR-007 域模块化）：audit 无租户Callback 依赖先行；
	// tenant 先于 iam AutoMigrate，保证业务模型加 tenant_id 列时默认租户已就绪
	auditRepo := auditrepository.NewRepository(db)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	iamRepo := repository.NewRepositories(db, rdb)
	if conf.DB.Migrate {
		if err := auditRepo.Migrate(); err != nil {
			return nil, err
		}
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

	// 域服务装配：审计/配额/事务管理为跨域基础能力，先于业务服务构造
	// （FIX-020/021：核心写路径经 TxManager 声明原子边界）
	auditSvc := auditservice.NewService(auditRepo)
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, iamRepo.User())
	txManager := infrastructure.NewTxManager(db)

	tenantService := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, conf.Tenant.Retention())
	accountService := service.NewAccountService(txManager, iamRepo.Account(), iamRepo.User(), tenantRepo, quotaSvc)
	userService := service.NewUserService(txManager, iamRepo.User(), iamRepo.Account(), iamRepo.RBAC(), iamRepo.Department(), quotaSvc, auditSvc)
	departmentService := service.NewDepartmentService(iamRepo.Department(), iamRepo.User(), auditSvc)
	groupService := service.NewGroupService(iamRepo.Group(), iamRepo.User(), iamRepo.RBAC(), auditSvc)
	jwtService := auth.NewJWTService(conf.Server.JWTSecret)
	smsService := sms.NewService(rdb.Client, sms.NewDevSender(), sms.Options{
		CodeTTL:  time.Duration(conf.SMS.CodeTTLSeconds) * time.Second,
		Cooldown: time.Duration(conf.SMS.CooldownSeconds) * time.Second,
		MaxTries: conf.SMS.MaxTries,
		DevEcho:  conf.SMS.DevEcho,
	})
	rbacService := service.NewRBACService(iamRepo.RBAC(), auditSvc)
	oauthManager := oauth.NewOAuthManager(conf.OAuthConfig)

	userController := iamcontroller.NewUserController(userService, departmentService)
	accountController := iamcontroller.NewAccountController(accountService)
	departmentController := iamcontroller.NewDepartmentController(departmentService)
	groupController := iamcontroller.NewGroupController(groupService)
	authController := authcontroller.NewAuthController(accountService, jwtService, oauthManager, tenantService, smsService)
	rbacController := iamcontroller.NewRbacController(rbacService)
	tenantController := tenantcontroller.NewTenantController(tenantService)

	// 鉴权器显式注入 iam 仓储（P0-4：拆除全局单例）
	authorizer := authorization.NewAuthorizer(iamRepo.User(), iamRepo.Group())

	controllers := []controller.Controller{userController, groupController, authController, rbacController, tenantController, accountController, departmentController}

	// 注销数据清理任务（FIX-012）：随服务生命周期启停
	purgeWorker := tenantservice.NewPurgeWorker(tenantRepo, conf.Tenant.PurgeInterval(), logger)

	gin.SetMode(conf.Server.ENV)

	e := gin.New()
	// 全局链路（FIX-008）：只保留与权限域无关的横切能力；
	// 认证挂全局（平台/租户两域都需要会话解析，未认证请求照常放行至鉴权层）
	e.Use(
		gin.Recovery(),
		rateLimitMiddleware,
		middleware.MonitorMiddleware(),
		middleware.CORSMiddleware(),
		middleware.RequestInfoMiddleware(&request.RequestInfoFactory{APIPrefixes: set.NewString("api")}),
		middleware.LogMiddleware(logger, "/"),
		middleware.AuthenticationMiddleware(jwtService, iamRepo.User()),
		middleware.TraceMiddleware(),
	)

	return &Server{
		engine:      e,
		config:      conf,
		logger:      logger,
		db:          db,
		rdb:         rdb,
		controllers: controllers,
		purgeWorker: purgeWorker,
		authorizer:  authorizer,
		tenantRepo:  tenantRepo,
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
	purgeWorker *tenantservice.PurgeWorker
	authorizer  *authorization.Authorizer
	tenantRepo  tenantrepository.TenantRepository
}

// graceful shutdown
func (s *Server) Run() error {
	defer s.Close()

	s.initRouter()

	// 注销清理任务随服务启动，ctx 取消即退出（FIX-012）
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go s.purgeWorker.Run(workerCtx)

	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)
	s.logger.Infof("Start server on: %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: s.engine,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

// initRouter 注册业务路由（FIX-008 域隔离）：
//
//	/api/v1/platform/*  Authentication + PlatformAuthorization（无租户上下文）
//	/api/v1/*           Authentication + Tenant + TenantStatus + Authorization
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

	// 平台运营域：无 TenantMiddleware/TenantStatusMiddleware（FIX-008）
	platform := api.Group("/platform")
	platform.Use(middleware.PlatformAuthorizationMiddleware())

	// 租户域：解析/注入租户 → 状态拦截（FIX-007）→ 资源级鉴权
	tenantAPI := api.Group("")
	tenantAPI.Use(
		middleware.TenantMiddleware(),
		middleware.TenantStatusMiddleware(s.tenantRepo),
		middleware.AuthorizationMiddleware(s.authorizer),
	)

	controllers := make([]string, 0, len(s.controllers))
	for _, router := range s.controllers {
		// PlatformController 标记决定归属域（FIX-008：两个权限域不可串用）
		if pc, ok := router.(controller.PlatformController); ok && pc.Platform() {
			router.RegisterRoute(platform)
		} else {
			router.RegisterRoute(tenantAPI)
		}
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
