package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/consumer"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"go.uber.org/zap"
)

func handleMarkCreated(ctx context.Context, event events.MarkEvent) error {
	fmt.Printf("[CREATE] User ID=%d", event.Payload.MarkID)

	return nil
}

func main() {
	log := logger.New()
	router := consumer.NewRouter(func(e events.MarkEvent) string {
		return e.Type
	})

	router.RegisterFunc(events.MarkCreated, handleMarkCreated)

	c := consumer.New(consumer.DefaultConfig().WithBrokers("localhost:9092").WithTopic("mark-service.events").WithGroupID("asd"), log)
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		fmt.Println("[CANCEL]")
		cancel()
	}()

	if err := c.Run(ctx, router.MessageHandler()); err != nil {
		log.Error("Ошибка", zap.Error(err))
	}
}
