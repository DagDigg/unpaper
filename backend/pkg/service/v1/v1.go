package v1

import (
	"database/sql"

	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	"github.com/DagDigg/unpaper/backend/pkg/chat"
	chatService "github.com/DagDigg/unpaper/backend/pkg/chat/service"
	"github.com/DagDigg/unpaper/backend/pkg/notifications"
	"github.com/DagDigg/unpaper/backend/pkg/usersession"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/DagDigg/unpaper/core/session"
	"github.com/Masterminds/squirrel"
	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	// postgres driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

const (
	// apiVersion is version of API is provided by server
	apiVersion = "v1"
)

// unpaperServiceServer is implementation of v1.UnpaperServiceServer proto interface
type unpaperServiceServer struct {
	db          *sql.DB
	cfg         *config.Config
	sb          squirrel.StatementBuilderType
	sm          *session.Manager
	rdb         *redis.Client
	chat        chat.Controller
	nm          notifications.SendListenReceiver
	usersession usersession.Sessioner
}

// Server defines the grpc server
type Server interface {
	v1API.UnpaperServiceServer
	GetDB() *sql.DB
	GetSB() squirrel.StatementBuilderType
	GetRDB() *redis.Client
	GetSM() *session.Manager
	GetNM() notifications.SendListenReceiver
	GetChat() chat.Controller
}

// NewUnpaperServiceServer creates Unpaper service
func NewUnpaperServiceServer(cfg *config.Config) (Server, error) {
	db, err := sql.Open("pgx", cfg.GetDBConnURL().String())
	if err != nil {
		return nil, err
	}
	sb := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db)

	opt, err := redis.ParseURL(cfg.GetRDBConnURL().String())
	if err != nil {
		return nil, err
	}
	rdb := redis.NewClient(opt)

	nm := notifications.NewManager(db, rdb)
	ch := chatService.New(rdb)
	sm := session.NewManager(rdb)
	usrsession := usersession.NewManager(rdb)

	return &unpaperServiceServer{
		db:          db,
		cfg:         cfg,
		sb:          sb,
		sm:          sm,
		rdb:         rdb,
		chat:        ch,
		nm:          nm,
		usersession: usrsession,
	}, nil
}

// checkAPI checks if the API version requested by client is supported by server
func (s *unpaperServiceServer) checkAPI(api string) error {
	// API version is "" means use current version of the service
	if len(api) > 0 {
		if apiVersion != api {
			return status.Errorf(codes.Unimplemented, "unsupported API version: service implements API version '%s', but asked for '%s'", apiVersion, api)
		}
	}
	return nil
}

// GetDB returns the server's database
func (s *unpaperServiceServer) GetDB() *sql.DB {
	return s.db
}

// GetSB returns the squirrel StatementBuilder of 's' for executing DB queries
func (s *unpaperServiceServer) GetSB() squirrel.StatementBuilderType {
	return s.sb
}

// GetRDB returns redis client instance
func (s *unpaperServiceServer) GetRDB() *redis.Client {
	return s.rdb
}

// GetChat returns the chat controller instance
func (s *unpaperServiceServer) GetChat() chat.Controller {
	return s.chat
}

// GetSM returns the session manager instance
func (s *unpaperServiceServer) GetSM() *session.Manager {
	return s.sm
}

// GetNM returns the underlying NotificationsManager instance
func (s *unpaperServiceServer) GetNM() notifications.SendListenReceiver {
	return s.nm
}
