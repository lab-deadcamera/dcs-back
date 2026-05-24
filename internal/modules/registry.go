// Package modules provides the Module interface and Registry for organizing
// feature modules.
//
// Each module is a self-contained unit that registers its own routes.
// All module routes are protected by default — auth middleware is applied
// to the entire module group. Modules can expose public routes where needed.
//
// # How to create a new module
//
// 1. Create a package under internal/ with your handler/service/store
// 2. Add a module.go file that exposes a Register function:
//
//	type Module struct { name string; hdl *Handler }
//
//	func NewModule(hdl *Handler) *Module {
//	    return &Module{name: "your-resource", hdl: hdl}
//	}
//
//	func (m *Module) Register(rg *gin.RouterGroup, authMw, adminMw gin.HandlerFunc) {
//	    protected := rg.Group(m.name)
//	    protected.Use(authMw)
//	    {
//	        protected.GET("", m.hdl.List)
//	        protected.POST("", m.hdl.Create)
//	        // ...
//	    }
//	}
//
// 3. In main.go, register the module with the Registry.
//
// # Public routes (no auth)
//
// Create a sub-group without the auth middleware:
//
//	public := rg.Group(m.name)
//	public.POST("/login", m.hdl.SomePublicHandler)
//
//	protected := rg.Group(m.name)
//	protected.Use(authMw)
//	protected.GET("/profile", m.hdl.SomeProtectedHandler)
package modules

import "github.com/gin-gonic/gin"

// Module defines a self-contained feature module that registers its own routes.
type Module interface {
	// Name returns the module name (for debugging/logging).
	Name() string
	// Register sets up routes on the API v1 router group.
	// authMw is the JWT authentication middleware.
	// adminMw checks for admin role (RequireRole(1)).
	Register(rg *gin.RouterGroup, authMw gin.HandlerFunc, adminMw gin.HandlerFunc)
}

// Registry holds all registered modules and sets them up on the router.
type Registry struct {
	modules []Module
}

// NewRegistry creates an empty module registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a module to the registry.
func (r *Registry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// Setup iterates all modules and calls Register on each.
func (r *Registry) Setup(v1 *gin.RouterGroup, authMw, adminMw gin.HandlerFunc) {
	for _, m := range r.modules {
		m.Register(v1, authMw, adminMw)
	}
}
