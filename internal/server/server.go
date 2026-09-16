package server

import (
	"github.com/michael-duren/career-strategy/internal/config"
	"github.com/michael-duren/career-strategy/internal/database"
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
}

func NewServer(c config.Config, db *database.Store) *http.Server {
	s := &Server{db: db, config: c, attempts: map[string]attempt{}}
	return &http.Server{Addr: c.ListenAddr, Handler: s.RegisterRoutes(), IdleTimeout: time.Minute, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 30 * time.Second, MaxHeaderBytes: 32 << 10}
}
