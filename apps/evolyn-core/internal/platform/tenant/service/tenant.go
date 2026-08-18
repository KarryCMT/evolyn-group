package service

import (
	"context"
	"fmt"
	"strconv"

	iammodel "evolyn/internal/platform/iam/model"
	iamrepository "evolyn/internal/platform/iam/repository"
	tenantmodel "evolyn/internal/platform/tenant/model"
	tenantrepository "evolyn/internal/platform/tenant/repository"

	"golang.org/x/crypto/bcrypt"
)

// 租户内基线角色名（与默认租户种子、db.sql 口径一致）
const (
	TenantAdminRole     = "tenant-admin"
	AuthenticatedRole   = "authenticated"
	UnAuthenticatedRole = "unauthenticated"
)

// tenantService 租户域服务：开通/查询/配置/生命周期流转（运营面）。
// 依赖 iam 仓储完成「开通即建 owner 成员 + 租户内系统组/角色种子」
type tenantService struct {
	tenantRepo tenantrepository.TenantRepository
	iam        *iamrepository.Repositories
}

func NewTenantService(tenantRepo tenantrepository.TenantRepository, iam *iamrepository.Repositories) TenantService {
	return &tenantService{
		tenantRepo: tenantRepo,
		iam:        iam,
	}
}

// OpenTenantRequest 开通租户请求：OwnerAccountID 与 Owner 账号信息二选一
type OpenTenantRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Plan string `json:"plan"`

	OwnerAccountID uint `json:"ownerAccountId"` // 复用既有账号

	// 新建 owner 账号（登录身份）
	OwnerName     string `json:"ownerName"`
	OwnerPhone    string `json:"ownerPhone"`
	OwnerEmail    string `json:"ownerEmail"`
	OwnerPassword string `json:"ownerPassword"`
}

// Validate 开通参数校验
func (r *OpenTenantRequest) Validate() error {
	if r.Code == "" || r.Name == "" {
		return fmt.Errorf("tenant code/name is required")
	}
	if r.Plan == "" {
		r.Plan = tenantmodel.PlanFree
	}
	if !tenantmodel.IsValidPlan(r.Plan) {
		return fmt.Errorf("unknown plan: %s", r.Plan)
	}
	if r.OwnerAccountID == 0 && (r.OwnerName == "" || len(r.OwnerPassword) < 6) {
		return fmt.Errorf("owner account id or owner name/password (>=6 chars) is required")
	}
	return nil
}

// Open 开通租户（P3-1）：租户 + owner 成员 + 租户内系统组/角色种子一步到位。
// 全程使用剥离租户的 ctx：运营者会话自带的租户上下文会污染新租户的数据写入
// 与按名查询（组/角色按租户过滤），此处以 Background 显式规避
func (s *tenantService) Open(ctx context.Context, req *OpenTenantRequest) (*tenantmodel.Tenant, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	bctx := context.Background()

	if _, err := s.tenantRepo.GetByCode(bctx, req.Code); err == nil {
		return nil, fmt.Errorf("tenant code %s already exists", req.Code)
	}

	// owner 账号：复用或新建
	var (
		ownerAccount *iammodel.Account
		err          error
	)
	if req.OwnerAccountID > 0 {
		ownerAccount, err = s.iam.Account().GetByID(bctx, req.OwnerAccountID)
	} else {
		var password []byte
		password, err = bcrypt.GenerateFromPassword([]byte(req.OwnerPassword), bcrypt.DefaultCost)
		if err == nil {
			ownerAccount, err = s.iam.Account().Create(bctx, &iammodel.Account{
				Name:     req.OwnerName,
				Nickname: req.OwnerName,
				Phone:    req.OwnerPhone,
				Email:    req.OwnerEmail,
				Password: string(password),
			})
		}
	}
	if err != nil {
		return nil, err
	}

	tenant := &tenantmodel.Tenant{
		Code:           req.Code,
		Name:           req.Name,
		Plan:           req.Plan,
		Status:         tenantmodel.TenantActive,
		OwnerAccountId: ownerAccount.ID,
		Config:         tenantmodel.DefaultTenantConfig(),
	}
	tenant, err = s.tenantRepo.Create(bctx, tenant)
	if err != nil {
		return nil, err
	}

	// owner 成员：显式指定租户归属
	nickname := ownerAccount.Nickname
	if nickname == "" {
		nickname = ownerAccount.Name
	}
	member := &iammodel.User{AccountId: ownerAccount.ID, Nickname: nickname}
	member.TenantID = tenant.ID
	if _, err = s.iam.User().Create(bctx, member); err != nil {
		return nil, err
	}

	if err = s.seedTenantBaseline(bctx, tenant.ID); err != nil {
		return nil, err
	}

	// owner 绑定租户管理员角色
	tenantAdmin, err := s.iam.RBAC().GetRoleByName(bctx, TenantAdminRole)
	if err != nil {
		return nil, err
	}
	if err = s.iam.User().AddRole(bctx, tenantAdmin, member); err != nil {
		return nil, err
	}

	return tenant, nil
}

