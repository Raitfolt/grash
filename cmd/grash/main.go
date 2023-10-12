package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Raitfolt/grash/internal/closer"
	"github.com/Raitfolt/grash/internal/config"
	"github.com/Raitfolt/grash/internal/logger"
	"go.uber.org/zap"
)

func main() {
	logger := logger.New()
	defer logger.Sync()

	cfg := config.MustLoad(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx, logger, cfg.Address, cfg.ShutdownTimeout); err != nil {
		logger.Fatal(err.Error())
	}
}

func runServer(ctx context.Context, log *zap.Logger, listenAddr string, shutdownTimeout time.Duration) error {
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
		w.Write([]byte("Hello, World!"))
	})
}
