package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Raitfolt/grash/internal/closer"
	"github.com/Raitfolt/grash/internal/config"
	"github.com/Raitfolt/grash/internal/logger"
	"go.uber.org/zap"
)

// LOG_PATH='./' CONFIG_PATH="./config/config.yaml" go run cmd/grash/main.go

func main() {
	logr := logger.New()
	defer func() {
		if err := logr.Sync(); err != nil {
			log.Fatal("logger sync is failed with error: %w", err)
		}
	}()
	cfg := config.MustLoad(logr)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx, logr, cfg.Address, cfg.ShutdownTimeout); err != nil {
		logr.Fatal(err.Error())
	}
}

// Logging is custom middleware for logging all request
func Logging(next http.Handler, log *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		t := time.Since(start).Nanoseconds()
		log.Info("Request",
			zap.String("method", r.Method),
			zap.String("host", r.Host),
			//TODO: add need headers to show
			zap.Int64("time", t))
	})
}

func runServer(ctx context.Context, log *zap.Logger, listenAddr string, shutdownTimeout time.Duration) error {
	var (
		mux = http.NewServeMux()
		srv = &http.Server{
			Addr:    listenAddr,
			Handler: Logging(mux, log),
		}
		c = closer.New()
	)

	mux.Handle("/", handleIndex())

	forClose := closer.Func{Name: "HTTP server", F: srv.Shutdown}
	c.Add(forClose)

	forClose = closer.Func{Name: "SQL connect", F: func(ctx context.Context) error {
		time.Sleep(1700 * time.Millisecond)
		return nil
	}}
	c.Add(forClose)

	forClose = closer.Func{Name: "Redis connect", F: func(ctx context.Context) error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}}
	c.Add(forClose)
	forClose = closer.Func{Name: "RabbitMQ connect", F: func(ctx context.Context) error {
		time.Sleep(600 * time.Millisecond)
		return nil
	}}
	c.Add(forClose)
	forClose = closer.Func{Name: "Another server", F: func(ctx context.Context) error {
		time.Sleep(1200 * time.Millisecond)
		return nil
	}}
	c.Add(forClose)

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("listen and serve:", zap.String("error", err.Error()))
		}
	}()

	log.Info("listening", zap.String("address", listenAddr))
	<-ctx.Done()

	log.Info("shutting down server gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	//  Если один из обработчиков
	// зависнет на время, достаточное для выхода за таймаут, Closer не вызовет
	// все последующие, т.к. при выходе за таймаут, процедура завершения программы
	// мгновенно останавливается.
	if err := c.Close(shutdownCtx, log); err != nil {
		return fmt.Errorf("closer: %v", err)
	}

	return nil
}

func handleIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		d := time.Now().Format(time.RFC1123Z)
		_, err := w.Write([]byte(d))
		if err != nil {
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		}
	})
}
