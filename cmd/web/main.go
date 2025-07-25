package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"stripe/internal/driver"
	"stripe/internal/models"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
)

const version = "1.0.0"
const cssVersion = "1"

var session *scs.SessionManager

type config struct {
	port int
	env  string
	api  string
	db   struct {
		dsn string
	}
	stripe struct {
		secret string
		key    string
	}
	secretkey string
	frontend  string
}

type application struct {
	config        config
	infoLog       *log.Logger
	errorLog      *log.Logger
	templateCache map[string]*template.Template
	version       string
	DB            models.DBModel
	Session       *scs.SessionManager
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		// create a logger for the server from the standard logger
		// ErrorLog: log.New(app.errorLog, "", 0),
		// IdleTimeout: 30 * time.Second,
		// ReadTimeout:  5 * time.Second,
		// WriteTimeout: 10 * time.Second,
	}

	app.infoLog.Printf("Starting HTTP server in %s mode on port %d", app.config.env, app.config.port)
	return srv.ListenAndServe()
}

func main() {
	gob.Register(TransactionData{}) // register for the session
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "Server port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application environment {development|production}")
	flag.StringVar(&cfg.db.dsn, "dsn", "amber:secret@tcp(localhost:3306)/widgets?parseTime=true&tls=false", "DSN")
	flag.StringVar(&cfg.api, "api", "http://localhost:4001", "URL for the api")
	flag.StringVar(&cfg.secretkey, "secretkey", "bRWmrwNUTqNUuzckjxsFlHZjxHkjrzKP", "secret key")
	flag.StringVar(&cfg.frontend, "frontend", "http://localhost:4000", "url of front end")

	flag.Parse()

	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := driver.OpenDB(cfg.db.dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer conn.Close()

	// set up session
	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Store = mysqlstore.New(conn)

	// create a template cache
	tc := make(map[string]*template.Template)

	// get all the files named *.page.tmpl from ./templates
	app := &application{
		config:        cfg,
		infoLog:       infoLog,
		errorLog:      errorLog,
		templateCache: tc,
		version:       version,
		DB:            models.DBModel{DB: conn},
		Session:       session,
	}

	// start the go routine in the background
	go app.ListenToWsChannel()

	err = app.serve()
	if err != nil {
		app.errorLog.Println(err)
		log.Fatal(err)
	}
}
