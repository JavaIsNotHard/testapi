package main

import (
	"bankapi/internal/data"
	"bankapi/internal/mailer"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

type config struct {
	env  string
	port int
	db   struct {
		dsn string
	}

	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	log    *slog.Logger
	models data.Models
	mailer mailer.Mailer
}

func main() {
	var cfg config

	flag.StringVar(&cfg.env, "env", "", "[production|development]")
	flag.IntVar(&cfg.port, "port", 8000, "port number for the server")
	flag.StringVar(&cfg.db.dsn, "db-dsn", fmt.Sprintf("postgres://postgres:postgres@localhost/bankapi?sslmode=disable"), "PostgreSQL DSN")

	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "ab2ecfccde9880", "SMTP username")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port number")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "31e563b1f75b1c", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender mail address")

	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	conn, err := openDB(cfg)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	defer func() {
		conn.Close(context.TODO())
		logger.Info("Closing database connection")
	}()

	logger.Info("database connection established")

	app := &application{
		config: cfg,
		log:    logger,
		models: data.NewModel(conn),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	srv := &http.Server{
		Addr:     fmt.Sprintf(":%d", cfg.port),
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
		Handler:  app.routes(),
	}

	logger.Info("Server listening on localhost :8000")
	err = srv.ListenAndServe()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func openDB(cfg config) (*pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), cfg.db.dsn)
	if err != nil {
		return nil, fmt.Errorf("Couldn't open DB", err)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	err = conn.Ping(ctx)
	if err != nil {
		conn.Close(context.TODO())
		return nil, fmt.Errorf("Couldn't connect to database", err)
	}

	return conn, nil
}
