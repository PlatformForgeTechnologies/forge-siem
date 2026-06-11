package api

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"

	"forge-siem/internal/config"
)

type Server struct {
	cfg config.AppConfig
}

func New(cfg config.AppConfig) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Run(ctx context.Context) error {
	app := fiber.New(fiber.Config{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
		BodyLimit:    1 * 1024 * 1024,
	})
	s.routes(app)

	go func() {
		<-ctx.Done()
		_ = app.Shutdown()
	}()

	return app.Listen(s.cfg.ListenAddress)
}

func (s *Server) routes(app *fiber.App) {
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": s.cfg.ServiceName})
	})
	api := app.Group("/api", s.requireAuth)
	v1 := api.Group("/v1")
	v1.Get("/agents", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{
			{
				"id":            "agent-1",
				"hostname":      "ip-10-0-0-10",
				"status":        "active",
				"last_seen":     time.Now().UTC(),
				"groups":        []string{"eks", "production"},
				"os":            "Amazon Linux 2023",
				"agent_version": "0.1.0",
			},
		})
	})
	v1.Get("/agents/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"id":        c.Params("id"),
			"hostname":  "ip-10-0-0-10",
			"status":    "active",
			"last_seen": time.Now().UTC(),
		})
	})
	v1.Get("/alerts", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{
			{
				"id":         "alert-1",
				"rule_id":    "ssh-bruteforce",
				"agent_id":   "agent-1",
				"severity":   "high",
				"title":      "SSH brute force",
				"status":     "open",
				"created_at": time.Now().UTC(),
			},
		})
	})
	v1.Get("/alerts/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"id":         c.Params("id"),
			"severity":   "high",
			"title":      "SSH brute force",
			"status":     "open",
			"created_at": time.Now().UTC(),
		})
	})
	v1.Patch("/alerts/:id/status", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"id":     c.Params("id"),
			"status": "updated",
		})
	})
	v1.Post("/alerts/:id/respond", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"id":     c.Params("id"),
			"status": "queued",
		})
	})
	v1.Get("/vuln", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{})
	})
	v1.Get("/rules", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{
			{"id": "ssh-bruteforce", "title": "SSH brute force", "severity": "high"},
		})
	})
	v1.Post("/rules", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusCreated)
	})
	v1.Put("/rules/:id", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"id": c.Params("id"), "status": "updated"})
	})
	v1.Get("/stats/overview", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"agents_active": 12,
			"alerts_open":   4,
			"events_24h":    92834,
		})
	})
	v1.Get("/stats/mitre", func(c *fiber.Ctx) error {
		return c.JSON([]fiber.Map{
			{"technique": "T1110", "count": 3},
		})
	})
	app.Get("/ws/alerts", s.requireAuth, func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNotImplemented)
	})
}

func (s *Server) requireAuth(c *fiber.Ctx) error {
	if s.cfg.APIAuthToken == "" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error": "api authentication is not configured",
		})
	}
	token := c.Get("Authorization")
	expected := "Bearer " + s.cfg.APIAuthToken
	if token != expected {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	return c.Next()
}
