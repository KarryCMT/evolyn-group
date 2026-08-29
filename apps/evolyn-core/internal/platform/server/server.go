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
	"evolyn/internal/contextx"
	"evolyn/internal/infrastructure"
	"evolyn/internal/infrastructure/ipregion"
	"evolyn/internal/infrastructure/objectstore"
	apperrors "evolyn/internal/platform/enterpriselog"
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
	emailauth "evolyn/internal/platform/auth/email"
	loginlogrepository "evolyn/internal/platform/auth/loginlog/repository"
	loginlogservice "evolyn/internal/platform/auth/loginlog/service"
	"evolyn/internal/platform/auth/oauth"
	"evolyn/internal/platform/auth/pki"
	securitycontroller "evolyn/internal/platform/auth/security/controller"
	securityrepository "evolyn/internal/platform/auth/security/repository"
	securityservice "evolyn/internal/platform/auth/security/service"
	"evolyn/internal/platform/auth/security/totp"
	authservice "evolyn/internal/platform/auth/service"
	"evolyn/internal/platform/auth/sms"
	"evolyn/internal/platform/controller"
	editioncontroller "evolyn/internal/platform/edition/controller"
	editionrepository "evolyn/internal/platform/edition/repository"
	editionservice "evolyn/internal/platform/edition/service"
	enterpriselogcontroller "evolyn/internal/platform/enterpriselog/controller"
	enterpriselogrepository "evolyn/internal/platform/enterpriselog/repository"
	enterpriselogservice "evolyn/internal/platform/enterpriselog/service"
	filecontroller "evolyn/internal/platform/file/controller"
	filerepository "evolyn/internal/platform/file/repository"
	fileservice "evolyn/internal/platform/file/service"
	formcontroller "evolyn/internal/platform/form/controller"
	formrepository "evolyn/internal/platform/form/repository"
	formservice "evolyn/internal/platform/form/service"
	workflowadapter "evolyn/internal/platform/workflow/adapter"
	workflowerrors "evolyn/internal/platform/workflow"
	engineruntime "evolyn/internal/engine/workflow/runtime"
	"evolyn/internal/engine/workflow/assignment"
	"evolyn/internal/engine/workflow/executor"
	workflowcontroller "evolyn/internal/platform/workflow/controller"
	workflowrepository "evolyn/internal/platform/workflow/repository"
	workflowservice "evolyn/internal/platform/workflow/service"
	"evolyn/internal/platform/httpx"
	"evolyn/internal/platform/iam/authorization"
	iamcontroller "evolyn/internal/platform/iam/controller"
	"evolyn/internal/platform/iam/repository"
	"evolyn/internal/platform/iam/service"
	"evolyn/internal/platform/middleware"
	notificationcontroller "evolyn/internal/platform/notification/controller"
	notificationrepository "evolyn/internal/platform/notification/repository"
	notificationservice "evolyn/internal/platform/notification/service"
	tenantcontroller "evolyn/internal/platform/tenant/controller"
	tenantrepository "evolyn/internal/platform/tenant/repository"
	tenantservice "evolyn/internal/platform/tenant/service"
	tenantproductcontroller "evolyn/internal/platform/tenantproduct/controller"
	tenantproductrepository "evolyn/internal/platform/tenantproduct/repository"
	tenantproductservice "evolyn/internal/platform/tenantproduct/service"
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

	// MFA 的 TOTP 主密钥仅从部署配置加载。release 绝不允许在没有密钥或
	// Redis 的情况下启动该能力；debug 未配置时保留既有登录链路，安全设置
	// 接口会明确返回不可用而不会降级绕过。
	var keyring *totp.Keyring
	if conf.Security.TOTP.CurrentKeyVersion != 0 || len(conf.Security.TOTP.MasterKeys) != 0 {
		if err := conf.Security.TOTP.Validate(); err != nil {
			return nil, err
		}
		keyring, err = totp.NewKeyring(conf.Security.TOTP.CurrentKeyVersion, conf.Security.TOTP.MasterKeys)
		if err != nil {
			return nil, errors.Wrap(err, "totp keyring init failed")
		}
		if !rdb.Endable() {
			return nil, fmt.Errorf("security.totp 配置后 redis 必须启用")
		}
	} else if conf.Server.ENV == gin.ReleaseMode {
		return nil, fmt.Errorf("release 环境必须配置 security.totp 主密钥")
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
	// 应用域落地接入 QuotaService（M2-A）；菜单仓储随 M2-菜单-1 接入
	applicationRepo := applicationrepository.NewRepository(db)
	menuRepo := applicationrepository.NewMenuRepository(db)
	fileRepo := filerepository.NewRepository(db)
	// 版本信息域仓储（一期）：与各域仓储同批创建，dev AutoMigrate 块可用
	editionRepo := editionrepository.NewRepository(db)
	// 产品中心域仓储（一期）：平台产品目录 + 租户产品配置 + 范围关联
	tenantProductRepo := tenantproductrepository.NewRepository(db)
	// 企业日志域仓储（000036）：login_logs/audit_logs 只读查询 + 导出任务
	enterpriseLogRepo := enterpriselogrepository.NewRepository(db)
	// 表单资产域仓储（000037/000038，ADR-010）：表单+草稿、不可变发布快照、记录
	formRepo := formrepository.NewRepository(db)
	formVersionRepo := formrepository.NewVersionRepository(db)
	formRecordRepo := formrepository.NewRecordRepository(db)
	// 流程引擎仓储（000048/000049，ADR-012）：定义+草稿、不可变发布快照、
	// 运行态六表（实例/执行路径/节点实例/任务/参与人/操作流水）
	workflowDefinitionRepo := workflowrepository.NewDefinitionRepository(db)
	workflowVersionRepo := workflowrepository.NewVersionRepository(db)
	workflowInstanceRepo, workflowExecutionRepo, workflowNodeRepo, workflowTaskRepo, workflowOperationRepo := workflowrepository.NewRuntimeRepositories(db)
	workflowEngineReader := workflowrepository.NewEngineDefinitionReader(db)
	workflowRuntimeReader := workflowrepository.NewRuntimeReader(db)
	// 消息中心域仓储（000039）：逻辑消息+收件箱、通知设置聚合、事务 Outbox
	notificationMessageRepo := notificationrepository.NewMessageRepository(db)
	notificationSettingRepo := notificationrepository.NewSettingRepository(db)
	notificationOutboxRepo := notificationrepository.NewOutboxRepository(db)
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
		if err := menuRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := fileRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := editionRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := tenantProductRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := enterpriseLogRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := formRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := formVersionRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := formRecordRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := workflowDefinitionRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := workflowVersionRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := notificationMessageRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := notificationSettingRepo.Migrate(); err != nil {
			return nil, err
		}
		if err := notificationOutboxRepo.Migrate(); err != nil {
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
	// （FIX-020/021：核心写路径经 TxManager 声明原子边界）。审计注入操作者
	// 显示名解析端口（企业日志展示快照，000036）：iam 成员仓储适配
	auditSvc := auditservice.NewService(auditRepo, auditActorNamer{users: iamRepo.User()})
	quotaSvc := tenantservice.NewQuotaService(tenantRepo, tenantRepo, iamRepo.User(), applicationRepo, fileRepo)
	// 表单配额键（forms）随表单域落地接入计量面（ADR-010）；注入失败属装配
	// 错误直接返回，不允许「表面配置了配额实际不生效」
	if injector, ok := quotaSvc.(tenantservice.QuotaFormCounterInjector); ok {
		injector.UseFormCounter(formRepo)
	}
	storageQuotaSvc, ok := quotaSvc.(tenantservice.StorageQuotaService)
	if !ok {
		return nil, fmt.Errorf("storage quota service not configured")
	}
	txManager := infrastructure.NewTxManager(db)

	// 版本信息域服务（一期）：先于租户服务装配——开通事务要经
	// SubscriptionSeeder 补种初始订阅；QuotaService 经守卫在到期窗口改读
	// 统一权益解析结果（设计 4.4.1）
	editionService := editionservice.NewEditionService(txManager, editionRepo, tenantRepo, auditSvc, iamRepo.User(), applicationRepo, fileRepo)
	if injector, ok := quotaSvc.(tenantservice.QuotaGuardInjector); ok {
		injector.UseExpiryGuard(editionService)
	}

	// 登录日志域（认证域内，000013）：IP 归属地解析器数据内嵌随二进制，
	// 装载失败仅可能为程序性损坏，fail-fast 拒绝启动
	ipResolver, err := ipregion.New()
	if err != nil {
		return nil, errors.Wrap(err, "ip region resolver init failed")
	}
	loginLogSvc := loginlogservice.NewService(loginLogRepo, ipResolver)

	// 成员信息管理（一期）：字段配置/成员档案服务先于租户服务构造——开通
	// 事务经 UseFieldSeeder 注入预置默认字段配置（读取侧另有幂等兜底）
	memberFieldService := service.NewMemberFieldService(txManager, iamRepo.MemberFieldSetting(), auditSvc)
	memberProfileService := service.NewMemberProfileService(txManager, iamRepo.MemberProfile(), iamRepo.MemberFieldSetting(), iamRepo.User(), auditSvc)

	// 管理组服务（权限中心-管理员模块）：应用清单经窄端口适配器桥接——
	// iam 不能反向依赖应用域（应用域已依赖 iam 鉴权），装配层逐 ID 探测
	adminGroupService := service.NewAdminGroupService(
		txManager, iamRepo.AdminGroup(), iamRepo.User(), iamRepo.Department(), iamRepo.RBAC(),
		adminGroupApplicationCatalog{applications: applicationRepo}, tenantRepo, auditSvc,
	)

	// 产品中心域服务（一期）：版本投影经窄端口只读消费 edition 服务；
	// 先于租户服务构造——开通事务经 UseProductSeeder 注入产品配置初始化
	tenantProductService := tenantproductservice.NewTenantProductService(txManager, tenantProductRepo, editionService, auditSvc)

	tenantService := tenantservice.NewTenantService(txManager, tenantRepo, iamRepo, quotaSvc, auditSvc, conf.Tenant.Retention(), editionService)
	if injector, ok := tenantService.(tenantservice.MemberFieldSeederInjector); ok {
		injector.UseFieldSeeder(memberFieldService)
	}
	if injector, ok := tenantService.(tenantservice.ProductConfigSeederInjector); ok {
		injector.UseProductSeeder(tenantProductService)
	}
	// 账号服务注入审计：换绑手机号等安全敏感操作落业务审计（best-effort）
	accountService := service.NewAccountService(txManager, iamRepo.Account(), iamRepo.User(), tenantRepo, quotaSvc, auditSvc)
	userService := service.NewUserService(txManager, iamRepo.User(), iamRepo.Account(), iamRepo.RBAC(), iamRepo.Department(), quotaSvc, auditSvc, tenantRepo)
	memberInvitationService := service.NewMemberInvitationService(txManager, iamRepo.Invitation(), iamRepo.Department(), iamRepo.Account(), userService, iamRepo.MemberProfile(), auditSvc)
	departmentService := service.NewDepartmentService(iamRepo.Department(), iamRepo.User(), auditSvc)
	groupService := service.NewGroupService(iamRepo.Group(), iamRepo.User(), iamRepo.RBAC(), auditSvc)
	jwtService := auth.NewJWTService(conf.Server.JWTSecret)
	// 令牌吊销器（P2-8）：登出拉黑 jti，登出前令牌固定 7 天有效的问题收口。
	// failClosed 来自 auth.revokeFailClosed：true 时 Redis 异常按「已吊销」
	// 拒绝请求（已泄露令牌立即失效优先），默认 false 可用性优先
	tokenRevoker := auth.NewTokenRevoker(rdb.Client, conf.Auth.RevokeFailClosed)
	// 登录失败锁定（上线前整改 P2）：密码登录按登录名/手机号计连续失败；
	// 独立密钥启用 HMAC 标识散列（防字典反查），未配置回退无密钥散列并告警
	if conf.Auth.LoginGuardSecret == "" && gin.Mode() == gin.ReleaseMode {
		logger.Warn("auth.loginGuardSecret 未配置：登录失败计数的标识散列回退无密钥 SHA-256（生产建议配置独立随机密钥，多实例共享同一把）")
	}
	loginGuard := auth.NewLoginGuard(rdb.Client, auth.LoginGuardOptions{
		MaxFails:     conf.Auth.LoginMaxFails,
		LockDuration: time.Duration(conf.Auth.LoginLockMinutes) * time.Minute,
		Secret:       conf.Auth.LoginGuardSecret,
	})
	// 短信通道按 provider 分派：dev（默认）走开发通道 + 固定验证码 666666，
	// 便于本地/测试环境联调；真实服务商待接入，配置即启动拦截（fail-fast）
	smsSender, smsFixedCode, err := buildSmsSender(conf.SMS.Provider)
	if err != nil {
		return nil, err
	}
	smsService := sms.NewService(rdb.Client, smsSender, sms.Options{
		CodeTTL:      time.Duration(conf.SMS.CodeTTLSeconds) * time.Second,
		Cooldown:     time.Duration(conf.SMS.CooldownSeconds) * time.Second,
		MaxTries:     conf.SMS.MaxTries,
		DailyLimit:   conf.SMS.DailyLimit,
		IPDailyLimit: conf.SMS.IPDailyLimit,
		DevEcho:      conf.SMS.DevEcho,
		FixedCode:    smsFixedCode,
	})
	// 邮箱绑定验证码独立于短信验证码：手机号仅用于第一步身份持有证明，第二步
	// 邮件必须发送至待绑定的新地址。release 不允许静默使用开发通道。
	emailSender, emailFixedCode, err := buildEmailSender(conf.Email, conf.Server.ENV)
	if err != nil {
		return nil, err
	}
	emailService := emailauth.NewService(rdb.Client, emailSender, emailauth.Options{
		CodeTTL:     time.Duration(conf.Email.CodeTTLSeconds) * time.Second,
		Cooldown:    time.Duration(conf.Email.CooldownSeconds) * time.Second,
		MaxTries:    conf.Email.MaxTries,
		IdentityTTL: time.Duration(conf.Email.IdentityTTL) * time.Second,
		DevEcho:     conf.Email.DevEcho,
		FixedCode:   emailFixedCode,
	})
	rbacService := service.NewRBACService(iamRepo.RBAC(), auditSvc)
	organizationRoleService := service.NewOrganizationRoleService(txManager, iamRepo.RBAC(), iamRepo.RoleGroup(), iamRepo.User(), userService, auditSvc)
	// 应用域服务（M2-A）：空白应用创建/查询/更新/软删 + 配额占位；
	// 访问判定与鉴权中间件同源（按 ID 重载成员 + authenticated 系统组）
	appAccess := applicationservice.NewRBACAccessEvaluator(iamRepo.User(), iamRepo.Group())
	applicationService := applicationservice.NewApplicationService(txManager, applicationRepo, quotaSvc, auditSvc, appAccess)
	// 消息中心域（000039）：收件箱/设置服务 + 事务 Outbox 发布端口与扇出
	// Dispatcher（成员目录/系统管理员解析经窄端口适配，域内不依赖 iam 具体
	// Service）；应用域事件发布器随 Dispatcher 前装配注入
	notificationEventPublisher := notificationservice.NewEventPublisher(notificationOutboxRepo)
	if injector, ok := applicationService.(applicationservice.AssetNotifierInjector); ok {
		injector.UseAssetNotifier(applicationAssetNotifier{publisher: notificationEventPublisher})
	}
	notificationDispatcher := notificationservice.NewDispatcher(
		txManager, notificationOutboxRepo, notificationMessageRepo, notificationSettingRepo,
		notificationMemberDirectory{users: iamRepo.User()},
		notificationAdminResolver{adminGroups: iamRepo.AdminGroup()},
		conf.Notification.RetentionMonthsValue(), logger,
	)
	notificationDispatcher.SetBatchSize(conf.Notification.OutboxBatch())
	notificationInboxService := notificationservice.NewInboxService(
		txManager, notificationMessageRepo, conf.Notification.RetentionMonthsValue(),
	)
	notificationSettingService := notificationservice.NewSettingService(
		txManager, notificationSettingRepo, auditSvc, conf.Notification.RecipientLimit(),
	)
	if injector, ok := tenantService.(tenantservice.NotificationSettingSeederInjector); ok {
		injector.UseNotificationSeeder(notificationSettingService)
	}
	// 应用菜单服务：读取与分组创建复用应用域权限评估器；分组写入经统一
	// 事务、menuRevision 乐观锁和提交后审计保证一致性。
	menuService := applicationservice.NewMenuService(txManager, menuRepo, auditSvc, appAccess)
	var storageStore objectstore.Store
	if conf.Storage.Enabled {
		storageStore, err = objectstore.NewRustFS(conf.Storage)
		if err != nil {
			return nil, errors.Wrap(err, "rustfs client init failed")
		}
	}
	fileService := fileservice.NewFileService(txManager, fileRepo, storageQuotaSvc, auditSvc, storageStore, conf.Storage)
	fileCleanupWorker := fileservice.NewUploadCleanupWorker(fileService, conf.Storage.UploadCleanupInterval(), logger)
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

	userController := iamcontroller.NewUserController(userService, departmentService, memberInvitationService)
	// 账号控制器注入换绑验证码校验器（认证域 sms.Service 实现窄接口）
	accountController := iamcontroller.NewAccountController(accountService, keypair, loginLogSvc, smsService, emailService)
	platformAccountService := service.NewAccountDeletionService(txManager, iamRepo.Account(), iamRepo.User(), tenantRepo, auditSvc)
	platformAccountController := iamcontroller.NewPlatformAccountController(platformAccountService)

	// 账号安全子域（ADR-009）：会话/因子/恢复码/开关仓储 + 服务装配
	securitySettingsRepo := securityrepository.NewSettingsRepository(db)
	securitySessionRepo := securityrepository.NewSessionRepository(db)
	// 会话清理 Worker 与会话仓储同处装配，避免 Server.Run 启动空指针任务。
	sessionCleanupWorker := securityservice.NewSessionCleanupWorker(securitySessionRepo, 0, logger)
	sessionService := securityservice.NewSessionService(txManager, securitySettingsRepo, securitySessionRepo)
	securitySvc := securityservice.NewSecurityService(
		txManager,
		securitySettingsRepo,
		securityrepository.NewFactorRepository(db),
		securityrepository.NewRecoveryRepository(db),
		securitySessionRepo,
		securityrepository.NewEventRepository(db),
	)
	var mfaSvc securityservice.MFAService
	if keyring != nil {
		mfaSvc = securityservice.NewMFAService(
			txManager,
			securitySettingsRepo,
			securityrepository.NewFactorRepository(db),
			securityrepository.NewRecoveryRepository(db),
			securitySessionRepo,
			securityrepository.NewEventRepository(db),
			keyring,
			rdb.Client,
		)
	}
	securityController := securitycontroller.NewSecurityController(securitySvc, mfaSvc, accountService, keypair, platformAccountService)
	departmentController := iamcontroller.NewDepartmentController(departmentService)
	groupController := iamcontroller.NewGroupController(groupService)
	// 注册编排服务（认证域）：注册向导最终提交「进入产品」的单事务落库
	// （免密注册账号 + 账号画像 + 租户开通/复用 + owner 成员解析）
	registrationService := authservice.NewRegistrationService(txManager, accountService, tenantService, auditSvc, memberInvitationService)
	authController := authcontroller.NewAuthController(accountService, registrationService, jwtService, oauthManager, tenantService, smsService, keypair, loginLogSvc, tokenRevoker, sessionService, mfaSvc, loginGuard)
	rbacController := iamcontroller.NewRbacController(rbacService)
	organizationRoleController := iamcontroller.NewOrganizationRoleController(organizationRoleService)
	tenantController := tenantcontroller.NewTenantController(tenantService)
	tenantProfileController := tenantcontroller.NewTenantProfileController(tenantService)
	applicationController := applicationcontroller.NewApplicationController(applicationService)
	menuController := applicationcontroller.NewMenuController(menuService)
	fileController := filecontroller.NewFileController(fileService)
	editionController := editioncontroller.NewEditionController(editionService)
	platformEditionController := editioncontroller.NewPlatformEditionController(editionService)
	// 产品中心（一期）：管理面控制器挂租户域链；TenantProductAccessEvaluator
	// 供产品真实受保护路径接入（产品后端落地时构造，同应用域 appAccess 先例）
	tenantProductController := tenantproductcontroller.NewTenantProductController(tenantProductService)
	memberFieldController := iamcontroller.NewMemberFieldController(memberFieldService)
	memberProfileController := iamcontroller.NewMemberProfileController(memberProfileService)
	adminGroupController := iamcontroller.NewAdminGroupController(adminGroupService)
	adminScopesController := iamcontroller.NewAdminScopesController(adminGroupService)

	// 鉴权器显式注入 iam 仓储（P0-4：拆除全局单例）；管理组仓储供 RBAC
	// 拒绝后的范围裁决回落（权限中心-管理员模块）
	authorizer := authorization.NewAuthorizer(iamRepo.User(), iamRepo.Group(), iamRepo.AdminGroup())

	// 企业日志域（000036）：只读查询 + 导出编排；成员目录/下载复核经窄端口
	// 适配（域间不直接耦合 iam），下载路径复核 create 动词（export 语义）
	enterpriseLogService := enterpriselogservice.NewEnterpriseLogService(
		enterpriseLogRepo, enterpriseLogMemberDirectory{users: iamRepo.User()}, auditSvc,
	)
	enterpriseLogController := enterpriselogcontroller.NewEnterpriseLogController(enterpriseLogService, authorizer)

	// M2-资产-1：菜单读侧接入表单目录（存在性裁剪 + target 投影），
	// 由表单仓储经装配层适配，application 域不依赖 form 域
	if injector, ok := menuService.(applicationservice.MenuFormDirectoryInjector); ok {
		injector.UseFormDirectory(formMenuDirectory{forms: formRepo})
	}

	// 表单资产域（ADR-010）：草稿/发布/提交；权限集、应用只读目录与菜单
	// 节点维护端口均经窄端口适配（域间不直接耦合 application），访问判定
	// 与鉴权中间件同源
	formMenuMaintenance := applicationservice.NewMenuMaintenanceService(menuRepo)
	formService := formservice.NewFormService(
		txManager, formRepo, formVersionRepo, formRecordRepo, quotaSvc, auditSvc,
		appAccess, formApplicationDirectory{applications: applicationRepo}, formMenuMaintenance,
	)
	// 引用视图只读端口（ADR-011）：form 域不反向依赖 application 域，装配层
	// 以菜单仓储桥接（跨应用反查引用指定表单的菜单节点）
	if injector, ok := formService.(formservice.FormReferenceSourceInjector); ok {
		injector.UseReferenceSource(formReferenceSource{menu: menuRepo})
	}
	formController := formcontroller.NewFormController(formService)

	// 流程引擎域（000048，ADR-012）Definition Engine：定义 CRUD/草稿/发布；
	// DSL 校验以引擎内核严格校验器为唯一事实源，权限集经窄端口与鉴权同源
	workflowService := workflowservice.NewDefinitionService(
		txManager, workflowDefinitionRepo, workflowVersionRepo, appAccess, auditSvc,
	)
	workflowController := workflowcontroller.NewWorkflowController(workflowService)

	// 最小 Runtime（000049）：引擎执行器注册表 + 事件日志适配器（Phase 6 换
	// Outbox 桥接）+ 表单目录窄端口（form 域仓储桥接，跨域不直接依赖），
	// 事务在 RuntimeService 内经 TxManager 建立，内核在同一事务内推进
	workflowExecutors := executor.NewRegistry(assignment.NewRegistry(nil))
	workflowRuntimeService := workflowservice.NewRuntimeService(
		txManager,
		engineruntime.NewRuntime(workflowEngineReader, workflowInstanceRepo, workflowExecutionRepo,
			workflowNodeRepo, workflowTaskRepo, workflowOperationRepo, workflowExecutors, workflowadapter.NewEventPublisher()),
		workflowRuntimeReader, workflowEngineReader,
		workflowFormDirectory{forms: formRepo, versions: formVersionRepo},
		appAccess, auditSvc,
	)
	workflowInstanceController := workflowcontroller.NewWorkflowInstanceController(workflowRuntimeService)

	// 消息中心域（000039）控制器：收件箱（notifications）/ 通知设置
	//（notification-settings）均挂租户域链
	notificationController := notificationcontroller.NewNotificationController(notificationInboxService)
	notificationSettingController := notificationcontroller.NewSettingController(notificationSettingService)

	controllers := []controller.Controller{userController, groupController, authController, rbacController, organizationRoleController, tenantController, tenantProfileController, accountController, platformAccountController, departmentController, applicationController, menuController, fileController, editionController, platformEditionController, memberFieldController, memberProfileController, adminGroupController, adminScopesController, tenantProductController, securityController, enterpriseLogController, formController, notificationController, notificationSettingController, workflowController, workflowInstanceController}

	// 注销数据清理任务（FIX-012）：随服务生命周期启停
	purgeWorker := tenantservice.NewPurgeWorker(tenantRepo, conf.Tenant.PurgeInterval(), logger)
	// 订阅到期降级任务（版本信息一期）：间隔取默认值（读时投影已兜底窗口期）
	editionWorker := editionservice.NewEditionWorker(editionService, 0, logger)
	// 消息中心（000039）：Outbox 物化 + 过期消息清理，随服务生命周期启停
	notificationOutboxWorker := notificationservice.NewOutboxWorker(
		notificationDispatcher, conf.Notification.OutboxInterval(), logger,
	)
	notificationRetentionWorker := notificationservice.NewRetentionWorker(
		notificationDispatcher, conf.Notification.RetentionInterval(), logger,
	)

	gin.SetMode(conf.Server.ENV)

	// P1 整改：CORS 白名单。release 携带凭证跨域绝不放行任意 Origin，
	// 空白名单视为配置错误拒绝启动（对齐 db.migrate/migrations 互斥校验口径；
	// 空串项已在 config 解析时过滤，[""] 不会绕过该校验）；
	// 仅 gin.DebugMode 空白名单回落放行本机回环地址（联调端口不固定），
	// TestMode 等非 release 模式不享受宽松回落
	if gin.Mode() == gin.ReleaseMode && len(conf.Server.AllowedOrigins) == 0 {
		return nil, fmt.Errorf("server.allowedOrigins 未配置：release 环境必须显式配置 CORS 来源白名单")
	}
	corsDevLoose := gin.Mode() == gin.DebugMode && len(conf.Server.AllowedOrigins) == 0
	if corsDevLoose {
		logger.Warn("server.allowedOrigins 未配置，debug 环境回落放行 localhost/127.0.0.1 任意端口（release 环境必须显式配置）")
	}

	e := gin.New()
	// 全局链路（FIX-008）：只保留与权限域无关的横切能力；
	// 认证挂全局（平台/租户两域都需要会话解析，未认证请求照常放行至鉴权层）
	e.Use(
		gin.Recovery(),
		rateLimitMiddleware,
		middleware.MonitorMiddleware(),
		middleware.CORSMiddleware(conf.Server.AllowedOrigins, corsDevLoose),
		middleware.RequestInfoMiddleware(&request.RequestInfoFactory{APIPrefixes: set.NewString("api")}),
		middleware.LogMiddleware(logger, "/"),
		middleware.AuthenticationMiddleware(jwtService, iamRepo.User(), iamRepo.Account(), tokenRevoker, sessionService),
		middleware.TraceMiddleware(),
	)

	return &Server{
		engine:                   e,
		config:                   conf,
		logger:                   logger,
		db:                       db,
		rdb:                      rdb,
		controllers:              controllers,
		fileController:           fileController,
		purgeWorker:              purgeWorker,
		editionWorker:            editionWorker,
		fileCleanupWorker:        fileCleanupWorker,
		sessionCleanupWorker:     sessionCleanupWorker,
		notificationOutboxWorker: notificationOutboxWorker,
		notificationRetWorker:    notificationRetentionWorker,
		authorizer:               authorizer,
		tenantRepo:               tenantRepo,
		pkiKeypair:               keypair,
	}, nil
}

type Server struct {
	engine *gin.Engine
	config *config.Config
	logger *logrus.Logger

	// 存活探测与关闭由 infrastructure 直接承担（域仓储不再聚合生命周期）
	db  *gorm.DB
	rdb *infrastructure.RedisDB

	controllers       []controller.Controller
	fileController    *filecontroller.FileController
	purgeWorker       *tenantservice.PurgeWorker
	editionWorker     *editionservice.EditionWorker
	fileCleanupWorker *fileservice.UploadCleanupWorker
	// 会话清理任务（ADR-009 SEC-3-4）：删除过期/超保留期的历史设备会话
	sessionCleanupWorker *securityservice.SessionCleanupWorker
	// 消息中心任务（000039）：Outbox 物化扇出 + 过期消息保留清理
	notificationOutboxWorker *notificationservice.OutboxWorker
	notificationRetWorker    *notificationservice.RetentionWorker
	authorizer               *authorization.Authorizer
	tenantRepo               tenantrepository.TenantRepository
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
	go s.editionWorker.Run(workerCtx)
	go s.fileCleanupWorker.Run(workerCtx)
	go s.sessionCleanupWorker.Run(workerCtx)
	go s.notificationOutboxWorker.Run(workerCtx)
	go s.notificationRetWorker.Run(workerCtx)

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
	// 头像等公开展示资源通过应用地址跳转至短期对象 URL，真实对象保持私有。
	s.fileController.RegisterPublicRoute(api)

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

// adminGroupApplicationCatalog 管理组应用清单窄端口适配器：iam 域不能反向
// 依赖应用域（应用域已依赖 iam 鉴权），装配层以逐 ID 探测桥接；选择器提交
// 的 ID 集合有限且租户应用为配额内规模，逐个 GetByID 足够（跨租户/已删 ID
// 经 Callback 过滤为 NotFound → false）
type adminGroupApplicationCatalog struct {
	applications applicationrepository.ApplicationRepository
}

func (c adminGroupApplicationCatalog) Exists(ctx context.Context, id uint) (bool, error) {
	_, err := c.applications.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// formMenuDirectory 菜单读侧的表单目录窄端口适配：application 域不反向
// 依赖 form 域，装配层以表单仓储桥接（批量存在性查询）
type formMenuDirectory struct {
	forms formrepository.FormRepository
}

func (d formMenuDirectory) ExistingFormTargets(ctx context.Context, ids []uint) (map[uint]applicationservice.FormTargetProjection, error) {
	forms, err := d.forms.ExistingFormTargets(ctx, ids)
	if err != nil {
		return nil, err
	}
	targets := make(map[uint]applicationservice.FormTargetProjection, len(forms))
	for id, form := range forms {
		targets[id] = applicationservice.FormTargetProjection{
			Code:     form.Code,
			FormType: string(form.FormType),
		}
	}
	return targets, nil
}

// formApplicationDirectory 表单域的应用只读目录窄端口适配：form 域不直接依赖
// application 域，装配层以应用仓储桥接；GetByID/GetByCode 经 ctx 租户过滤，
// 跨租户即 NotFound（与 form 域 ErrFormAppInvalid 口径一致）
type formApplicationDirectory struct {
	applications applicationrepository.ApplicationRepository
}

// formReferenceSource 表单引用视图只读窄端口适配（ADR-011）：application
// 域不反向依赖 form 域，装配层以菜单仓储桥接；租户上下文由请求 ctx 承载，
// 仓储查询显式携带租户条件
type formReferenceSource struct {
	menu applicationrepository.MenuRepository
}

// workflowFormDirectory 流程引擎的表单目录窄端口适配（ADR-012）：把
// (formCode, formVersionNo) 解析为内部 form_id / form_version_id；表单
// 不存在/跨租户/版本号不存在统一映射 WORKFLOW_FORM_VERSION_INVALID。
type workflowFormDirectory struct {
	forms    formrepository.FormRepository
	versions formrepository.FormVersionRepository
}

func (d workflowFormDirectory) ResolveFormVersion(ctx context.Context, formCode string, versionNo int) (uint, uint, error) {
	form, err := d.forms.GetByCode(ctx, formCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, workflowerrors.ErrFormVersionInvalid
		}
		return 0, 0, err
	}
	version, err := d.versions.GetByFormAndVersionNo(ctx, form.ID, versionNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, 0, workflowerrors.ErrFormVersionInvalid
		}
		return 0, 0, err
	}
	return form.ID, version.ID, nil
}

func (s formReferenceSource) ListFormReferences(ctx context.Context, formID uint) ([]formservice.FormReference, error) {
	tenantID, ok := contextx.TenantIDFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("tenant context required")
	}
	rows, err := s.menu.ListFormMenuReferences(ctx, tenantID, formID)
	if err != nil {
		return nil, err
	}
	references := make([]formservice.FormReference, 0, len(rows))
	for _, row := range rows {
		references = append(references, formservice.FormReference{
			ApplicationCode: row.ApplicationCode,
			ApplicationName: row.ApplicationName,
			EntryID:         row.EntryCode,
			EntryName:       row.EntryName,
			ParentEntryID:   row.ParentEntryCode,
		})
	}
	return references, nil
}

