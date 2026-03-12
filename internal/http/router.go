package http

import (
	"sukumad/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/config"
	"sukumad/internal/handlers"
)

type Router struct {
	db  *pgxpool.Pool
	cfg config.Config
}

func NewRouter(db *pgxpool.Pool, cfg config.Config) *Router {
	return &Router{db: db, cfg: cfg}
}

func (rt *Router) Register(r *gin.Engine) {
	// Healthz
	r.GET("/healthz", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	authh := handlers.NewAuthHandler(rt.db, rt.cfg)
	r.POST("/api/auth/login", authh.Login)
	r.POST("/api/auth/refresh", authh.Refresh)
	r.POST("/api/auth/password/reset/request", authh.RequestPasswordReset)
	r.POST("/api/auth/password/reset/confirm", authh.ConfirmPasswordReset)

	// Handlers
	u := handlers.NewUserHandler(rt.db)
	pr := handlers.NewPermissionHandler(rt.db)
	rl := handlers.NewRoleHandler(rt.db)
	s := handlers.NewServerHandler(rt.db)
	req := handlers.NewRequestHandler(rt.db)
	dl := handlers.NewDeliveryHandler(rt.db)

	// Servers CRUD
	api := r.Group("/api")
	api.Use(middleware.Authn(rt.cfg))
	api.POST("/auth/logout", authh.Logout)
	{
		// Users
		api.GET("/users", middleware.RequirePerm("admin"), u.List)
		api.POST("/users", middleware.RequirePerm("admin"), u.Create)
		api.GET("/users/:id", middleware.RequirePerm("admin"), u.Get)
		api.PUT("/users/:id", middleware.RequirePerm("admin"), u.Update)
		api.DELETE("/users/:id", middleware.RequirePerm("admin"), u.Delete)
		api.GET("/users/:id/permissions", middleware.RequirePerm("admin"), u.ListEffectivePermissions)
		api.POST("/users/:id/permissions", middleware.RequirePerm("admin"), u.GrantPermission)
		api.DELETE("/users/:id/permissions", middleware.RequirePerm("admin"), u.RevokePermission)
		api.POST("/users/:id/role", middleware.RequirePerm("admin"), u.SetRole)
		api.POST("/users/:id/unlock", middleware.RequirePerm("admin"), u.Unlock)

		// Roles
		api.GET("/roles", middleware.RequirePerm("admin"), rl.List)
		api.POST("/roles", middleware.RequirePerm("admin"), rl.Create)
		api.PUT("/roles/:id", middleware.RequirePerm("admin"), rl.Update)
		api.DELETE("/roles/:id", middleware.RequirePerm("admin"), rl.Delete)
		api.POST("/roles/:id/permissions", middleware.RequirePerm("admin"), rl.GrantPermission)
		api.DELETE("/roles/:id/permissions", middleware.RequirePerm("admin"), rl.RevokePermission)

		// Permissions
		api.GET("/permissions", middleware.RequirePerm("admin"), pr.List)
		api.POST("/permissions", middleware.RequirePerm("admin"), pr.Create)
		api.PUT("/permissions/:id", middleware.RequirePerm("admin"), pr.Update)
		api.DELETE("/permissions/:id", middleware.RequirePerm("admin"), pr.Delete)

		api.GET("/servers", s.List)
		api.POST("/servers", s.Create)
		api.GET("/servers/:id", s.Get)
		api.PUT("/servers/:id", s.Update)
		api.DELETE("/servers/:id", s.Delete)

		// Requests CRUD + expand
		api.GET("/requests", req.List)
		api.POST("/requests", req.Create)
		api.GET("/requests/:id", req.Get)
		api.PUT("/requests/:id", req.Update)
		api.DELETE("/requests/:id", req.Delete)
		api.POST("/requests/:id/expand", req.Expand) // calls expand_request_to_deliveries

		// Deliveries CRUD (read-heavy)
		api.GET("/deliveries", dl.List)
		api.GET("/deliveries/:id", dl.Get)
		api.PATCH("/deliveries/:id", dl.Patch) // e.g., update status or next_run_at
		api.DELETE("/deliveries/:id", dl.Delete)
	}
}
