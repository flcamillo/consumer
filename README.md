# Consumer - Event Processing Service

Uma aplicação Go robusta e observável para consumir e processar eventos de múltiplas fontes (AWS SQS e Kafka), com instrumentação completa de OpenTelemetry.

## 📋 Visão Geral

O **Consumer** é um serviço de processamento de eventos que:

- Consome mensagens de **AWS SQS** e **Kafka** simultaneamente
- Processa eventos usando um pool de **workers** configurável
- Fornece **observabilidade completa** através de OpenTelemetry (traces, métricas e logs estruturados)
- Suporta integração com **AWS Transfer Family** para orquestração de conectores
- Implementa **tratamento robusto de erros** e **graceful shutdown**

## 🚀 Tecnologias

- **Go 1.25.5**
- **AWS SDK v2** (SQS, DynamoDB, S3, Secrets Manager, Transfer)
- **Kafka** (via segmentio/kafka-go)
- **OpenTelemetry** (traces, métricas, logs distribuídos)
- **Grafana Stack** (LGTM - Loki, Grafana, Tempo, Mimir)

## 📦 Dependências Principais

```go
// AWS Services
github.com/aws/aws-sdk-go-v2/service/sqs
github.com/aws/aws-sdk-go-v2/service/dynamodb
github.com/aws/aws-sdk-go-v2/service/s3
github.com/aws/aws-sdk-go-v2/service/secretsmanager
github.com/aws/aws-sdk-go-v2/service/transfer

// Message Broker
github.com/segmentio/kafka-go

// Observability
go.opentelemetry.io/otel
go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc
```

## 🏗️ Arquitetura

```mermaid
graph TD
    SQS["AWS SQS<br/>Fila de Mensagens"]
    Kafka["Kafka<br/>Message Broker"]
    
    SQSConsumer["SQS Consumer<br/>Polling Contínuo"]
    KafkaConsumer["Kafka Consumer<br/>Consumo Contínuo"]
    
    MessageChan["Message Channel<br/>Canal de Distribuição"]
    
    W1["Worker 1"]
    W2["Worker 2"]
    W3["Worker 3"]
    WN["Worker N"]
    
    OTEL["OpenTelemetry<br/>Traces, Métricas, Logs"]
    Logger["Logger Estruturado<br/>log/slog"]
    
    SQS --> SQSConsumer
    Kafka --> KafkaConsumer
    SQSConsumer --> MessageChan
    KafkaConsumer --> MessageChan
    
    MessageChan --> W1
    MessageChan --> W2
    MessageChan --> W3
    MessageChan --> WN
    
    W1 --> OTEL
    W2 --> OTEL
    W3 --> OTEL
    WN --> OTEL
    
    W1 --> Logger
    W2 --> Logger
    W3 --> Logger
    WN --> Logger
    
    style SQS fill:#FF9900
    style Kafka fill:#000,color:#fff
    style OTEL fill:#4472C4
    style Logger fill:#70AD47
```

### Componentes

- **SQS Consumer**: Consome mensagens de fila AWS SQS com polling contínuo
- **Kafka Consumer**: Consome mensagens de tópico Kafka
- **Worker Pool**: Pool de tasks que processam mensagens em paralelo (escalável)
- **Message Channel**: Canal Go que distribui mensagens aos workers
- **OTEL SDK**: Coleta traces, métricas e logs estruturados
- **Logger**: Sistema de logs estruturado com contexto distribuído

## 🔧 Configuração

### Variáveis de Ambiente

| Variável | Descrição | Obrigatória | Exemplo |
|----------|-----------|------------|---------|
| `OTEL_SERVICE_NAME` | Nome do serviço para telemetria | ✅ | `consumer` |
| `SQS_QUEUE_URL` | URL da fila AWS SQS | ✅ | `https://sqs.{region}.amazonaws.com/{account}/queue-name` |
| `KAFKA_BROKER` | Endereço do broker Kafka | ✅ | `localhost:9092` |
| `KAFKA_TOPIC` | Tópico Kafka para consumir | ✅ | `topic1` |
| `KAFKA_CONSUMER_GROUP` | Grupo de consumo Kafka | ✅ | `consumer1` |
| `MAX_WORKERS` | Número de workers para processamento | ❌ | `10` (padrão) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Endpoint do collector OTLP | ❌ | `http://localhost:4317` |

