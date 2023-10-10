package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Raitfolt/grash/internal/closer"
	"github.com/Raitfolt/grash/internal/logger"
	"go.uber.org/zap"
)

const (
	listenAddr      = "127.0.0.1:8080"
	shutdownTimeout = 5 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	//TODO: change to createLogger(ctx context.Context) error
	logger := logger.New()
	defer logger.Sync()

	if err := runServer(ctx, logger); err != nil {
		logger.Fatal(err.Error())
	}
}

func runServer(ctx context.Context, log *zap.Logger) error {
	var (
		mux = http.NewServeMux()
		srv = &http.Server{
			Addr:    listenAddr,
			Handler: mux,
		}
		c = closer.New()
	)

	mux.Handle("/", handleIndex())

	c.Add(srv.Shutdown)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen and serve:", zap.String("error", err.Error()))
		}
	}()

	log.Info("listening", zap.String("port", listenAddr))
	<-ctx.Done()

	log.Info("shutting down server gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	//  Если один из обработчиков
	// зависнет на время, достаточное для выхода за таймаут, Closer не вызовет
	// все последующие, т.к. при выходе за таймаут, процедура завершения программы
	// мгновенно останавливается.
	if err := c.Close(shutdownCtx); err != nil {
		return fmt.Errorf("closer: %v", err)
	}

	return nil
}

func handleIndex() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})
}
