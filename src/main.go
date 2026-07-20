package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ChatDetectiveORG/payment-service/src/application"
	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/config"
	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/postgresql"
	"github.com/ChatDetectiveORG/payment-service/src/infrastructure/rabbitmq"
	utils "github.com/ChatDetectiveORG/shared/utils"
)

func main() {
	cfg, err := config.FetchConfig()
	if !err.IsNil() {
		log.Fatal(err.JSON())
	}
	if keyErr := utils.ValidateMasterKeyFromEnv(); !keyErr.IsNil() {
		log.Fatal(keyErr.JSON())
	}

	err = rabbitmq.InitRabbitMQ(cfg, rabbitmq.RequiredModels)
	if !err.IsNil() {
		log.Fatal(err.JSON())
	}

	err = postgresql.InitPostgresql()
	if !err.IsNil() {
		log.Fatal(err.JSON())
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	wg := &sync.WaitGroup{}
	log.Println("Service started. Listening to payment queues...")
	err = application.ListenToRabbitmq(cfg, ctx, wg)
	if !err.IsNil() {
		log.Fatal(err.JSON())
	}

	<-ctx.Done()
	log.Println("Shutdown signal received. Exiting...")

	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
	case <-time.After(30 * time.Second):
		log.Println("Timeout reached while waiting for WaitGroup, exiting forcefully")
	}

	log.Println("Service stopped")
}