### Credenciais AWS

A aplicação utiliza o **AWS SDK v2** que suporta múltiplas formas de autenticação:

```mermaid
graph TD
    A["AWS SDK v2"] --> B{Busca Credenciais}
    B --> C["1️⃣ Variáveis<br/>de Ambiente"]
    B --> D["2️⃣ AWS Credentials<br/>File ~/.aws/credentials"]
    B --> E["3️⃣ IAM Role<br/>Recomendado em Produção"]
    B --> F["4️⃣ AssumeRole<br/>com Temp Credentials"]
    
    C --> Found{"Encontrou?"}
    D --> Found
    E --> Found
    F --> Found
    
    Found -->|Sim| Success["✅ Autenticado"]
    Found -->|Não| Error["❌ Erro de Autenticação"]
    
    style Success fill:#70AD47,color:#fff
    style Error fill:#FF6B6B,color:#fff
```

Para usar credenciais temporárias da AWS:

```bash
# Obter credenciais temporárias
aws sts get-session-token --duration-seconds 3600

# Configurar variáveis de ambiente
set AWS_ACCESS_KEY_ID=<ACCESS_KEY>
set AWS_SECRET_ACCESS_KEY=<SECRET_KEY>
set AWS_SESSION_TOKEN=<TOKEN>
```

## 📚 Estrutura de Arquivos

```mermaid
graph TD
    ROOT["consumer/"]
    ROOT --> GOMOD["go.mod<br/>Definição do módulo"]
    ROOT --> GOSUM["go.sum<br/>Checksums"]
    ROOT --> MAIN["main.go<br/>Entrada & Inicialização"]
    ROOT --> WORKER["worker.go<br/>Pool de Workers"]
    ROOT --> SQS["sqs_consumer.go<br/>Consumer SQS"]
    ROOT --> KAFKA["kafka_consumer.go<br/>Consumer Kafka"]
    ROOT --> MSG["message_context.go<br/>Contexto de Mensagem"]
    ROOT --> ORCH["orchestrator.go<br/>Orquestrador"]
    ROOT --> OTEL["otel.go<br/>Configuração OpenTelemetry"]
    ROOT --> UTILS["utils.go<br/>Funções Utilitárias"]
    ROOT --> README["README.md<br/>Documentação"]
    ROOT --> GIT[".gitignore<br/>Git Ignore"]
    
    style ROOT fill:#4472C4,color:#fff
    style MAIN fill:#70AD47
    style GOMOD fill:#FF9900
    style OTEL fill:#4472C4
```

## 🚀 Como Executar

### Fluxo de Inicialização

```mermaid
graph TD
    Start["Iniciar Aplicação"] --> EnvVars["Carregar Variáveis<br/>de Ambiente"]
    EnvVars --> Validate{Validar<br/>Configurações}
    Validate -->|Erro| Error["❌ Erro de Configuração<br/>Encerrar"]
    Validate -->|OK| OTel["Setup OpenTelemetry<br/>Traces, Métricas, Logs"]
    OTel --> Runtime["Ativar Runtime Metrics<br/>Monitoramento de Memória"]
    Runtime --> SQSInit["Inicializar SQS Consumer"]
    Runtime --> KafkaInit["Inicializar Kafka Consumer"]
    SQSInit --> WorkerPool["Criar Worker Pool<br/>MAX_WORKERS instances"]
    KafkaInit --> WorkerPool
    WorkerPool --> Running["✅ Aplicação em Execução"]
    Running --> ProcessMessages["Processar Mensagens"]
    ProcessMessages --> Signals{"Receber Sinal?"}
    Signals -->|SIGTERM/SIGINT| Shutdown["Graceful Shutdown"]
    Signals -->|Não| ProcessMessages
    Shutdown --> Done["✅ Encerrado"]
    
    style Start fill:#4472C4,color:#fff
    style Running fill:#70AD47,color:#fff
    style Done fill:#70AD47,color:#fff
    style Error fill:#FF6B6B,color:#fff
    style Shutdown fill:#FFA500
```

- **Go 1.25.5+** instalado
- **Docker** (para services auxiliares)
- **Credenciais AWS** configuradas

### Build

```bash
# Clonar repositório
git clone https://github.com/flcamillo/consumer.git

# Compilar
go build -o consumer.exe

# Ou usando make (se disponível)
make build
```

