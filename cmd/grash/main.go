package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Raitfolt/grash/internal/closer"
	"github.com/Raitfolt/grash/internal/logger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func main() {
	logger := logger.New()
	defer logger.Sync()

	if err := godotenv.Load("../../config/vars.env"); err != nil {
		logger.Warn(err.Error())
	}

	listenAddr := getEnv("ADDRESS", "")
	if listenAddr == "" {
		logger.Error("Unable to find ADDRESS variable")
	} else {
		logger.Info("Env", zap.String("Listen address", listenAddr))
	}

	shutdownTimeoutEnv := getEnv("SHUTDOWN_TIMEOUT", "")
	var shutdownTimeout time.Duration
	if shutdownTimeoutEnv == "" {
		logger.Error("Unable to find SHUTDOWN_TIMEOUT variable")
	} else {
		secs, err := strconv.Atoi(shutdownTimeoutEnv)
		if err != nil {
			logger.Error("Can't convert SHUTDOWN_TIMEOUT to int")
		}
		shutdownTimeout = time.Second * time.Duration(secs)
		logger.Info("Env", zap.Any("Shutdown timeout", shutdownTimeout))
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx, logger, listenAddr, shutdownTimeout); err != nil {
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
