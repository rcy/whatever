package cli

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/web"
)

type ServeCmd struct {
	Port               string `default:"9999" env:"PORT"`
	BaseURL            string `env:"BASE_URL"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`
	SessionSecret      string `env:"SESSION_SECRET"`
}

func (c *ServeCmd) Run(app *app.App) error {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", c.Port)
	}

	mux, err := web.Server(app, web.Config{
		BaseURL:            baseURL,
		GoogleClientID:     c.GoogleClientID,
		GoogleClientSecret: c.GoogleClientSecret,
		SessionSecret:      c.SessionSecret,
	})
	if err != nil {
		return err
	}

	srv := http.Server{
		Addr:    ":" + c.Port,
		Handler: mux,
	}

	go func() {
		fmt.Printf("Listening on http://localhost:%s\n", c.Port)
		err := srv.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-signalCtx.Done()
	fmt.Println("Got signal")

	fmt.Println("Shutting down server...")

	shutdownCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		return err
	}
	fmt.Println("Shutting down server...done")

	return nil
}
