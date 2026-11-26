package server

import (
	"context"
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
	"github.com/michael-duren/career-strategy/internal/scheduler"
	"net/http"
	"sync"
	"time"
)

type attempt struct {
	count int
	until time.Time
}
type Server struct {
	db       *database.Store
	config   config.Config
	mu       sync.Mutex
	attempts map[string]attempt
	// now overrides the clock in tests.
	now func() time.Time
	// rand overrides the spot-check draw in [0, 1) in tests.
	rand func() float64
	// ntfyClient sends Leetgrinder test notifications; nil uses the default.
	ntfyClient            *http.Client
	schedulerGoogleClient *scheduler.GoogleClient
	// schedulerTick overrides the scheduler worker's tick in tests.
	schedulerTick func(context.Context, time.Time) error
}

func NewServer(c config.Config, db *database.Store) *http.Server {
	s := &Server{db: db, config: c, attempts: map[string]attempt{}}
	return &http.Server{Addr: c.ListenAddr, Handler: s.RegisterRoutes(), IdleTimeout: time.Minute, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 2 * time.Minute, WriteTimeout: 3 * time.Minute, MaxHeaderBytes: 32 << 10}
}
