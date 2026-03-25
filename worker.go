package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Define a configuração do Worker.
type WorkerConfig struct {
	// Canal para as menagens recebidas para processamento.
	MessageChan chan *MessageContext
}

// Define a estrutura para o processo responsável por processar as mensagens
// recebidas da fila AWS SQS.
type Worker struct {
	// Identificador do worker.
	id int
	// Configuração gerais.
	config *WorkerConfig
	// Tracer para criar spans de telemetria durante o processo.
	tracer trace.Tracer
	// metricas de requisições
	messageProcessed     metric.Int64Counter
	messageFailed        metric.Int64Counter
	messageWaitHistogram metric.Float64Histogram
}

// Construtor para criar uma nova instância do Worker
func NewWorker(id int, config *WorkerConfig) *Worker {
	worker := &Worker{
		id:     id,
		tracer: otel.Tracer("Worker"),
		config: config,
	}
	// configura as metricas
	meter := otel.Meter("Worker.metrics")
	if counter, err := meter.Int64Counter("custom.worker.messages.processed",
		metric.WithDescription("The number of messages processed sucessfully"),
		metric.WithUnit("{messages}")); err == nil {
		worker.messageProcessed = counter
	} else {
		panic(err)
	}
	if counter, err := meter.Int64Counter("custom.worker.messages.failed",
		metric.WithDescription("The number of messages processed with failue"),
		metric.WithUnit("{messages}")); err == nil {
		worker.messageFailed = counter
	} else {
		panic(err)
	}
	if histogram, err := meter.Float64Histogram("custom.worker.messages.wait.duration",
		metric.WithDescription("Message waiting time before being processed by worker."),
		metric.WithUnit("s")); err == nil {
		worker.messageWaitHistogram = histogram
	} else {
		panic(err)
	}
	return worker
}

// Método para iniciar o processo de consumo de mensagens da fila AWS SQS.
// O processo do worker só será encerrado caso o canal de mensagens seja
// encerrado. Isso garante que as mensagens em processamento sejam executadas.
func (p *Worker) Run(ctx context.Context) {
	slog.InfoContext(ctx, fmt.Sprintf("starting Worker {%d}...", p.id))
	defer slog.WarnContext(ctx, fmt.Sprintf("Worker stopped {%d}", p.id))
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "stopping Worker due to context cancellation...")
			return
		default:
			messageContext, ok := <-p.config.MessageChan
			if !ok {
				return
			}
			retry, err := p.processMessage(messageContext.Context, messageContext)
			if err != nil {
				slog.ErrorContext(messageContext.Context, "failed to process message",
					slog.String("message_id", messageContext.Id),
					slog.String("message_body", messageContext.Body),
					slog.Any("error", err),
				)
			}
			// deve atualizar as metricas e remover a mensagem caso a mesma tenha
			// sido processada com sucesso ou caso tenha ocorrido um erro que
			// não seja passível de retry
			if !retry {
				if err != nil {
					p.messageFailed.Add(messageContext.Context, 1)
				} else {
					p.messageProcessed.Add(messageContext.Context, 1)
				}
				p.messageWaitHistogram.Record(messageContext.Context, time.Since(messageContext.Received).Seconds())
				err = messageContext.Commit(messageContext.Context)
				if err != nil {
					slog.ErrorContext(messageContext.Context, "failed to commit message",
						slog.String("message_id", messageContext.Id),
						slog.String("message_body", messageContext.Body),
						slog.Any("error", err),
					)
				}
			}
		}
	}
}

// Método para identificar o tipo da mensagem recebida e processá-la de acordo.
// Retorna um booleano indicando se a mensagem deve ser reprocessada.
func (p *Worker) processMessage(ctx context.Context, messageContext *MessageContext) (retry bool, err error) {
	ctx, span := p.tracer.Start(ctx, "processMessage")
	defer span.End()
	slog.InfoContext(ctx, fmt.Sprintf("processing message id {%s} from {%s} data {%s}", messageContext.Id, messageContext.Source, messageContext.Body))
	time.Sleep(5 * time.Second)
	return false, err
}