func (d formApplicationDirectory) ApplicationByID(ctx context.Context, id uint) (formservice.ApplicationView, bool, error) {
	app, err := d.applications.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return formservice.ApplicationView{}, true, nil
		}
		return formservice.ApplicationView{}, false, err
	}
	return formservice.ApplicationView{ID: app.ID, Status: app.Status}, false, nil
}

func (d formApplicationDirectory) ApplicationByCode(ctx context.Context, code string) (formservice.ApplicationView, bool, error) {
	app, err := d.applications.GetByCode(ctx, code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return formservice.ApplicationView{}, true, nil
		}
		return formservice.ApplicationView{}, false, err
	}
	return formservice.ApplicationView{ID: app.ID, Status: app.Status}, false, nil
}

// auditActorNamer 审计操作者显示名解析窄端口适配（000036 企业日志）：
// 审计域不反向依赖 iam，装配层以成员仓储桥接；解析失败返回空串（读取侧
// JOIN 当前昵称兜底），不阻断审计写入
type auditActorNamer struct {
	users repository.UserRepository
}

func (n auditActorNamer) MemberDisplayName(ctx context.Context, memberID uint) string {
	member, err := n.users.GetUserByID(ctx, memberID)
	if err != nil {
		return ""
	}
	return member.Nickname
}

