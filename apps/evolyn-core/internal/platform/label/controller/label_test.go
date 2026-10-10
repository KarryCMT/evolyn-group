package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
)

// TestRegisterRouteUsesFormDomainWildcard verifies label routes can coexist with
// the form domain's canonical :code wildcard without making Gin panic at startup.
func TestRegisterRouteUsesFormDomainWildcard(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	api.GET("/forms/:code", func(*gin.Context) {})

	controller := &LabelController{}
	controller.RegisterRoute(api)

	want := map[string]bool{
		"GET /api/v1/forms/:code/labels/profile":       false,
		"POST /api/v1/forms/:code/labels/preview":      false,
		"POST /api/v1/forms/:code/labels/batch-render": false,
	}
	for _, route := range router.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, registered := range want {
		if !registered {
			t.Errorf("expected route %q to be registered", route)
		}
	}
}