### Executar Localmente

#### 1. Iniciar Stack de Observabilidade (Grafana LGTM)

```bash
docker run -d \
  -p 3000:3000 \
  -p 4317:4317 \
  -p 4318:4318 \
  --rm \
  --name lgtm \
  grafana/otel-lgtm:latest

# Acessar Grafana em: http://localhost:3000
```

#### 2. Iniciar Kafka

```bash
# Iniciar broker Kafka
docker run -d \
  -p 9092:9092 \
  --rm \
  --name broker \
  apache/kafka:latest

# Criar tópico
docker exec --workdir /opt/kafka/bin/ -it broker sh
./kafka-topics.sh --bootstrap-server localhost:9092 --create --topic topic1

# (Opcional) Enviar mensagem de teste
./kafka-console-producer.sh --bootstrap-server localhost:9092 --topic topic1
# Digite uma mensagem e pressione Enter
```

#### 3. Configurar Variáveis de Ambiente

```bash
# Windows (PowerShell)
$env:OTEL_SERVICE_NAME = "consumer"
$env:SQS_QUEUE_URL = "https://sqs.sa-east-1.amazonaws.com/{account-id}/queue-name"
$env:KAFKA_BROKER = "localhost:9092"
$env:KAFKA_TOPIC = "topic1"
$env:KAFKA_CONSUMER_GROUP = "consumer1"
$env:MAX_WORKERS = "5"
$env:OTEL_EXPORTER_OTLP_ENDPOINT = "http://localhost:4317"

# Windows (cmd)
set OTEL_SERVICE_NAME=consumer
set SQS_QUEUE_URL=https://sqs.sa-east-1.amazonaws.com/{account-id}/queue-name
set KAFKA_BROKER=localhost:9092
set KAFKA_TOPIC=topic1
set KAFKA_CONSUMER_GROUP=consumer1
set MAX_WORKERS=5
set OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
```

#### 4. Executar Consumer

```bash
.\consumer.exe

# Ou diretamente com Go
go run main.go kafka_consumer.go sqs_consumer.go worker.go orchestrator.go message_context.go otel.go utils.go
```

#### 5. Monitorar em Grafana

Acesse http://localhost:3000 com as credenciais padrão:
- **Usuário**: admin
- **Senha**: admin

## 📊 Observabilidade

### Fluxo de Telemetria

```mermaid
graph LR
    App["Consumer App"] 
    App -->|gRPC OTLP| Collector["OTEL Collector<br/>localhost:4317/4318"]
    
    Collector --> Tempo["Grafana Tempo<br/>Traces"]
    Collector --> Mimir["Grafana Mimir<br/>Métricas"]
    Collector --> Loki["Grafana Loki<br/>Logs"]
    
    Tempo --> Grafana["Grafana Dashboard<br/>localhost:3000"]
    Mimir --> Grafana
    Loki --> Grafana
    
    Grafana --> Visualization["📊 Visualização<br/>e Análise"]
    
    style App fill:#4472C4,color:#fff
    style Collector fill:#FF9900
    style Grafana fill:#FF6B6B,color:#fff
    style Visualization fill:#70AD47,color:#fff
```

### OpenTelemetry Exporters

O serviço exporta dados via **gRPC OTLP** para:

- **Traces**: Grafana Tempo
- **Métricas**: Grafana Mimir
- **Logs**: Grafana Loki

### Métricas Personalizadas

| Métrica | Tipo | Descrição | Unidade |
|---------|------|-----------|---------|
| `custom.sqsconsumer.messages.received` | Counter | Mensagens recebidas da SQS | messages |
| `custom.KafkaConsumer.messages.received` | Counter | Mensagens recebidas do Kafka | messages |
| `custom.worker.messages.processed` | Counter | Mensagens processadas com sucesso | messages |
| `custom.worker.messages.failed` | Counter | Mensagens que falharam | messages |
| `custom.sqsconsumer.messages.wait.duration` | Histogram | Tempo de espera (SQS) | s |
| `custom.KafkaConsumer.messages.wait.duration` | Histogram | Tempo de espera (Kafka) | s |
| `custom.worker.messages.wait.duration` | Histogram | Tempo de espera antes do processamento | s |

### Traces

Cada operação gera spans com informações de:
- Duração
- Status (sucesso/erro)
- Atributos contextuais
- Logs detalhados