// enterpriseLogMemberDirectory 企业日志成员目录窄端口适配：登录人/操作人
// 筛选的成员归属校验。以租户上下文包裹后经成员仓储查询——租户 Callback
// 自动过滤，跨租户/已删成员返回 NotFound → ErrMemberInvalid
type enterpriseLogMemberDirectory struct {
	users repository.UserRepository
}

func (d enterpriseLogMemberDirectory) ValidateMember(ctx context.Context, tenantID, memberID uint) error {
	member, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrMemberInvalid
		}
		return httpx.Wrap(httpx.NewBiz(httpx.CodeInternalServer, "成员校验失败", http.StatusInternalServerError), err)
	}
	_ = member
	return nil
}

// notificationMemberDirectory 消息中心成员目录窄端口适配：扇出前的成员
// 归属/有效状态复核与模板 actorName 固化（口径同企业日志，跨租户/已删成员
// NotFound → ErrMemberInvalid，Dispatcher 静默收敛扇出范围）
type notificationMemberDirectory struct {
	users repository.UserRepository
}

func (d notificationMemberDirectory) ValidateMember(ctx context.Context, tenantID, memberID uint) error {
	_, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.ErrMemberInvalid
		}
		return httpx.Wrap(httpx.NewBiz(httpx.CodeInternalServer, "成员校验失败", http.StatusInternalServerError), err)
	}
	return nil
}

