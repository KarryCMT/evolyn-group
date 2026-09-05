package request

import (
	"net/http"
	"strings"

	"evolyn/internal/utils/set"
)

const (
	NamespaceNone = ""
	NamespaceRoot = "root"
)

const (
	GetOperation    = "get"
	ListOperation   = "list"
	CreateOperation = "create"
	UpdateOperation = "update"
	PatchOperation  = "patch"
	DeleteOperation = "delete"
)

type RequestInfoResolver interface {
	NewRequestInfo(req *http.Request) (*RequestInfo, error)
}

// RequestInfo holds information parsed from the http.Request
type RequestInfo struct {
	IsResourceRequest bool
	Path              string
	Verb              string

	APIPrefix  string
	APIGroup   string
	APIVersion string
	Namespace  string
	// Resource is the name of the resource being requested.  This is not the kind.  For example: pods
	Resource    string
	Subresource string
	// Name is empty for some verbs, but if the request directly indicates a name (not in body content) then this field is filled in.
	Name string
	// Parts are the path parts for the request, always starting with /{resource}/{name}
	Parts []string
}

type RequestInfoFactory struct {
	APIPrefixes set.String
}

// TODO write an integration test against the swagger doc to test the RequestInfo and match up behavior to responses
// NewRequestInfo returns the information from the http request.  If error is not nil, RequestInfo holds the information as best it is known before the failure
// It handles both resource and non-resource requests and fills in all the pertinent information for each.
// Valid Inputs:
// Resource paths
// /apis/{api-group}/{version}/namespaces
// /api/{version}/namespaces
// /api/{version}/namespaces/{namespace}
// /api/{version}/namespaces/{namespace}/{resource}
// /api/{version}/namespaces/{namespace}/{resource}/{resourceName}
// /api/{version}/{resource}
// /api/{version}/{resource}/{resourceName}
//
// Special verbs with subresources:
// /api/{version}/watch/{resource}
// /api/{version}/watch/namespaces/{namespace}/{resource}
//
// NonResource paths
// /apis/{api-group}/{version}
// /apis/{api-group}
// /apis
// /api/{version}
// /api
// /healthz
// /
func (r *RequestInfoFactory) NewRequestInfo(req *http.Request) (*RequestInfo, error) {
	// start with a non-resource request until proven otherwise
	requestInfo := RequestInfo{
		IsResourceRequest: false,
		Path:              req.URL.Path,
		Verb:              strings.ToLower(req.Method),
	}

	currentParts := splitPath(req.URL.Path)
	if len(currentParts) < 3 {
		// return a non-resource request
		return &requestInfo, nil
	}

	if !r.APIPrefixes.Has(currentParts[0]) {
		// return a non-resource request
		return &requestInfo, nil
	}
	requestInfo.APIPrefix = currentParts[0]
	currentParts = currentParts[1:]

	requestInfo.IsResourceRequest = true
	requestInfo.APIVersion = currentParts[0]
	currentParts = currentParts[1:]

	switch req.Method {
	case "POST":
		requestInfo.Verb = CreateOperation
	case "GET", "HEAD":
		requestInfo.Verb = GetOperation
	case "PUT":
		requestInfo.Verb = UpdateOperation
	case "PATCH":
		requestInfo.Verb = PatchOperation
	case "DELETE":
		requestInfo.Verb = DeleteOperation
	default:
		requestInfo.Verb = ""
	}

	// URL forms: /namespaces/{namespace}/{kind}/*, where parts are adjusted to be relative to kind
	if currentParts[0] == "namespaces" {
		if len(currentParts) > 1 {
			requestInfo.Namespace = currentParts[1]

			// if there is another step after the namespace name and it is not a known namespace subresource
			// move currentParts to include it as a resource in its own right
			if len(currentParts) > 2 {
				currentParts = currentParts[2:]
			}
		}
	} else {
		requestInfo.Namespace = NamespaceRoot
	}

	// parsing successful, so we now know the proper value for .Parts
	requestInfo.Parts = currentParts

	// parts look like: resource/resourceName/subresource/other/stuff/we/don't/interpret
	switch {
	case len(requestInfo.Parts) >= 3:
		requestInfo.Subresource = requestInfo.Parts[2]
		fallthrough
	case len(requestInfo.Parts) >= 2:
		requestInfo.Name = requestInfo.Parts[1]
		fallthrough
	case len(requestInfo.Parts) >= 1:
		requestInfo.Resource = requestInfo.Parts[0]
	}

	// 表单数据列表为了保持公开路由的资源上下文，挂在
	// /forms/{formCode}/records；它并不是表单设计管理面。将这条唯一的
	// 子资源路由映射到 form-records，才能由记录数据面的 authenticated
	// 基线和后续 FormPermissionEvaluator 共同裁决，而不会要求 forms:get。
	// 接口为承载完整 Query DSL 已由 GET 改为 POST，但语义仍是查询：动词
	// 在此归一化为 get（命中 form-records:view 基线），不落入 POST→create
	// 的提交门，避免查询权限与提交权限耦合。
	if requestInfo.Resource == "forms" && requestInfo.Subresource == "records" && requestInfo.Name != "" {
		requestInfo.Resource = "form-records"
		requestInfo.Subresource = ""
		requestInfo.Verb = GetOperation
	}

	// if there's no name on the request and we thought it was a get before, then the actual verb is a list or a watch
	if len(requestInfo.Name) == 0 && requestInfo.Verb == GetOperation {
		requestInfo.Verb = ListOperation
	}

	return &requestInfo, nil
}

// splitPath returns the segments for a URL path.
func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return []string{}
	}
	return strings.Split(path, "/")
}
