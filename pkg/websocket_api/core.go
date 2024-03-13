package websocket_api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/davecgh/go-spew/spew"
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
type Messageable interface {
	Address() string
	Name() string
	SendMessage(string) error
}

type SocketManager interface {
	Create(string) (Messageable, error)
	Delete(string) error
	GetConnections() ([]Messageable, error)
	Find(string) (Messageable, bool)
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

	liveConnections, _ := s.SocketMgr.GetConnections()
	slog.Info(fmt.Sprintf("Live Connections: %v", liveConnections))
	for _, c := range liveConnections {
		slog.Info(fmt.Sprintf("Messaging to %v", c))
		msg, _ := json.Marshal(b)
		c.SendMessage(string(msg))
	}

	return nil
}

type WhoAll struct {
	Action string        `json:"action"`
	Users  []Messageable `json:"users"`
	From   string        `json:"from"`
}

func (w WhoAll) Exec(s *Service) error {
	slog.Info("Getting all users!!")

	c, exists := s.SocketMgr.Find(w.From)
	if !exists {
		slog.Info(fmt.Sprintf("Could not find sender address: %v", w.From))
		return nil
	}

	liveConnections, _ := s.SocketMgr.GetConnections()
	slog.Info(fmt.Sprintf("Live Connections: %v", spew.Sdump(liveConnections)))
	w.Users = liveConnections

	resp, _ := json.Marshal(w)
	c.SendMessage(string(resp))

	return nil
}