func (d notificationMemberDirectory) MemberDisplayName(ctx context.Context, tenantID, memberID uint) string {
	member, err := d.users.GetUserByID(contextx.NewTenantContext(ctx, tenantID), memberID)
	if err != nil {
		return ""
	}
	return member.Nickname
}

// notificationAdminResolver 系统管理员窄端口适配：经内置系统管理员组的
// tenant-admin 角色绑定实时推导成员（单一事实源），不依赖可改名的角色名
type notificationAdminResolver struct {
	adminGroups repository.AdminGroupRepository
}

func (r notificationAdminResolver) ResolveAdminMemberIDs(ctx context.Context, tenantID uint) ([]uint, error) {
	tenantCtx := contextx.NewTenantContext(ctx, tenantID)
	roleID, err := r.adminGroups.ResolveBuiltinRoleID(tenantCtx)
	if err != nil {
		return nil, err
	}
	members, err := r.adminGroups.ListBuiltinMembers(tenantCtx, roleID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.ID)
	}
	return ids, nil
}

// applicationAssetNotifier 应用资产变更事件窄端口适配：application 域不
// 直接依赖 notification 域，装配层转发到事务 Outbox 发布端口；事件码与
// 参数 Schema 由 notification 事件注册表终审
type applicationAssetNotifier struct {
	publisher notificationservice.EventPublisher
}

