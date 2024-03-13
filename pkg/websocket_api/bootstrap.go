package websocket_api

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/gorilla/mux"
	"github.com/pressly/goose/v3"

	"go_websocket_demo/migrations"

	httpin_integration "github.com/ggicci/httpin/integration"

	_ "github.com/lib/pq"
)

type Environment struct {
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string
	DBPasswordArn string
	DBName        string
	AWSProfile    string
	LogLevel      string
}

// //////////////////////////////////////////////////////////////////
// Initialization
// //////////////////////////////////////////////////////////////////
func NewEnvironment() *Environment {

	var dbPort int
	var dbHost string
	var dbUser string
	var dbName string
	var dbPassword string
	var dbPasswordArn string
	var awsProfile string
	var logLevel string

	if port, exists := os.LookupEnv("DB_PORT"); exists {
		dbPort, _ = strconv.Atoi(port)
	} else {
		slog.Error("Missing `DB_PORT` environment variable")
	}

	if field, exists := os.LookupEnv("DB_HOST"); exists {
		dbHost = field
	} else {
		slog.Error("Missing `DB_HOST` environment variable")
	}

	if field, exists := os.LookupEnv("DB_USER"); exists {
		dbUser = field
	} else {
		slog.Error("Missing `DB_USER` environment variable")
	}

	if field, exists := os.LookupEnv("DB_NAME"); exists {
		dbName = field
	} else {
		slog.Error("Missing `DB_NAME` environment variable")
	}

	if field, exists := os.LookupEnv("DB_PASSWORD"); exists {
		dbPassword = field
	} else {
		slog.Warn("Missing `DB_PASSWORD` environment variable")
	}

	if field, exists := os.LookupEnv("DB_PASSWORD_ARN"); exists {
		dbPasswordArn = field
	} else {
		slog.Warn("Missing `DB_PASSWORD_ARN` environment variable")
	}

	if field, exists := os.LookupEnv("AWS_PROFILE"); exists {
		awsProfile = field
	} else {
		slog.Warn("Missing `AWS_PROFILE` environment variable")
	}

	if field, exists := os.LookupEnv("LOG_LEVEL"); exists {
		logLevel = field
	} else {
		slog.Warn("Missing `LOG_LEVEL` environment variable")
	}

	return &Environment{
		DBHost:        dbHost,
		DBPort:        dbPort,
		DBUser:        dbUser,
		DBPassword:    dbPassword,
		DBName:        dbName,
		DBPasswordArn: dbPasswordArn,
		AWSProfile:    awsProfile,
		LogLevel:      logLevel,
	}
}

func ResolveDBPassword(env *Environment) string {
	if env.DBPassword != "" {
		slog.Info("Using DB Password from Environment varibles")
		return env.DBPassword
	}

	slog.Info("Retrieving DB Password from Secretsmanager")
	var sess *session.Session
	var sessErr error

	if env.AWSProfile != "" {
		slog.Info(fmt.Sprintf("Using AWS_PROFILE=%v for session", env.AWSProfile))
		sess, sessErr = session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Profile:           env.AWSProfile,
		})
	} else {
		slog.Info("Creating new AWS session")
		sess, sessErr = session.NewSession()
	}

	if sessErr != nil {
		slog.Error(fmt.Sprintf("Could not create session\n%v", sessErr))
		panic("Error resolving DB password - could not create new AWS session")
	}
	slog.Info("Created AWS session")

	sm := secretsmanager.New(sess)
	slog.Info("Created SecretManager client")
	result, err := sm.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: &env.DBPasswordArn,
	})

	if err != nil {
		slog.Error(fmt.Sprintf("Could not retrieve secret\n%v", err))
		panic(fmt.Sprintf("Error resolving DB password - could not retrieve secret from '%v'\n"+
			"Did you set the `DB_PASSWORD_ARN` environment variable?", env.DBPasswordArn))
	}
	slog.Info("Retrieved secret")

	return *result.SecretString
}

func init() {
	// Register a directive named "path" to retrieve values from `mux.Vars`,
	// i.e. decode path variables.
	httpin_integration.UseGorillaMux("path", mux.Vars)
}

func MigrateDB(db *sql.DB) error {
	goose.SetBaseFS(migrations.EmbedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, "."); err != nil {
		return err
	}
	return nil
}

// //////////////////////////////////////////////////////////////////
// Entrypoint
// //////////////////////////////////////////////////////////////////
func NewService() *Service {
	env := NewEnvironment()

	conn := &Connection{
		Host:     env.DBHost,
		Port:     env.DBPort,
		User:     env.DBUser,
		Password: ResolveDBPassword(env),
		DBName:   env.DBName,
	}

	if env.LogLevel == "DEBUG" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	db := NewDB(conn)
	if err := MigrateDB(db); err != nil {
		slog.Error(fmt.Sprintf("Could not migrate DB. Aborting startup: %v", err))
		panic("Could not migrate DB. Aborting startup")
	}

	return &Service{
		DB: db,
	}
}
