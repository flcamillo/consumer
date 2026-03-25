package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
)

// Esta aplicação receberá eventos do AWS EventBridge através
// de uma fila AWS SQS.
// Os eventos recebidos são referentes as ações para inicio ou
// encerramento de execução de conectores do AWS Transfer Family.
// A aplicação esta instrumentada com OpenTelemetry para coletar métricas,
// traces e logs dos componentes envolidos.
// Para que a telemetria seja coletada corretamente é esperado que seja
// informado via variável de ambiente o nome do serviço OTEL_SERVICE_NAME.
// Para configurar a aplicação também são esperadas as seguintes variáveis
// de ambiente:
// - SQS_QUEUE_URL: URL da fila AWS SQS para receber as mensagens que devem
// ser processadas.
// - KAFKA_BROKER: Endereço do servidor do Kafka dono do tópico que será lido.
// - KAFKA_TOPIC: Nome do tópico do Kafka para receber as mensagens que devem
// ser processadas.
// - KAFKA_CONSUMER_GROUP: Nome do grupo de consumo das mensagens do tópico do
// Kafka.
// - MAX_WORKERS: quantidade de workers para processar as mensagens recebidas
// da fila AWS SQS. O valor padrão é 10.

var (
	// encerramento do OTel SDK
	otelShutdown func(ctx context.Context) error
	// URL da fila AWS SQS para consumir mensagens.
	sqsQueueUrl string
	// Endereço do servidor do kafka.
	kafkaBroker string
	// Tópico do kafka para consumir mensagens.
	kafkaTopic string
	// Nome do grupo de consumo do tópico do kafka.
	kafkaConsumerGroup string
	// quantidade de workers para processar as mensagens recebidas da fila AWS SQS
	maxWorkers int
	// Client AWS SQS para executar operações na fila.
	sqsService SqsService
	// Cliente do Kafka para consumir mensagens da fila.
	kafkaService KafkaService
)

// Inicializa recursos essenciais da aplicação.
func init() {
	var err error
	ctx := context.Background()
	// agenda uma função para encerrar a aplicação e a telemetria caso ocorra qualquer erro
	defer func() {
		if err != nil {
			if otelShutdown != nil {
				err = otelShutdown(ctx)
				if err != nil {
					fmt.Printf("failed to shutdown OTel SDK, %s", err)
				}
			}
			os.Exit(1)
		}
	}()
	// inicializa o log padrão
	slog.SetDefault(otelslog.NewLogger(os.Getenv("OTEL_SERVICE_NAME")))
	// inicializa a telemetria
	otelShutdown, err = setupOTelSDK(context.Background())
	if err != nil {
		fmt.Printf("failed to setup OTel SDK, %s\n", err)
		return
	}
	// coleta metricas de runtime
	err = runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second))
	if err != nil {
		slog.Error("failed to setup runtime metrics",
			slog.Any("error", err),
		)
		return
	}
	// inicializa as configurações da aplicação
	sqsQueueUrl = os.Getenv("SQS_QUEUE_URL")
	if sqsQueueUrl == "" {
		err = fmt.Errorf("SQS_QUEUE_URL environment variable is not set")
		slog.Error(err.Error())
		return
	}
	kafkaBroker = os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		err = fmt.Errorf("KAFKA_BROKER environment variable is not set")
		slog.Error(err.Error())
		return
	}
	kafkaTopic = os.Getenv("KAFKA_TOPIC")
	if kafkaTopic == "" {
		err = fmt.Errorf("KAFKA_TOPIC environment variable is not set")
		slog.Error(err.Error())
		return
	}
	kafkaConsumerGroup = os.Getenv("KAFKA_CONSUMER_GROUP")
	if kafkaConsumerGroup == "" {
		err = fmt.Errorf("KAFKA_CONSUMER_GROUP environment variable is not set")
		slog.Error(err.Error())
		return
	}
	maxWorkers = 10
	if maxWorkersEnv := os.Getenv("MAX_WORKERS"); maxWorkersEnv != "" {
		n, err := strconv.Atoi(maxWorkersEnv)
		if err != nil {
			slog.Warn("failed to parse MAX_WORKERS",
				slog.String("MAX_WORKERS", maxWorkersEnv),
				slog.Any("error", err),
			)
		} else {
			maxWorkers = n
		}
	}
	// inicializa o sdk da AWS
	sdkConfig, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("failed to load AWS SDK config",
			slog.Any("error", err),
		)
		return
	}
	otelaws.AppendMiddlewares(&sdkConfig.APIOptions)
	sqsService = sqs.NewFromConfig(sdkConfig)
	// inicializa o cliente do Kafka
	kafkaService = kafka.NewReader(kafka.ReaderConfig{
		Dialer: &kafka.Dialer{
			Timeout: 10 * time.Second,
		},
		Brokers:        []string{kafkaBroker},
		GroupID:        kafkaConsumerGroup,
		Topic:          kafkaTopic,
		MaxBytes:       10 * 1024 * 1024,
		CommitInterval: time.Second,
	})
}

// Função principal.
func main() {
	// deve inicializar um context com cancelamento para receber sinais de término
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// exibe o resumo dos parametros recebidos
	slog.InfoContext(ctx, fmt.Sprintf("using sqs queue: %s", sqsQueueUrl))
	slog.InfoContext(ctx, fmt.Sprintf("using kafka broker: %s", kafkaBroker))
	slog.InfoContext(ctx, fmt.Sprintf("using kafka topic: %s", kafkaTopic))
	slog.InfoContext(ctx, fmt.Sprintf("using kafka consumer group: %s", kafkaConsumerGroup))
	slog.InfoContext(ctx, fmt.Sprintf("using kafka topic: %s", kafkaTopic))
	slog.InfoContext(ctx, fmt.Sprintf("using max workers: %d", maxWorkers))
	// inicia o processo de consumo e processamento de mensagens
	errChan := make(chan error, 1)
	messageChan := make(chan *MessageContext, maxWorkers)
	orchestrator := NewOrchestrator(&OrchestratorConfig{
		SqsQueueUrl:  sqsQueueUrl,
		SqsService:   sqsService,
		KafkaService: kafkaService,
		MessageChan:  messageChan,
		MaxWorkers:   maxWorkers,
	})
	wg := sync.WaitGroup{}
	wg.Go(func() {
		errChan <- orchestrator.Run(ctx)
	})
	// aguarda o sinal de término
	select {
	case <-ctx.Done():
		slog.WarnContext(ctx, "received shutdown signal")
	case err := <-errChan:
		if err != nil {
			slog.ErrorContext(ctx, "Orchestrator stopped",
				slog.Any("error", err),
			)
		}
	}
	wg.Wait()
	// encerra a telemetria
	err := otelShutdown(context.Background())
	if err != nil {
		fmt.Printf("failed to shutdown OTel SDK, %s\n", err)
	}
}
