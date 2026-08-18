package middleware

import (
	"evolyn/internal/platform/ginctx"
	"evolyn/internal/platform/httpx"
	"fmt"
	"net/http"

	"evolyn/internal/authorization"
	"evolyn/internal/platform/iam/model"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

func AuthorizationMiddleware() gin.HandlerFunc {
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
			ok, err := authorization.Authorize(c.Request.Context(), user, ri)
			if err != nil {
				httpx.ResponseFailed(c, http.StatusInternalServerError, err)
				c.Abort()
				return
			}

			logrus.Infof("authorize user [%s(%d)], namespace [%s] resource [%s(%s)] verb [%s], result: %t",
				user.Name, user.ID, ri.Namespace, ri.Resource, ri.Name, ri.Verb, ok)

			if !ok {
				if user.Name == "" {
					httpx.ResponseFailed(c, http.StatusUnauthorized, nil)
				} else {
					httpx.ResponseFailed(c, http.StatusForbidden, fmt.Errorf("user [%s] is forbidden for resource %s in namespace %s", user.Name, resource, ri.Namespace))
				}
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