func (n applicationAssetNotifier) NotifyAssetChanged(
	ctx context.Context, eventID, verb, appCode, appName string, actorMemberID uint,
) error {
	return n.publisher.PublishInTx(ctx, notificationservice.EventInput{
		EventID:       eventID,
		EventCode:     "application.asset.changed",
		ActorMemberID: actorMemberID,
		Parameters: map[string]string{
			"appName": appName,
			"verb":    verb,
			"appCode": appCode,
		},
	})
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

// buildEmailSender 按环境分派邮件通道。开发通道可支撑本地联调；release 环境
// 必须显式配置 SMTP，避免用户误以为验证码已真实送达。
func buildEmailSender(conf config.EmailConfig, env string) (emailauth.Sender, string, error) {
	if env == gin.ReleaseMode && conf.DevEcho {
		return nil, "", fmt.Errorf("release 环境 email.devEcho 必须关闭")
	}
	switch conf.Provider {
	case "", "dev":
		if env == gin.ReleaseMode {
			return nil, "", fmt.Errorf("release 环境邮箱验证码必须配置 email.provider=smtp")
		}
		return emailauth.NewDevSender(), emailauth.DevFixedCode, nil
	case "smtp":
		sender, err := emailauth.NewSMTPSender(
			conf.SMTPHost,
			conf.SMTPPort,
			conf.SMTPUsername,
			conf.SMTPPassword,
			conf.SMTPFrom,
			conf.SMTPImplicitTLS,
		)
		if err != nil {
			return nil, "", fmt.Errorf("smtp 邮件通道配置无效: %w", err)
		}
		return sender, "", nil
	default:
		return nil, "", fmt.Errorf("邮件服务商 provider=%q 尚未支持（当前支持 dev/smtp）", conf.Provider)
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
