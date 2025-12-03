package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/serphona/serphona/backend/go/libs/platform-events/config"
	"github.com/serphona/serphona/backend/go/libs/platform-events/consumer"
	"github.com/serphona/serphona/backend/go/libs/platform-events/topics"
	"github.com/serphona/serphona/backend/go/libs/platform-events/types"
)

func main() {
	// Criar configuração
	cfg := config.LoadFromEnv()
	cfg.ServiceName = "example-consumer"
	cfg.GroupID = "example-consumer-group"
	cfg.Debug = true

	// Definir tópicos para consumir
	topicsToConsume := []string{
		topics.UserCreated,
		topics.TenantCreated,
		topics.AgentCreated,
	}

	// Criar consumer
	cons, err := consumer.New(cfg, topicsToConsume)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer cons.Close()

	// Registrar handler para eventos de usuário criado
	cons.Subscribe(topics.UserCreated, func(event *types.Event) error {
		log.Printf("Received UserCreated event: %+v", event)
		// Processar evento aqui
		return nil
	})

	// Registrar handler para eventos de tenant criado
	cons.Subscribe(topics.TenantCreated, func(event *types.Event) error {
		log.Printf("Received TenantCreated event: %+v", event)
		// Processar evento aqui
		return nil
	})

	// Registrar handler com filtro para eventos de agente criado
	cons.SubscribeWithFilter(
		topics.AgentCreated,
		func(event *types.Event) bool {
			// Filtrar apenas eventos de um tenant específico
			return event.TenantID == "tenant-789"
		},
		func(event *types.Event) error {
			log.Printf("Received filtered AgentCreated event: %+v", event)
			// Processar evento aqui
			return nil
		},
	)

	// Iniciar consumer
	if err := cons.Start(); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	log.Println("Consumer started. Press Ctrl+C to stop...")

	// Aguardar sinal de término
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	<-sigterm

	log.Println("Shutting down consumer...")
}