// seedTenantBaseline 租户内基线种子：系统组×3 + 基础角色×3 + 组角色绑定。
// 显式写 TenantID（bctx 无租户上下文，避免落到列默认值）
func (s *tenantService) seedTenantBaseline(bctx context.Context, tenantID uint) error {
	roles := []iammodel.Role{
		{Name: TenantAdminRole, Rules: iammodel.Rules{
			{Resource: "users", Operation: iammodel.AllOperation},
			{Resource: "groups", Operation: iammodel.AllOperation},
			{Resource: "roles", Operation: iammodel.AllOperation},
			{Resource: "departments", Operation: iammodel.AllOperation},
		}},
		{Name: AuthenticatedRole, Rules: iammodel.Rules{
			{Resource: "users", Operation: iammodel.AllOperation},
			{Resource: "auth", Operation: iammodel.AllOperation},
		}},
		{Name: UnAuthenticatedRole, Rules: iammodel.Rules{
			{Resource: "auth", Operation: "create"},
		}},
	}
	for i := range roles {
		roles[i].TenantID = tenantID
		if _, err := s.iam.RBAC().Create(bctx, &roles[i]); err != nil {
			return err
		}
	}

	groups := []iammodel.Group{
		{Name: iammodel.RootGroup, Kind: iammodel.SystemGroup, Describe: "tenant root group"},
		{Name: iammodel.AuthenticatedGroup, Kind: iammodel.SystemGroup, Describe: "system group contains all authenticated user"},
		{Name: iammodel.UnAuthenticatedGroup, Kind: iammodel.SystemGroup, Describe: "system group contains all unauthenticated user"},
	}
	for i := range groups {
		groups[i].TenantID = tenantID
	}
	if err := s.iam.Group().CreateGroups(bctx, groups); err != nil {
		return err
	}

	// 组-角色绑定（按名取回带 ID 的对象，bctx 无租户过滤不适用——
	// 组/角色查询仍会按 ctx 过滤，此处直接用刚创建的切片对象）
	bindings := []struct{ groupName, roleName string }{
		{iammodel.RootGroup, TenantAdminRole},
		{iammodel.AuthenticatedGroup, AuthenticatedRole},
		{iammodel.UnAuthenticatedGroup, UnAuthenticatedRole},
	}
	for _, b := range bindings {
		var (
			group *iammodel.Group
			role  *iammodel.Role
		)
		for i := range groups {
			if groups[i].Name == b.groupName {
				group = &groups[i]
				break
			}
		}
		for i := range roles {
			if roles[i].Name == b.roleName {
				role = &roles[i]
				break
			}
		}
		if group == nil || role == nil {
			return fmt.Errorf("baseline seed mismatch: %s/%s", b.groupName, b.roleName)
		}
		if err := s.iam.Group().AddRole(bctx, role, group); err != nil {
			return err
		}
	}

	return nil
}

func (s *tenantService) List(ctx context.Context) ([]tenantmodel.Tenant, error) {
	return s.tenantRepo.List(ctx)
}

func (s *tenantService) Get(ctx context.Context, id string) (*tenantmodel.Tenant, error) {
	tid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return s.tenantRepo.GetByID(ctx, uint(tid))
}

func (s *tenantService) Update(ctx context.Context, id string, tenant *tenantmodel.Tenant) (*tenantmodel.Tenant, error) {
	tid, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	if tenant.Plan != "" && !tenantmodel.IsValidPlan(tenant.Plan) {
		return nil, fmt.Errorf("unknown plan: %s", tenant.Plan)
	}
	tenant.ID = uint(tid)
	return s.tenantRepo.Update(ctx, tenant)
}

// SetStatus 生命周期流转：active/frozen/deleted（架构文档 26.2）
func (s *tenantService) SetStatus(ctx context.Context, id string, status string) error {
	switch status {
	case tenantmodel.TenantActive, tenantmodel.TenantFrozen, tenantmodel.TenantDeleted:
	default:
		return fmt.Errorf("invalid tenant status: %s", status)
	}

	tid, err := strconv.Atoi(id)
	if err != nil {
		return err
	}
	return s.tenantRepo.UpdateStatus(ctx, uint(tid), status)
}
