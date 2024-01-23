package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/httprate"
	jmw "github.com/johnweldon/middleware.go"
	"github.com/spf13/cobra"

	"github.com/tempusbreve/vanity/pkg/handlers"
)

// ServeCmd generates the web server command.
// ht: https://marcofranssen.nl/go-webserver-with-graceful-shutdown/
func ServeCmd() *cobra.Command {
	bindListen := "127.0.0.1:39999"
	jsonPath := "import_db.json"
	staticPath := ""

	srv := &cobra.Command{
		Use:   "serve",
		Short: "vanity import web server",
		RunE: func(c *cobra.Command, args []string) error {
			var (
				logger = log.New(c.OutOrStdout(), " vanityserver ", log.LstdFlags|log.Lshortfile|log.LUTC)
				quit   = make(chan os.Signal, 1)
				done   = make(chan bool, 1)
			)

			httpSrv := newServer(c, handler(c, logger), logger)

			signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
			go shutdown(httpSrv, quit, done, logger)

			logger.Printf("starting server; listening on %s", httpSrv.Addr)
			if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("error serving: %w", err)
			}

			<-done
			logger.Print("server stopped")

			return nil
		},
	}

	srv.Flags().StringVarP(&bindListen, "bind_listen", "b", bindListen, "set interface and port to listen on")
	srv.Flags().StringVarP(&jsonPath, "json_path", "j", jsonPath, "path to the JSON db")
	srv.Flags().StringVarP(&staticPath, "static_files", "s", staticPath, "fallback static files directory path")

	return srv
}

func shutdown(s *http.Server, quit <-chan os.Signal, done chan<- bool, logger *log.Logger) {
	<-quit

	logger.Print("shutting down")

	const timeout = 30 * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	s.SetKeepAlivesEnabled(false)

	if err := s.Shutdown(ctx); err != nil {
		if logger != nil {
			logger.Printf("error shutting down: %v", err)
		}
	}

	close(done)
}

func newServer(c *cobra.Command, h http.Handler, logger *log.Logger) *http.Server {
	var (
		err    error
		listen string
	)

	const (
		generalTimeout = 30 * time.Second
		maxHeaderBytes = 1 << 18
	)

	if listen, err = c.Flags().GetString("bind_listen"); err == nil {
		listen = "0.0.0.0:39999"
	}

	return &http.Server{
		Addr:           listen,
		Handler:        h,
		ReadTimeout:    generalTimeout,
		WriteTimeout:   generalTimeout,
		MaxHeaderBytes: maxHeaderBytes,
		ErrorLog:       logger,
	}
}

func handler(c *cobra.Command, logger *log.Logger) http.Handler {
	const (
		window = 10 * time.Minute
		limit  = 500
	)

	stores := []handlers.ImportStore{}

	jsonPath, err := c.Flags().GetString("json_path")
	if err != nil {
		jsonPath = "import_db.json"
	}

	if fi, err := os.Stat(jsonPath); err == nil {
		if !fi.IsDir() && fi.Size() > 0 {
			stores = append(stores, handlers.NewJSONStore(handlers.NewFileReader(jsonPath)))
		}
	}

	stores = append(stores, handlers.NewDNSStore(nil))

	var fallback http.Handler
	if staticPath, err := c.Flags().GetString("static_files"); err == nil {
		fallback = http.FileServer(http.Dir(staticPath))
		logger.Printf("fallback serving from: %q", staticPath)
	}

	importHandler := handlers.NewImportHandler(fallback, stores...)

	lmw := jmw.Logger(jmw.TextLevel("minimal"), c.OutOrStdout())

	r := chi.NewRouter()
	r.Use(lmw.Handler)
	r.Use(httprate.LimitByIP(limit, window))
	r.Handle("/logger/", http.StripPrefix("/logger", lmw.LevelHandler()))
	r.Method(http.MethodGet, "/healthz/", http.StripPrefix("/healthz", jmw.HealthzHandler()))
	r.Mount("/", importHandler)

	return r
}
