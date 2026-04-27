package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ariga.io/atlas-go-sdk/atlasexec"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	financev1connect "github.com/kiridovg/lifepilot-finance-service/gen/finance/v1/financev1connect"
	"github.com/kiridovg/lifepilot-finance-service/internal/db"
	"github.com/kiridovg/lifepilot-finance-service/internal/handler"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Error("DATABASE_URL is not set")
		os.Exit(1)
	}

	if err := runMigrations(dbURL, log); err != nil {
		log.Error("migration failed", "err", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Error("failed to connect to database", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := db.New(pool)
	mux := http.NewServeMux()

	mux.Handle(financev1connect.NewExpenseServiceHandler(handler.NewExpenseHandler(q)))
	mux.Handle(financev1connect.NewTransferServiceHandler(handler.NewTransferHandler(pool)))
	mux.Handle(financev1connect.NewAccountServiceHandler(handler.NewAccountHandler(pool)))
	mux.Handle(financev1connect.NewCurrencyServiceHandler(handler.NewCurrencyHandler(pool)))
	mux.Handle(financev1connect.NewUserServiceHandler(handler.NewUserHandler(pool)))
	mux.Handle(financev1connect.NewCategoryServiceHandler(handler.NewCategoryHandler(pool)))
	mux.Handle(financev1connect.NewIncomeServiceHandler(handler.NewIncomeHandler(q)))

	addr := ":" + getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	go func() {
		log.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", "err", err)
	}
	log.Info("server stopped")
}

func runMigrations(dbURL string, log *slog.Logger) error {
	client, err := atlasexec.NewClient(".", "atlas")
	if err != nil {
		return err
	}

	for attempt := 1; attempt <= 5; attempt++ {
		res, err := client.MigrateApply(context.Background(), &atlasexec.MigrateApplyParams{
			URL:             dbURL,
			DirURL:          "file://internal/db/migrations",
			RevisionsSchema: "public",
		})
		if err != nil {
			if strings.Contains(err.Error(), "lock is held by other session") {
				log.Warn("migration lock busy, retrying", "attempt", attempt)
				time.Sleep(time.Duration(attempt) * 2 * time.Second)
				continue
			}
			return err
		}
		for _, m := range res.Applied {
			log.Info("migration applied", "version", m.Version, "description", m.Description)
		}
		if res.Current == res.Target {
			log.Info("migrations up to date", "version", res.Current)
		}
		return nil
	}
	return fmt.Errorf("migration lock not released after retries")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
