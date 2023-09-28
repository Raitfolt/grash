package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/Raitfolt/grash/closer"
)

const (
	listenAddr      = "127.0.0.1:8080"
	shutdownTimeout = 5 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx); err != nil {
		log.Fatal(err)
	}

}

func runServer(ctx context.Context) error {
	var (
		mux = http.NewServeMux()
		srv = &http.Server{
			Addr:    listenAddr,
			Handler: mux,
		}
		c = &closer.Closer{}
	)

	mux.Handle("/", handleIndex())

	c.Add(srv.Shutdown)

	c.Add(func(ctx context.Context) error {
		time.Sleep(6 * time.Second)

		return nil
	})

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	log.Printf("listening on %s", listenAddr)
	<-ctx.Done()

	log.Println("shutting down server gracefully")

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
