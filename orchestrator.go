package main

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Define a configuração do Orchestrator.
type OrchestratorConfig struct {
	// URL da fila AWS SQS para consumir mensagens.
	SqsQueueUrl string
	// cliente do AWS SQS para consumir mensagens da fila AWS SQS.
	SqsService SqsService
	// cliente do Kafka para consumir mensagens do topico.
	KafkaService KafkaService
	// Canal para as menagens recebidas para processamento.
	MessageChan chan *MessageContext
	// Número máximo de workers para processar as mensagens da fila AWS SQS.
	MaxWorkers int
}

// Orchestrator é a estrutura principal que gerencia o processo de consumo e
// processamento de mensagens da fila AWS SQS.
type Orchestrator struct {
	// Configuração gerais.
	config *OrchestratorConfig
	// Tracer para criar spans de telemetria durante o processo.
	tracer trace.Tracer
	// SqsConsumer para consumir mensagens da fila AWS SQS.
	sqsConsumer *SqsConsumer
	// KafkaConsumer para consumir mensagens do tópico Kafka.
	kafkaConsumer *KafkaConsumer
	// Worker para processar as mensagens recebidas da fila AWS SQS.
	workers []*Worker
}

// Construtor para criar uma nova instância do Orchestrator.
func NewOrchestrator(config *OrchestratorConfig) *Orchestrator {
	var workers []*Worker
	configWorker := &WorkerConfig{
		MessageChan: config.MessageChan,
	}
	for i := 0; i < config.MaxWorkers; i++ {
		workers = append(workers, NewWorker(i, configWorker))
	}
	return &Orchestrator{
		config: config,
		tracer: otel.Tracer("Orchestrator"),
		sqsConsumer: NewSqsConsumer(&SqsConsumerConfig{
			SqsService:  config.SqsService,
			QueueUrl:    config.SqsQueueUrl,
			MessageChan: config.MessageChan,
		}),
		kafkaConsumer: NewKafkaConsumer(&KafkaConsumerConfig{
			KafkaService: config.KafkaService,
			MessageChan:  config.MessageChan,
		}),
		workers: workers,
	}
}

// Inicia o processo de consumo e processamento de mensagens.
func (p *Orchestrator) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// agenda uma funcão para aguardar todos os processos terminarem
	// os workers só serão encerrados com o fechamando do canal de mensagens
	wgWorker := sync.WaitGroup{}
	wgConsumer := sync.WaitGroup{}
	defer func() {
		cancel()
		wgConsumer.Wait()
		close(p.config.MessageChan)
		wgWorker.Wait()
	}()
	// incia todos os workers em processos separados
	for _, worker := range p.workers {
		wgWorker.Go(func() {
			worker.Run(ctx)
		})
	}
	// inicia os consumidores
	errChan := make(chan error, 2)
	wgConsumer.Go(func() {
		errChan <- p.sqsConsumer.Run(ctx)
	})
	wgConsumer.Go(func() {
		errChan <- p.kafkaConsumer.Run(ctx)
	})
	// aguarda por erro ou cancelamento
	select {
	case <-ctx.Done():
		return nil
	case err := <-errChan:
		return err
	}
}
