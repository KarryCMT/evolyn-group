package middleware

import (
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"fmt"
	"net/http"

	"evolyn/internal/platform/iam/authorization"
	"evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// AuthorizationMiddleware 资源级鉴权；authorizer 由 server 显式注入（P0-4 拆单例）
//
// 鉴权拒绝的稳定码（ADR-008）：未登录 401/已登录无权限 403，
// member ID/资源等细节只在上方鉴权日志与错误日志中出现，不出网
var (
	ErrAuthRequired = httpx.NewBiz(httpx.CodeUnauthorized, "请先登录", http.StatusUnauthorized)
	ErrForbidden    = httpx.NewBiz(httpx.CodeForbidden, "没有执行该操作的权限", http.StatusForbidden)
)

func AuthorizationMiddleware(authorizer *authorization.Authorizer) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := ginctx.GetUser(c)
		if user == nil {
			user = &model.User{}
		}

		ri := ginctx.GetRequestInfo(c)
		if ri == nil {
			httpx.ResponseFailed(c, http.StatusBadRequest, fmt.Errorf("failed to get request info"))
			c.Abort()
			return
		}

		if ri.IsResourceRequest {
			resource := ri.Resource
			ok, err := authorizer.Authorize(c.Request.Context(), user, ri)
			if err != nil {
				httpx.ResponseFailed(c, http.StatusInternalServerError, err)
				c.Abort()
				return
			}

			logrus.Infof("authorize member [%s(%d)], namespace [%s] resource [%s(%s)] verb [%s], result: %t",
				user.Nickname, user.ID, ri.Namespace, ri.Resource, ri.Name, ri.Verb, ok)

			if !ok {
				if user.ID == 0 {
					httpx.ResponseFailed(c, http.StatusUnauthorized, ErrAuthRequired)
				} else {
					// 被拒成员/资源细节经 Wrap 进日志，响应只见通用文案（ADR-008 脱敏）
					httpx.ResponseFailed(c, http.StatusForbidden, httpx.Wrap(ErrForbidden,
						fmt.Errorf("member [%d] is forbidden for resource %s in namespace %s", user.ID, resource, ri.Namespace)))
				}
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