### Logs Estruturados

Utiliza `log/slog` com integração OpenTelemetry para logs estruturados com contexto distribuído.

## ⚙️ Processamento de Mensagens

### Ciclo de Vida

```mermaid
sequenceDiagram
    participant Source as SQS/Kafka
    participant Consumer as Consumer
    participant Channel as Message Channel
    participant Worker
    participant Processing as Processamento
    participant Telemetry as OTEL

    Source->>Consumer: Enviar Mensagem
    Consumer->>Telemetry: Trace: Mensagem Recebida
    Consumer->>Channel: Enfileirar
    Channel->>Worker: Distribuir
    Worker->>Telemetry: Trace: Processamento Iniciado
    Worker->>Processing: Executar Lógica
    Processing-->>Worker: Resultado
    Worker->>Telemetry: Trace: Conclusão<br/>Métricas: Sucesso/Falha
    Worker->>Telemetry: Log: Detalhes
```

### Tratamento de Erros

- Tentativas automáticas com backoff
- Logs estruturados de erros com contexto
- Spans de telemetria marcam erros
- Graceful degradation em falhas não-críticas

## 🛑 Graceful Shutdown

A aplicação escuta sinais:
- `SIGTERM`: Encerramento ordenado
- `SIGINT`: Interrupção (Ctrl+C)

```mermaid
graph LR
    A["Recebe Sinal<br/>SIGTERM/SIGINT"] --> B["Para Aceitar<br/>Novas Mensagens"]
    B --> C["Aguarda Processamento<br/>em Andamento"]
    C --> D["Encerra Conexões<br/>SQS/Kafka"]
    D --> E["Faz Flush de Traces<br/>e Métricas"]
    E --> F["Encerra SDK<br/>OpenTelemetry"]
    F --> G["Exit"]
    
    style A fill:#FF6B6B
    style B fill:#FFA500
    style C fill:#FFD700
    style D fill:#87CEEB
    style E fill:#4472C4
    style F fill:#4472C4
    style G fill:#333,color:#fff
```

## 🔐 Segurança

- ✅ Credenciais via variáveis de ambiente (não hardcoded)
- ✅ Suporte a AWS IAM Roles
- ✅ Tokens de sessão temporários
- ✅ Contexto distribuído rastreável
- ✅ Logging estruturado para auditoria

## 🧪 Testes

### Teste Manual com Kafka

```bash
# Terminal 1: Iniciar produtor
docker exec -it broker /opt/kafka/bin/kafka-console-producer.sh \
  --bootstrap-server localhost:9092 \
  --topic topic1

# Digite mensagens para enviar

# Terminal 2: Monitorar logs/telemetria
# A aplicação consumer receberá e processará as mensagens
```

### Teste com SQS (Local)

Para testes locais com SQS, considere usar **LocalStack**:

```bash
docker run -d \
  -p 4566:4566 \
  --name localstack \
  localstack/localstack:latest

# Configurar SQS_QUEUE_URL para endpoint local
```

## 📈 Performance

- **Throughput**: Configurável via `MAX_WORKERS`
- **Latência**: Rastreada em spans de telemetria
- **Retenção**: Configurável por fonte (SQS/Kafka)

### Tuning

Para aumentar throughput:

```bash
set MAX_WORKERS=20
```

Para reduzir latência:
- Reduzir timeout de polling
- Aumentar batch size de pull
- Otimizar lógica de processamento

## 🐛 Troubleshooting

```mermaid
graph TD
    Issue["🚨 Problema?"]
    
    Issue -->|SQS não recebe| SQSCheck{"SQS_QUEUE_URL<br/>está correto?"}
    SQSCheck -->|Não| SQSFix["✏️ Ajustar variável"]
    SQSCheck -->|Sim| CheckAWS["Verificar credenciais AWS"]
    CheckAWS --> ComboFix["Atualizar variáveis<br/>AWS_*"]
    
    Issue -->|Kafka não conecta| KafkaCheck{"KAFKA_BROKER<br/>está acessível?"}
    KafkaCheck -->|Não| KafkaFix["✏️ Verificar Docker<br/>ou Rede"]
    KafkaCheck -->|Sim| BrokerFix["Verificar porta<br/>9092"]
    
    Issue -->|Sem Telemetria| TelemetryCheck{"OTEL_EXPORTER<br/>configurado?"}
    TelemetryCheck -->|Não| TelemetryFix1["✏️ Adicionar Variável"]
    TelemetryCheck -->|Sim| TelemetryFix2["Verificar Collector<br/>está rodando"]
    
    Issue -->|Memória crescendo| MemFix["Ajustar MAX_WORKERS<br/>ou revisar lógica"]
    
    SQSFix --> Done["✅ Resolvido"]
    ComboFix --> Done
    KafkaFix --> Done
    BrokerFix --> Done
    TelemetryFix1 --> Done
    TelemetryFix2 --> Done
    MemFix --> Done
    
    style Issue fill:#FF6B6B,color:#fff
    style Done fill:#70AD47,color:#fff
```

