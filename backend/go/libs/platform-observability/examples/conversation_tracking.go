package main

import (
	"context"
	"fmt"
	"time"

	obs "github.com/serphona/backend/go/libs/platform-observability"
	"github.com/serphona/backend/go/libs/platform-observability/config"
	"github.com/serphona/backend/go/libs/platform-observability/types"
)

func main() {
	// 1. Inicializar observabilidade
	cfg := &config.Config{
		ServiceName:          "agent-orchestrator",
		ServiceVersion:       "1.0.0",
		Environment:          "development",
		TracingEnabled:       true,
		MetricsEnabled:       true,
		LoggingEnabled:       true,
		ConversationTracking: true,
	}

	observer, err := obs.Init(cfg)
	if err != nil {
		panic(err)
	}
	defer observer.Shutdown(context.Background())

	// 2. Iniciar uma conversação
	ctx := context.Background()
	conversationID := obs.StartConversation(ctx, types.ConversationStart{
		TenantID:   "tenant-123",
		AgentID:    "agent-456",
		CustomerID: "customer-789",
		Channel:    "voice",
		Language:   "pt-BR",
		Metadata: map[string]string{
			"campaign_id": "summer-2025",
			"queue":       "support",
		},
	})

	fmt.Printf("Conversação iniciada: %s\n", conversationID)

	// 3. Rastrear interações

	// Saudação do agente
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:    "agent_message",
		Speaker: "agent",
		Content: "Olá! Seja bem-vindo ao suporte da Serphona. Como posso ajudá-lo hoje?",
		Metadata: map[string]string{
			"sentiment": "positive",
			"intent":    "greeting",
		},
	})

	time.Sleep(2 * time.Second) // Simular tempo de resposta

	// Resposta do cliente
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:       "customer_message",
		Speaker:    "customer",
		Content:    "Olá, estou com problemas para acessar minha conta.",
		Sentiment:  "neutral",
		Intent:     "account_access_issue",
		Confidence: 0.95,
	})

	time.Sleep(1 * time.Second)

	// Agente coleta informações
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:    "agent_message",
		Speaker: "agent",
		Content: "Entendo. Você pode me informar qual erro está recebendo ao tentar fazer login?",
		Metadata: map[string]string{
			"sentiment": "neutral",
			"intent":    "information_gathering",
		},
	})

	time.Sleep(3 * time.Second)

	// Cliente fornece detalhes
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:       "customer_message",
		Speaker:    "customer",
		Content:    "Aparece uma mensagem dizendo que minha senha está incorreta, mas tenho certeza que é a correta.",
		Sentiment:  "frustrated",
		Intent:     "password_issue",
		Confidence: 0.88,
	})

	// 4. Rastrear decisão do agente
	obs.TrackDecision(ctx, conversationID, types.Decision{
		DecisionType: "offer",
		Option:       "password_reset",
		Reason:       "Customer unable to login with current password",
		Context: map[string]string{
			"attempts":   "3",
			"last_login": "2025-11-28",
		},
	})

	time.Sleep(1 * time.Second)

	// Agente oferece solução
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:    "agent_message",
		Speaker: "agent",
		Content: "Vou enviar um link de redefinição de senha para seu e-mail cadastrado. Você receberá em alguns minutos.",
		Metadata: map[string]string{
			"sentiment": "helpful",
			"intent":    "solution_offer",
		},
	})

	time.Sleep(2 * time.Second)

	// Cliente aceita
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:       "customer_message",
		Speaker:    "customer",
		Content:    "Perfeito, muito obrigado!",
		Sentiment:  "positive",
		Intent:     "gratitude",
		Confidence: 0.98,
	})

	time.Sleep(1 * time.Second)

	// Despedida do agente
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:    "agent_message",
		Speaker: "agent",
		Content: "Por nada! Há mais alguma coisa em que posso ajudar?",
		Metadata: map[string]string{
			"sentiment": "positive",
			"intent":    "offer_additional_help",
		},
	})

	time.Sleep(1 * time.Second)

	// Cliente encerra
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:       "customer_message",
		Speaker:    "customer",
		Content:    "Não, era só isso mesmo. Obrigado!",
		Sentiment:  "positive",
		Intent:     "farewell",
		Confidence: 0.99,
	})

	// 5. Finalizar conversação
	err = obs.EndConversation(ctx, conversationID, types.ConversationEnd{
		Resolution: "solved",
		Rating:     5,
		Tags:       []string{"account", "password", "self-service"},
		Metadata: map[string]string{
			"resolution_time":          "120s",
			"first_contact_resolution": "true",
		},
	})

	if err != nil {
		fmt.Printf("Erro ao finalizar conversação: %v\n", err)
		return
	}

	fmt.Println("Conversação finalizada com sucesso!")

	// 6. Exemplo de conversação com transferência
	demonstrateTransfer(ctx)
}

func demonstrateTransfer(ctx context.Context) {
	fmt.Println("\n--- Exemplo de Transferência ---")

	conversationID := obs.StartConversation(ctx, types.ConversationStart{
		TenantID:   "tenant-123",
		AgentID:    "agent-100",
		CustomerID: "customer-999",
		Channel:    "chat",
		Language:   "pt-BR",
	})

	// Cliente tem problema técnico complexo
	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:       "customer_message",
		Speaker:    "customer",
		Content:    "Minha integração via API não está funcionando. Recebo erro 500.",
		Sentiment:  "frustrated",
		Intent:     "technical_issue",
		Confidence: 0.92,
	})

	time.Sleep(1 * time.Second)

	// Agente decide transferir
	obs.TrackDecision(ctx, conversationID, types.Decision{
		DecisionType: "transfer",
		Option:       "technical_support",
		Reason:       "Issue requires technical expertise beyond L1 support",
		Context: map[string]string{
			"issue_type": "api_integration",
			"severity":   "high",
		},
	})

	obs.TrackInteraction(ctx, conversationID, types.Interaction{
		Type:    "agent_message",
		Speaker: "agent",
		Content: "Vou transferir você para nossa equipe técnica especializada que poderá ajudar melhor com questões de API.",
	})

	// Finalizar com transferência
	obs.EndConversation(ctx, conversationID, types.ConversationEnd{
		Resolution: "transferred",
		Rating:     4,
		Tags:       []string{"api", "integration", "technical"},
	})

	fmt.Println("Conversação com transferência finalizada!")
}
