package websocket_api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"
)

// ///////////////////////////////////////////////////////
// persistence
// ///////////////////////////////////////////////////////
type Service struct {
	DB        *sql.DB
	SocketMgr SocketManager
}

type Connection struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

func NewDB(c *Connection) *sql.DB {
	connectionString := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		c.Host, c.Port, c.User, c.Password, c.DBName)
	pool, err := sql.Open("postgres", connectionString)
	if err != nil {
		slog.Error("Could not connect to DB - invalid parameters")
		panic("Invalid database connection string")
	}

	if err := pool.Ping(); err != nil {
		slog.Error(fmt.Sprintf("Error pinging database:\n %v", err))
		panic("Could not connect to database")
	}

	slog.Info("Connected to DB")
	return pool
}

// ///////////////////////////////////////////////////////
// XX
// ///////////////////////////////////////////////////////
type MessagePusher interface {
	Send(string) error
}

type SocketManager interface {
	Create(string) (MessagePusher, error)
	Delete(string) error
	GetLiveConnections() ([]MessagePusher, error)

	//Send(WSConnection, string) error
}

type WSConnection struct {
	ConnectionID string
}

// ///////////////////////////////////////////////////////
// Websocket
// ///////////////////////////////////////////////////////
type Command interface {
	Exec(*Service) error
}

type Broadcast struct {
	Action  string `json:"action"`
	Message string `json:"message"`
	From    string `json:"from"`
}

func (b Broadcast) Exec(s *Service) error {
	slog.Info("Broadcasting!!")

	// Broadcast messages....
	// For each known client, send a message

	liveConnections, _ := s.SocketMgr.GetLiveConnections()
	slog.Info(fmt.Sprintf("Live Connections: %v", liveConnections))
	for _, c := range liveConnections {
		slog.Info(fmt.Sprintf("Messaging to %v", c))
		msg, _ := json.Marshal(b)
		c.Send(string(msg))
	}

	return nil
}