### Problemas Comuns

| Problema | Verificação | Solução |
|----------|-----------|---------|
| SQS falha | `SQS_QUEUE_URL` | Verificar URL e credenciais AWS |
| Kafka falha | `KAFKA_BROKER` acessível | Verificar Docker, porta 9092 |
| Sem Telemetria | `OTEL_EXPORTER_OTLP_ENDPOINT` | Configurar e iniciar Grafana LGTM |
| Memória crescendo | Métricas de runtime | Ajustar `MAX_WORKERS` |

## 📝 Logging

A aplicação usa `log/slog` com níveis:

- **INFO**: Eventos normais (start/stop)
- **WARN**: Condições inesperadas
- **ERROR**: Falhas na operação

Exemplo de log estruturado:
```
time=2024-03-24T10:30:45.123Z level=INFO msg="starting SqsConsumer..." service.name=consumer
time=2024-03-24T10:30:46.456Z level=INFO msg="message received" service.name=consumer trace_id=abc123 span_id=def456
```

## 🚀 Deployment

### Topologia de Deployment

```mermaid
graph TB
    subgraph Local["🏠 Local Development"]
        Docker1["Docker<br/>Grafana LGTM"]
        Docker2["Docker<br/>Kafka"]
        Consumer1["Consumer<br/>Executable"]
    end
    
    subgraph Prod["☁️ Produção"]
        K8s["Kubernetes Cluster"]
        K8s --> Replicas["3x Consumer Pods<br/>Load Balanced"]
        K8s --> ConfigMap["ConfigMap<br/>Variáveis Config"]
        K8s --> Secret["Secret<br/>AWS Credentials"]
        Replicas --> ConfigMap
        Replicas --> Secret
    end
    
    Docker1 -.->|Telemetria| Prod
    Docker2 -.->|Mensagens| Prod
    
    style Docker1 fill:#FF9900
    style Docker2 fill:#000,color:#fff
    style Consumer1 fill:#4472C4,color:#fff
    style K8s fill:#326CE5,color:#fff
    style Replicas fill:#70AD47,color:#fff
```

### Docker

Construir imagem:

```dockerfile
FROM golang:1.25.5 as builder
WORKDIR /app
COPY . .
RUN go build -o consumer

FROM debian:bookworm-slim
COPY --from=builder /app/consumer /usr/local/bin/
EXPOSE 4317 4318
CMD ["consumer"]
```

Executar container:

```bash
docker build -t consumer:latest .
docker run -d \
  -e OTEL_SERVICE_NAME=consumer \
  -e SQS_QUEUE_URL=https://sqs.region.amazonaws.com/account/queue \
  -e KAFKA_BROKER=kafka:9092 \
  consumer:latest
```

### Kubernetes

Deploy com 3 replicas:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: consumer
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: consumer
  template:
    metadata:
      labels:
        app: consumer
    spec:
      containers:
      - name: consumer
        image: consumer:latest
        imagePullPolicy: Always
        env:
        - name: OTEL_SERVICE_NAME
          value: "consumer"
        - name: MAX_WORKERS
          valueFrom:
            configMapKeyRef:
              name: consumer-config
              key: max-workers
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: access-key-id
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: secret-access-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          exec:
            command:
            - /bin/sh
            - -c
            - ps aux | grep consumer || exit 1
          initialDelaySeconds: 10
          periodSeconds: 10
```

## 📄 Licença

[Especificar licença do projeto]

## 👥 Contribuintes

[Lista de contribuintes]

## 📞 Suporte

Para questões e suporte, abra uma **issue** no repositório de origem.

---

**Última atualização**: Março de 2024
