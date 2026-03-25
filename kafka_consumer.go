package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	"go.opentelemetry.io/otel/trace"
)

// Define a estrutura para a configuração do processo responsável por consumir mensagens
type KafkaConsumerConfig struct {
	// Cliente do Kafka para consumir mensagens da fila.
	KafkaService KafkaService
	// Canal para enviar as menagens recebidas para processamento.
	MessageChan chan *MessageContext
}

// Define a estrutura para o processo responsável por consumir mensagens
// do Kafka.
type KafkaConsumer struct {
	// Configuração gerais.
	config *KafkaConsumerConfig
	// Tracer para criar spans de telemetria durante o processo.
	tracer trace.Tracer
	// metricas de requisições
	messageCounter       metric.Int64Counter
	messageWaitHistogram metric.Float64Histogram
}

// Construtor para criar uma nova instância do KafkaConsumer.
func NewKafkaConsumer(config *KafkaConsumerConfig) *KafkaConsumer {
	consumer := &KafkaConsumer{
		tracer: otel.Tracer("KafkaConsumer"),
		config: config,
	}
	// configura as metricas
	meter := otel.Meter("KafkaConsumer.metrics")
	if counter, err := meter.Int64Counter("custom.KafkaConsumer.messages.received",
		metric.WithDescription("The number of messages received from queue"),
		metric.WithUnit("{messages}")); err == nil {
		consumer.messageCounter = counter
	} else {
		panic(err)
	}
	if histogram, err := meter.Float64Histogram("custom.KafkaConsumer.messages.wait.duration",
		metric.WithDescription("Message waiting time before being queued for a worker."),
		metric.WithUnit("s")); err == nil {
		consumer.messageWaitHistogram = histogram
	} else {
		panic(err)
	}
	return consumer
}

// Método para iniciar o processo de consumo de mensagens do kafka.
func (p *KafkaConsumer) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "starting KafkaConsumer...")
	defer slog.WarnContext(ctx, "KafkaConsumer stopped")
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "stopping KafkaConsumer due to context cancellation...")
			return nil
		default:
			ctx, span := p.tracer.Start(ctx, "Waiting.For.Kafka.Messages")
			messages, err := p.receiveMessage(ctx)
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "failed to receive messages")
				slog.ErrorContext(ctx, "failed to receive messages",
					slog.Any("error", err),
				)
				span.End()
				return err
			}
			if len(messages) == 0 {
				span.End()
				continue
			}
			// envia as mensagens recebidas para o worker incrementando as
			// as metricas de quantidade de mensagens recebidas e também a
			// metrica que indica quanto tempo levou para a mensagem ser
			// enfileirada para o worker
			p.messageCounter.Add(ctx, int64(len(messages)))
			for _, msg := range messages {
				started := time.Now()
				p.config.MessageChan <- msg
				p.messageWaitHistogram.Record(ctx, time.Since(started).Seconds())
			}
			span.End()
		}
	}
}

// Recebe as mensagens do Kafka.
func (p *KafkaConsumer) receiveMessage(ctx context.Context) (messagesContext []*MessageContext, err error) {
	ctx, span := p.tracer.Start(ctx, "Kafka.FetchMessage", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		semconv.MessagingSystemKafka,
		semconv.MessagingOperationTypeReceive,
		semconv.MessagingConsumerGroupName(p.config.KafkaService.Config().GroupID),
		semconv.MessagingDestinationName(p.config.KafkaService.Config().Topic),
		semconv.ServerAddress(strings.Join(p.config.KafkaService.Config().Brokers, ", ")),
	)
	message, err := p.config.KafkaService.FetchMessage(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, nil
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch message from kafka")
		return nil, err
	}
	// define a função para fazer o commit da mensagem processada
	commitFunc := func(ctx context.Context) error {
		return p.commitMessage(ctx, message)
	}
	messagesContext = append(messagesContext, &MessageContext{
		Context:  ctx,
		Id:       uuid.NewString(),
		Body:     string(message.Value),
		Received: time.Now(),
		Source:   "Kafka",
		Commit:   commitFunc,
	})
	return messagesContext, nil
}

// Realiza o commit da mensagem recebida no Kafka.
func (p *KafkaConsumer) commitMessage(ctx context.Context, message kafka.Message) error {
	ctx, span := p.tracer.Start(ctx, "Kafka.CommitMessages", trace.WithSpanKind(trace.SpanKindClient))
	defer span.End()
	span.SetAttributes(
		semconv.MessagingSystemKafka,
		semconv.MessagingOperationTypeProcess,
		semconv.MessagingDestinationName(message.Topic),
		semconv.MessagingKafkaOffset(int(message.Offset)),
		semconv.MessagingDestinationPartitionID(fmt.Sprintf("%d", message.Partition)),
		semconv.ServerAddress(strings.Join(p.config.KafkaService.Config().Brokers, ", ")),
	)
	err := p.config.KafkaService.CommitMessages(ctx, message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to commit message to kafka")
		return err
	}
	return nil
}
