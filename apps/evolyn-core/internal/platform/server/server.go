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
	"evolyn/internal/infrastructure/ipregion"
	// swagger spec 注册：swag init 生成的 docs 包经 init() 把接口定义
	// 注册给 gin-swagger，不引入则 /swagger/doc.json 500（页面能开但无内容）
	_ "evolyn/docs"
	applicationcontroller "evolyn/internal/platform/application/controller"
	applicationrepository "evolyn/internal/platform/application/repository"
	applicationservice "evolyn/internal/platform/application/service"
	auditrepository "evolyn/internal/platform/audit/repository"
	auditservice "evolyn/internal/platform/audit/service"
	"evolyn/internal/platform/auth"
	authcontroller "evolyn/internal/platform/auth/controller"
	loginlogrepository "evolyn/internal/platform/auth/loginlog/repository"
	loginlogservice "evolyn/internal/platform/auth/loginlog/service"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/auth/pki"
	authservice "evolyn/internal/platform/auth/service"
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
	loginLogRepo := loginlogrepository.NewRepository(db)
	tenantRepo := tenantrepository.NewRepository(db, rdb)
	iamRepo := repository.NewRepositories(db, rdb)
	// 应用域仓储先于配额服务装配：apps 计量面（CountBillableByTenant）随
	// 应用域落地接入 QuotaService（M2-A）
	applicationRepo := applicationrepository.NewRepository(db)
	if conf.DB.Migrate {
		if err := auditRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := loginLogRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := tenantRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := iamRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := applicationRepo.Migrate(); err != nil {
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
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), applicationRepo)
	txManager := infrastructure.NewTxManager(db)

	// 登录日志域（认证域内，000013）：IP 归属地解析器数据内嵌随二进制，
	// 装载失败仅可能为程序性损坏，fail-fast 拒绝启动
	ipResolver, err := ipregion.New()
	if err != nil {
		return nil, errors.Wrap(err, "ip region resolver init failed")
	}
	loginLogSvc := loginlogservice.NewService(loginLogRepo, ipResolver)

	tenantService := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, conf.Tenant.Retention())
	accountService := service.NewAccountService(txManager, iamRepo.Account(), iamRepo.User(), tenantRepo, quotaSvc)
	userService := service.NewUserService(txManager, iamRepo.User(), iamRepo.Account(), iamRepo.RBAC(), iamRepo.Department(), quotaSvc, auditSvc)
	departmentService := service.NewDepartmentService(iamRepo.Department(), iamRepo.User(), auditSvc)
	groupService := service.NewGroupService(iamRepo.Group(), iamRepo.User(), iamRepo.RBAC(), auditSvc)
	jwtService := auth.NewJWTService(conf.Server.JWTSecret)
	// 短信通道按 provider 分派：dev（默认）走开发通道 + 固定验证码 666666，
	// 便于本地/测试环境联调；真实服务商待接入，配置即启动拦截（fail-fast）
	smsSender, smsFixedCode, err := buildSmsSender(conf.SMS.Provider)
	if err != nil {
		return nil, err
	}
	smsService := sms.NewService(rdb.Client, smsSender, sms.Options{
		CodeTTL:   time.Duration(conf.SMS.CodeTTLSeconds) * time.Second,
		Cooldown:  time.Duration(conf.SMS.CooldownSeconds) * time.Second,
		MaxTries:  conf.SMS.MaxTries,
		DevEcho:   conf.SMS.DevEcho,
		FixedCode: smsFixedCode,
	})
	rbacService := service.NewRBACService(iamRepo.RBAC(), auditSvc)
	// 应用域服务（M2-A）：空白应用创建/查询/更新/软删 + 配额占位；
	// 访问判定与鉴权中间件同源（按 ID 重载成员 + authenticated 系统组）
	appAccess := applicationservice.NewRBACAccessEvaluator(iamRepo.User(), iamRepo.Group())
	applicationService := applicationservice.NewApplicationService(txManager, applicationRepo, quotaSvc, auditSvc, appAccess)
	oauthManager := oauth.NewOAuthManager(conf.OAuthConfig)

	// 登录口令加密密钥对：私钥留服务端解密，公钥经 /app/conf 下发前端。
	// 未配置私钥时启动随机生成（仅开发/测试），日志提醒生产必须显式配置
	keypair, err := pki.Load(conf.PKI.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "pki keypair init failed")
	}
	if conf.PKI.PrivateKey == "" {
		logger.Warn("pki.privateKey 未配置，已随机生成临时 RSA 密钥对（重启轮换；生产与多实例部署必须显式配置同一密钥对）")
	}

	userController := iamcontroller.NewUserController(userService, departmentService)
	accountController := iamcontroller.NewAccountController(accountService, keypair, loginLogSvc)
	departmentController := iamcontroller.NewDepartmentController(departmentService)
	groupController := iamcontroller.NewGroupController(groupService)
	// 注册编排服务（认证域）：注册向导最终提交「进入产品」的单事务落库
	// （免密注册账号 + 账号画像 + 租户开通/复用 + owner 成员解析）
	registrationService := authservice.NewRegistrationService(txManager, accountService, tenantService, auditSvc)
	authController := authcontroller.NewAuthController(accountService, registrationService, jwtService, oauthManager, tenantService, smsService, keypair, loginLogSvc)
	rbacController := iamcontroller.NewRbacController(rbacService)
	tenantController := tenantcontroller.NewTenantController(tenantService)
	applicationController := applicationcontroller.NewApplicationController(applicationService)

	// 鉴权器显式注入 iam 仓储（P0-4：拆除全局单例）
	authorizer := authorization.NewAuthorizer(iamRepo.User(), iamRepo.Group())

	controllers := []controller.Controller{userController, groupController, authController, rbacController, tenantController, accountController, departmentController, applicationController}

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
		pkiKeypair:  keypair,
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
	// 登录口令加密密钥对：登录/改密解密与 /app/conf 公钥下发共用
	pkiKeypair *pki.Keypair
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

	// 应用配置为公开引导端点（匿名可达，与 /healthz 同级定位）：直接挂
	// api 组仅过全局链。不能进租户域 RBAC——RequestInfo 会把 /app/conf
	// 解析为资源请求（resource=app），匿名用户仅持有 auth:create 规则，
	// 未登录的登录/注册页将拿不到区号列表
	controller.NewAppConfController(s.pkiKeypair).RegisterRoute(api)

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

// buildSmsSender 按配置分派短信通道与固定验证码：
//   - dev（默认/空）：开发日志通道 + 固定码 666666（本地与测试环境联调用）
//   - 其他值：真实服务商通道待接入，配置即启动拦截，避免静默走假通道
func buildSmsSender(provider string) (sms.Sender, string, error) {
	switch provider {
	case "", "dev":
		return sms.NewDevSender(), sms.DevFixedCode, nil
	default:
		return nil, "", fmt.Errorf("短信服务商 provider=%q 尚未支持（当前仅 dev），接入真实通道后实现 sms.Sender 并在此分派", provider)
	}
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
