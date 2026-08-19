package controller

import (
	"evolyn/internal/platform/httpx"
	"evolyn/internal/version"

	"github.com/gin-gonic/gin"
)

// callingCodeText 区号文案（多语言）：键名与结构对齐简道云 conf 接口，
// 便于前端对照消费
type callingCodeText struct {
	ZhCn string `json:"zh_cn"`
	EnUs string `json:"en_us"`
	ZhTw string `json:"zh_tw"`
}

// callingCode 单个区号项：value 为 E.164 前缀（如 +86）
type callingCode struct {
	Text  callingCodeText `json:"text"`
	Value string          `json:"value"`
}

// callingCodeGroup 区号分组（一期单组，label 留空，结构预留分组扩展）
type callingCodeGroup struct {
	Label    string        `json:"label"`
	Children []callingCode `json:"children"`
}

// appConfResponse 应用配置：客户端（登录/注册页等）启动所需的区号列表与
// 平台能力开关。字段命名对齐简道云 conf 接口形态，能力开关只下发已落地
// 能力，随里程碑逐步扩展（captcha/pki 等未落地项不下发）
type appConfResponse struct {
	Version         string             `json:"version"`
	CallingCodeList []callingCodeGroup `json:"calling_code_list"`
	// 能力开关（均为已落地能力）
	TenantRegister  bool `json:"tenant_register"`  // 自助开通租户（注册向导「创建团队」）
	PlatformSms     bool `json:"platform_sms"`     // 平台短信验证码通道（登录/注册）
	RegisterPersona bool `json:"register_persona"` // 注册引导画像（完善信息）
}

// callingCodes 手机区号静态数据。注意：短信验证码通道目前仅支持中国大陆
// 号段（^1[3-9]\d{9}$），港澳台区号先供前端展示选择，通道支持后端不变
var callingCodes = []callingCode{
	{Text: callingCodeText{ZhCn: "中国 +86", EnUs: "China +86", ZhTw: "中國 +86"}, Value: "+86"},
	{Text: callingCodeText{ZhCn: "中国台湾 +886", EnUs: "Taiwan +886", ZhTw: "中國台灣 +886"}, Value: "+886"},
	{Text: callingCodeText{ZhCn: "中国香港 +852", EnUs: "Hong Kong +852", ZhTw: "中國香港 +852"}, Value: "+852"},
	{Text: callingCodeText{ZhCn: "中国澳门 +853", EnUs: "Macao +853", ZhTw: "中國澳門 +853"}, Value: "+853"},
}

// AppConfController 应用配置（公开域，匿名可访问）：/app/conf 为非资源
// 路径，经全局链（认证仅解析不强制）即可匿名到达
type AppConfController struct{}

// NewAppConfController 构造应用配置控制器（无外部依赖，数据为静态配置）
func NewAppConfController() *AppConfController { return &AppConfController{} }

// @Summary 应用配置
// @Description 客户端启动配置（匿名可访问）：服务版本、手机区号列表（三语文案，
// 形态对齐简道云 conf 接口）与平台能力开关（仅下发已落地能力，随里程碑扩展）
// @Produce json
// @Tags 应用配置
// @Success 200 {object} httpx.Response{data=controller.appConfResponse}
// @Router /api/v1/app/conf [get]
func (a *AppConfController) GetConf(c *gin.Context) {
	httpx.ResponseSuccess(c, appConfResponse{
		Version:         version.Get().Version,
		CallingCodeList: []callingCodeGroup{{Label: "", Children: callingCodes}},
		TenantRegister:  true,
		PlatformSms:     true,
		RegisterPersona: true,
	})
}

func (a *AppConfController) RegisterRoute(api *gin.RouterGroup) {
	api.GET("/app/conf", a.GetConf)
}

func (a *AppConfController) Name() string { return "AppConf" }
