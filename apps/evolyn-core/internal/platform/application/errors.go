// Package application 应用管理域（M2-A）：空白应用创建/查询/更新/软删与
// 配额占位。稳定业务错误码集中定义于本包（ADR-008），调用方按 errCode
// 分支；内部细节经 httpx.Wrap 只入日志。模板安装（M2-B）与异步实例化
// （M2-C）的错误码（TEMPLATE_* / APP_PROVISION_FAILED）随对应批次补充
package application

import (
	"net/http"

	"evolyn/internal/platform/httpx"
)

var (
	// ErrNameInvalid 应用名称不符合要求（去首尾空格后 1–128 字符）
	ErrNameInvalid = httpx.NewBiz("APP_NAME_INVALID", "应用名称不符合要求", http.StatusBadRequest)

	// ErrIconInvalid 图标键不在服务端稳定枚举内（不存前端组件名）
	ErrIconInvalid = httpx.NewBiz("APP_ICON_INVALID", "应用图标配置无效", http.StatusBadRequest)

	// ErrColorInvalid 颜色键不在服务端稳定枚举内（不存 CSS 字面值）
	ErrColorInvalid = httpx.NewBiz("APP_COLOR_INVALID", "应用颜色配置无效", http.StatusBadRequest)

	// ErrMemberInvalid 操作者不是当前租户成员（跨租户成员绑定拦截，§9.3）
	ErrMemberInvalid = httpx.NewBiz("APP_MEMBER_INVALID", "当前成员不属于该租户", http.StatusForbidden)

	// ErrForbidden 应用域操作越权（Service 内部经 ApplicationAccessEvaluator
	// 复核，与鉴权中间件共用 FORBIDDEN 稳定码，前端无需新分支）
	ErrForbidden = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)

	// ErrQueryInvalid 列表查询参数非法（status 过滤值不在枚举内等）
	ErrQueryInvalid = httpx.NewBiz("APP_QUERY_INVALID", "应用查询参数无效", http.StatusBadRequest)

	// ErrCursorInvalid 分页游标非法（非 base64/缺字段），客户端应刷新列表重取
	ErrCursorInvalid = httpx.NewBiz("APP_CURSOR_INVALID", "分页游标无效，请刷新列表后重试", http.StatusBadRequest)

	// ErrNotFound 应用不存在或无权访问（租户过滤后的 NotFound 统一口径）
	ErrNotFound = httpx.NewBiz("APP_NOT_FOUND", "应用不存在或无权访问", http.StatusNotFound)

	// ErrStatusInvalid 状态流转不合法（仅 active↔archived，§7.1）
	ErrStatusInvalid = httpx.NewBiz("APP_STATUS_INVALID", "当前应用状态不支持此操作", http.StatusConflict)

	// ErrProvisioning 实例化进行中（pending/running 不可编辑/删除/进设计器）
	ErrProvisioning = httpx.NewBiz("APP_PROVISIONING", "应用正在初始化，请稍后重试", http.StatusConflict)
)

// 应用菜单稳定错误码（M2-菜单，方案 §9）：读取与分组创建已启用对应
// 错误；重排、移动、删除接口继续复用本组稳定码。
var (
	// ErrMenuNameInvalid 分组名称不符合要求（去首尾空格后 1–128 字符）
	ErrMenuNameInvalid = httpx.NewBiz("APP_MENU_NAME_INVALID", "分组名称不符合要求", http.StatusBadRequest)

	// ErrMenuNotFound 菜单节点不存在或无权访问（管理接口路径参数定位节点）
	ErrMenuNotFound = httpx.NewBiz("APP_MENU_NOT_FOUND", "菜单节点不存在或无权访问", http.StatusNotFound)

	// ErrMenuParentInvalid 菜单父节点无效（跨应用/跨租户/非分组/不存在）
	ErrMenuParentInvalid = httpx.NewBiz("APP_MENU_PARENT_INVALID", "菜单父节点无效", http.StatusBadRequest)

	// ErrMenuTargetInvalid 菜单关联的应用资产无效（不存在/跨应用/已软删）
	ErrMenuTargetInvalid = httpx.NewBiz("APP_MENU_TARGET_INVALID", "菜单关联的应用资产无效", http.StatusBadRequest)

	// ErrMenuDepthExceeded 菜单层级超过当前限制（分组两层/总深度三层）
	ErrMenuDepthExceeded = httpx.NewBiz("APP_MENU_DEPTH_EXCEEDED", "菜单层级超过当前限制", http.StatusBadRequest)

	// ErrMenuOrderInvalid 重排请求无效（同级集合不一致/编码非法）
	ErrMenuOrderInvalid = httpx.NewBiz("APP_MENU_ORDER_INVALID", "菜单排序请求无效", http.StatusBadRequest)

	// ErrMenuVersionConflict 菜单修订号冲突（baseMenuRevision 不匹配），客户端刷新后重试
	ErrMenuVersionConflict = httpx.NewBiz("APP_MENU_VERSION_CONFLICT", "应用菜单已更新，请刷新后重试", http.StatusConflict)

	// ErrMenuInvalid 菜单树完整性故障（孤儿节点/父节点非分组/循环）：
	// 服务端数据损坏而非客户端冲突，读取路径返回 500 而非 409
	ErrMenuInvalid = httpx.NewBiz("APP_MENU_INVALID", "应用菜单数据异常，请联系管理员", http.StatusInternalServerError)

	// ErrMenuEntryRenameForbidden 资产节点不支持经菜单接口改名：资产节点
	// 名称/图标以资产域为事实源，须经对应资产接口修改（同事务同步回节点）
	ErrMenuEntryRenameForbidden = httpx.NewBiz("APP_MENU_ENTRY_RENAME_FORBIDDEN", "资产节点名称请通过对应资产接口修改", http.StatusBadRequest)

	// ErrMenuHiddenInvalid 分组节点不支持对成员隐藏：分组可见性由后代节点
	// 派生，隐藏语义仅对资产节点成立
	ErrMenuHiddenInvalid = httpx.NewBiz("APP_MENU_HIDDEN_INVALID", "仅资产节点支持对成员隐藏", http.StatusBadRequest)

	// ErrMenuMoveInvalid 移动请求无效（目标父节点自身/后代、层级超限等）
	ErrMenuMoveInvalid = httpx.NewBiz("APP_MENU_MOVE_INVALID", "菜单移动请求无效", http.StatusBadRequest)

	// ErrMenuFavoriteInvalid 收藏请求无效（应用/节点定位失败或跨应用不一致）
	ErrMenuFavoriteInvalid = httpx.NewBiz("APP_MENU_FAVORITE_INVALID", "收藏请求无效", http.StatusBadRequest)
)
