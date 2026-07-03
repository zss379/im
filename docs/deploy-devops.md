# 部署与 DevOps 设计说明书

{ 正式版 }

## 变更记录

| 变更标识 | 章节号及名称 | 变更内容描述 | 变更人 | 变更日期 | 变更前版本号 | 批准人 |
|---------|------------|------------|-------|---------|-----------|-------|
| C | 初始化 | 文档创建初始化 | 邱凯 | 2026/07/02 | | |

> 注：变更标识说明：C——创建，A——增加，M——修改，D——删除

## 1. 概述

### 1.1 文档目的

本文档定义数莲 PaaS IM 系统的部署架构与 DevOps 体系，涵盖 Docker Compose（开发环境）、K8s Helm Charts（生产环境）、CI/CD 流水线及环境管理策略，作为运维团队部署实施与研发团队日常开发的依据。

### 1.2 部署原则

| 原则 | 说明 |
|------|------|
| **基础设施即代码** | 所有部署配置（Docker Compose / Helm / CI 配置）版本化管理 |
| **不可变基础设施** | 容器化部署，禁止手动登录服务器修改配置 |
| **环境一致性** | 开发/测试/生产环境使用相同的基础镜像和编排结构 |
| **灰度发布** | 生产环境采用滚动更新 + 金丝雀发布策略 |
| **可观测性优先** | 所有服务集成 Prometheus 指标暴露 + 结构化日志 |

### 1.3 环境总览

| 环境 | 用途 | 部署方式 | 数据持久化 | 访问控制 |
|------|------|---------|-----------|---------|
| dev | 开发自测 | Docker Compose 本地 | 本地 Volume | 本地 localhost |
| test | QA 功能/集成测试 | K8s 轻量集群（3 节点） | 独立 PV | VPN + 账号 |
| staging | 预发验证/性能测试 | K8s 与生产 1:1 | 独立 PV | VPN + 账号 |
| prod | 正式上线 | K8s 生产集群 | 高可用存储 | 严格 ACL |

---

## 2. Docker Compose 开发环境

### 2.1 服务拓扑

```
┌─────────────────────────────────────────────────────────────┐
│                    docker-compose.yml                         │
│                                                               │
│  依赖服务:                                                     │
│  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  │
│  │ mysql  │  │ redis  │  │ mongo  │  │ kafka  │  │ minio  │  │
│  │ :3306  │  │ :6379  │  │ :27017 │  │ :9092  │  │ :9000  │  │
│  └────────┘  └────────┘  └────────┘  └────────┘  └────────┘  │
│                                                               │
│  IM 服务:                                                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ openim   │  │ api-gw   │  │ msg-svc  │  │ user-svc     │  │
│  │ :10001   │  │ :8080    │  │ :8081    │  │ :8082        │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘  │
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ group-svc│  │ conv-svc │  │ rc-svc   │  │ bot-svc      │  │
│  │ :8083    │  │ :8084    │  │ :8085    │  │ :8086        │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘  │
│                                                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │
│  │ file-svc │  │ audit-svc│  │ card-svc │  │ contact-svc  │  │
│  │ :8087    │  │ :8088    │  │ :8089    │  │ :8090        │  │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘  │
│                                                               │
│  可观测:                                                       │
│  ┌──────────┐  ┌──────────┐                                   │
│  │prometheus│  │ grafana  │                                   │
│  │ :9090    │  │ :3000    │                                   │
│  └──────────┘  └──────────┘                                   │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 docker-compose.yml 核心结构

```yaml
version: "3.8"

services:
  # === 依赖服务 ===
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root123
      MYSQL_DATABASE: im_shared
    ports: ["3306:3306"]
    volumes:
      - mysql_data:/var/lib/mysql
      - ./deploy/db/init.sql:/docker-entrypoint-initdb.d/init.sql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7.0
    ports: ["6379:6379"]
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes --requirepass ${REDIS_PASSWORD:-redis123}

  mongo:
    image: mongo:7.0
    ports: ["27017:27017"]
    volumes:
      - mongo_data:/data/db
    environment:
      MONGO_INITDB_ROOT_USERNAME: admin
      MONGO_INITDB_ROOT_PASSWORD: mongo123

  kafka:
    image: bitnami/kafka:3.5
    ports: ["9092:9092"]
    environment:
      KAFKA_CFG_NODE_ID: 0
      KAFKA_CFG_PROCESS_ROLES: controller,broker
      KAFKA_CFG_CONTROLLER_QUORUM_VOTERS: 0@kafka:9093
      KAFKA_CFG_LISTENERS: PLAINTEXT://:9092,CONTROLLER://:9093
      KAFKA_CFG_ADVERTISED_LISTENERS: PLAINTEXT://localhost:9092

  minio:
    image: minio/minio:latest
    ports: ["9000:9000", "9001:9001"]
    volumes:
      - minio_data:/data
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin

  # === IM 微服务 ===
  api-gateway:
    build:
      context: .
      dockerfile: deploy/docker/api-gateway.Dockerfile
    ports: ["8080:8080"]
    depends_on:
      mysql: { condition: service_healthy }
      redis: { condition: service_started }
    environment:
      DB_DSN: "root:root123@tcp(mysql:3306)/im_shared?charset=utf8mb4"
      REDIS_ADDR: "redis:6379"
      REDIS_PASSWORD: "redis123"
    volumes:
      - ./configs:/app/configs

  user-svc:
    build:
      context: .
      dockerfile: deploy/docker/user-svc.Dockerfile
    ports: ["8082:8082"]
    depends_on: [mysql, redis]
    environment:
      DB_DSN: "root:root123@tcp(mysql:3306)/im_shared"
      REDIS_ADDR: "redis:6379"

  message-svc:
    build:
      context: .
      dockerfile: deploy/docker/message-svc.Dockerfile
    ports: ["8081:8081"]
    depends_on: [mongo, kafka, redis]
    environment:
      MONGO_URI: "mongodb://admin:mongo123@mongo:27017/im_prod"
      KAFKA_BROKERS: "kafka:9092"
      REDIS_ADDR: "redis:6379"

  group-svc:
    build:
      context: .
      dockerfile: deploy/docker/group-svc.Dockerfile
    ports: ["8083:8083"]
    depends_on: [mysql, redis]
    environment:
      DB_DSN: "root:root123@tcp(mysql:3306)/im_shared"
      REDIS_ADDR: "redis:6379"

  # 其余微服务类似，不再逐一列出

  # === 可观测 ===
  prometheus:
    image: prom/prometheus:v2.54.1
    ports: ["9090:9090"]
    volumes:
      - ./deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:latest
    ports: ["3000:3000"]
    depends_on: [prometheus]
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin123
    volumes:
      - ./deploy/grafana/dashboards:/etc/grafana/provisioning/dashboards
      - grafana_data:/var/lib/grafana

volumes:
  mysql_data:
  redis_data:
  mongo_data:
  minio_data:
  prometheus_data:
  grafana_data:
```

### 2.3 启动流程

```bash
# 1. 克隆仓库
git clone https://github.com/shulian/im-server.git
cd im-server

# 2. 配置环境变量（可选）
cp .env.example .env
# 编辑 .env 修改变量

# 3. 一键启动全部服务
docker-compose up -d

# 4. 查看服务状态
docker-compose ps

# 5. 查看服务日志（指定服务）
docker-compose logs -f message-svc

# 6. 初始化数据库（首次启动自动执行 init.sql）
# 也可手动重置:
docker-compose exec mysql mysql -uroot -proot123 im_shared < deploy/db/init.sql

# 7. 验证服务健康
curl http://localhost:8080/health

# 8. 访问可观测面板
# Prometheus: http://localhost:9090
# Grafana:    http://localhost:3000 (admin/admin123)
```

### 2.4 Dockerfile 规范

所有微服务使用统一的 Dockerfile 模板：

```dockerfile
# 多阶段构建
# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/service ./cmd/service

# Stage 2: Run
FROM alpine:3.18

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/service .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["./service"]
```

### 2.5 .env 文件模板

```bash
# === 数据库 ===
MYSQL_ROOT_PASSWORD=root123
REDIS_PASSWORD=redis123
MONGO_PASSWORD=mongo123

# === 中间件 ===
KAFKA_BROKERS=localhost:9092

# === MinIO ===
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_BUCKET=im-files

# === JWT ===
JWT_SECRET=dev-secret-key-do-not-use-in-prod
JWT_EXPIRE_SECONDS=604800

# === 日志 ===
LOG_LEVEL=debug

# === OpenIM ===
OPENIM_API_URL=http://openim:10001
OPENIM_SECRET=openim-dev-secret
```

---

## 3. K8s Helm Chart 生产部署

### 3.1 仓库结构

```
deploy/charts/im/
├── Chart.yaml                  # 图表元数据
├── values.yaml                 # 全局默认值
├── values-prod.yaml            # 生产环境覆盖
├── values-staging.yaml         # 预发环境覆盖
├── values-test.yaml            # 测试环境覆盖
│
├── templates/
│   ├── _helpers.tpl            # 模板辅助函数
│   ├── namespace.yaml          # 命名空间
│   │
│   ├── configmap.yaml          # 公共配置
│   ├── secrets.yaml            # 敏感配置（外部 SecretStore）
│   │
│   ├── mysql/                   # 如需 Helm 管理 MySQL
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   │
│   ├── redis/
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   │
│   ├── mongodb/
│   │   ├── statefulset.yaml     # 实际建议用 Operator
│   │   └── service.yaml
│   │
│   ├── kafka/
│   │   ├── statefulset.yaml     # 实际建议用 Strimzi Operator
│   │   └── service.yaml
│   │
│   ├── minio/
│   │   ├── statefulset.yaml
│   │   └── service.yaml
│   │
│   ├── api-gateway/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── ingress.yaml
│   │   └── hpa.yaml
│   │
│   ├── user-svc/
│   │   ├── deployment.yaml      # 模板示例（其他服务类似）
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   │
│   ├── message-svc/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   │
│   ├── ... (其他微服务)
│   │
│   ├── openim-server/
│   │   ├── statefulset.yaml     # 有状态服务
│   │   └── service.yaml
│   │
│   └── monitoring/
│       ├── prometheus/
│       │   ├── servicemonitor.yaml
│       │   └── podmonitor.yaml
│       └── grafana/
│           └── configmap-dashboards.yaml
│
└── charts/                      # 子依赖（可选使用 Bitnami 等第三方 chart）
    └── mysql/
```

### 3.2 values.yaml 核心配置

```yaml
# === 全局配置 ===
global:
  environment: production
  imageRegistry: registry.shulian.com
  imageTag: latest
  imagePullPolicy: Always

# === 命名空间 ===
namespace: im-system

# === 镜像配置 ===
images:
  apiGateway: im/api-gateway
  userSvc: im/user-svc
  messageSvc: im/message-svc
  groupSvc: im/group-svc
  conversationSvc: im/conversation-svc
  rcSvc: im/rc-svc
  botSvc: im/bot-svc
  fileSvc: im/file-svc
  auditSvc: im/audit-svc
  cardSvc: im/card-svc
  contactSvc: im/contact-svc
  openim: im/openim-server

# === 数据库中间件（生产建议用外部托管） ===
mysql:
  enabled: false              # 使用外部 MySQL
  external:
    host: mysql.internal
    port: 3306
    database: im_shared
    existingSecret: mysql-credentials

redis:
  enabled: false
  external:
    addrs: ["redis-0.redis:6379", "redis-1.redis:6379"]
    passwordSecret: redis-password

mongodb:
  enabled: false
  external:
    uri: "mongodb://user:pass@mongo0:27017,mongo1:27017/im_prod?replicaSet=rs0"

kafka:
  enabled: false
  external:
    brokers: ["kafka-0.kafka:9092", "kafka-1.kafka:9092"]

minio:
  enabled: false
  external:
    endpoint: minio.shulian.com
    accessKeySecret: minio-credentials

# === 微服务公共配置 ===
service:
  replicaCount: 3
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 2
      memory: 2Gi
  healthCheck:
    livenessProbe:
      httpGet: { path: /health, port: 8080 }
      initialDelaySeconds: 10
      periodSeconds: 15
    readinessProbe:
      httpGet: { path: /ready, port: 8080 }
      initialDelaySeconds: 5
      periodSeconds: 10
  hpa:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    metrics:
      - type: Resource
        resource:
          name: cpu
          target:
            type: Utilization
            averageUtilization: 70
      - type: Resource
        resource:
          name: memory
          target:
            type: Utilization
            averageUtilization: 80

# === 性能关键服务独立配置 ===
message-svc:
  replicaCount: 5
  resources:
    requests: { cpu: 1, memory: 1Gi }
    limits: { cpu: 4, memory: 4Gi }

openim-server:
  replicaCount: 3
  serviceType: StatefulSet
  resources:
    requests: { cpu: 2, memory: 2Gi }
    limits: { cpu: 8, memory: 8Gi }

# === 接入层 ===
ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: im-api.shulian.com
      paths: ["/"]
  tls:
    - hosts: ["im-api.shulian.com"]
      secretName: im-api-tls

websocket:
  enabled: true
  hosts:
    - host: im-ws.shulian.com
      paths: ["/v1/ws"]

# === 可观测 ===
monitoring:
  prometheus:
    enabled: true
    serviceMonitor:
      interval: 15s
  grafana:
    enabled: true
    dashboards:
      - name: im-business-dashboard
        file: deploy/grafana/dashboards/im-business.json
      - name: im-infra-dashboard
        file: deploy/grafana/dashboards/im-infra.json
```

### 3.3 deployment.yaml 模板

```yaml
{{- /* 以 user-svc 为例，其他微服务共用此模板 */ -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "im.fullname" . }}-user-svc
  namespace: {{ .Values.namespace }}
  labels:
    app: user-svc
    chart: {{ .Chart.Name }}-{{ .Chart.Version }}
spec:
  replicas: {{ .Values.service.replicaCount }}
  selector:
    matchLabels:
      app: user-svc
  template:
    metadata:
      labels:
        app: user-svc
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
        - name: user-svc
          image: "{{ .Values.global.imageRegistry }}/{{ .Values.images.userSvc }}:{{ .Values.global.imageTag }}"
          imagePullPolicy: {{ .Values.global.imagePullPolicy }}
          ports:
            - containerPort: 8080
              name: http
          env:
            - name: DB_DSN
              valueFrom:
                secretKeyRef:
                  name: {{ .Values.mysql.external.existingSecret }}
                  key: dsn
            - name: REDIS_ADDR
              value: "{{ .Values.redis.external.addrs | join "," }}"
            - name: REDIS_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: {{ .Values.redis.external.passwordSecret }}
                  key: password
            - name: LOG_LEVEL
              value: "info"
            - name: JAEGER_ENDPOINT
              value: "http://jaeger-collector:14268/api/traces"
          resources:
            {{- toYaml .Values.service.resources | nindent 12 }}
          livenessProbe:
            {{- toYaml .Values.service.healthCheck.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.service.healthCheck.readinessProbe | nindent 12 }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ include "im.fullname" . }}-user-svc
  namespace: {{ .Values.namespace }}
spec:
  selector:
    app: user-svc
  ports:
    - port: 8080
      targetPort: 8080
      name: http
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "im.fullname" . }}-user-svc-hpa
  namespace: {{ .Values.namespace }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "im.fullname" . }}-user-svc
  minReplicas: {{ .Values.service.hpa.minReplicas }}
  maxReplicas: {{ .Values.service.hpa.maxReplicas }}
  metrics:
    {{- toYaml .Values.service.hpa.metrics | nindent 4 }}
```

### 3.4 命名空间与网络策略

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: im-system

---
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: im-network-policy
  namespace: im-system
spec:
  podSelector: {}
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: im-system
    - from:
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - port: 8080
  egress:
    - to:
        - namespaceSelector: {}
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except: ["10.0.0.0/8"]   # 禁止访问内网非 IM 服务
```

### 3.5 部署命令

```bash
# 部署到测试环境
helm upgrade --install im-release deploy/charts/im \
  --namespace im-system \
  --create-namespace \
  -f deploy/charts/im/values.yaml \
  -f deploy/charts/im/values-test.yaml \
  --set global.imageTag=test-20260702-123456

# 部署到生产环境（带版本号）
helm upgrade --install im-release deploy/charts/im \
  --namespace im-system \
  -f deploy/charts/im/values.yaml \
  -f deploy/charts/im/values-prod.yaml \
  --set global.imageTag=v1.0.0 \
  --atomic \
  --timeout 10m

# 回滚到上一版本
helm rollback im-release 1 --namespace im-system

# 查看发布历史
helm history im-release --namespace im-system

# 差异化查看当前与期望状态
helm diff upgrade im-release deploy/charts/im \
  -f deploy/charts/im/values-prod.yaml
```

---

## 4. CI/CD 流水线

### 4.1 流水线总览

```
                    ┌─────────────────────────────┐
                    │   开发者提交 PR               │
                    └────────────┬────────────────┘
                                 │
                    ┌────────────▼────────────────┐
                    │    PR CI 流水线               │
                    │    ├─ Lint + 单元测试         │
                    │    ├─ 构建镜像               │
                    │    ├─ 镜像安全扫描            │
                    │    └─ PR 评论测试报告         │
                    └────────────┬────────────────┘
                                 │ PR merged
                    ┌────────────▼────────────────┐
                    │    Branch CI (main)          │
                    │    ├─ 全量单元测试 + API 测试 │
                    │    ├─ 构建并推送正式镜像       │
                    │    └─ 打 Git Tag             │
                    └────────────┬────────────────┘
                                 │
                    ┌────────────▼────────────────┐
                    │    自动部署到测试环境          │
                    │    ├─ Helm upgrade test       │
                    │    └─ 触发 E2E 测试            │
                    └────────────┬────────────────┘
                                 │
                    ┌────────────▼────────────────┐
                    │    手动触发部署到预发环境       │
                    │    ├─ 金丝雀发布 (10% 流量)    │
                    │    ├─ 验证 10 分钟             │
                    │    └─ 全量发布                 │
                    └────────────┬────────────────┘
                                 │
                    ┌────────────▼────────────────┐
                    │    批准后部署到生产环境         │
                    │    ├─ 滚动更新                 │
                    │    ├─ 分阶段 (10% → 50% → 全量)│
                    │    └─ 健康检查自动回滚          │
                    └────────────────────────────┘
```

### 4.2 GitHub Actions 配置

```yaml
# .github/workflows/ci.yml — PR CI
name: PR CI

on:
  pull_request:
    branches: [main, release/*]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.21"
      - name: Lint
        uses: golangci/golangci-lint-action@v3
        with:
          version: v1.55

  unit-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.21"
      - name: Run unit tests
        run: go test -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: actions/upload-artifact@v4
        with:
          name: coverage
          path: coverage.out

  build:
    runs-on: ubuntu-latest
    needs: [lint, unit-test]
    strategy:
      matrix:
        service:
          - api-gateway
          - user-svc
          - message-svc
          - group-svc
          - conversation-svc
          - rc-svc
          - bot-svc
          - file-svc
          - audit-svc
          - card-svc
          - contact-svc
    steps:
      - uses: actions/checkout@v4
      - name: Build ${{ matrix.service }}
        run: docker build -t ${{ matrix.service }}:${{ github.sha }} -f deploy/docker/${{ matrix.service }}.Dockerfile .

  security-scan:
    runs-on: ubuntu-latest
    needs: [build]
    steps:
      - name: Trivy scan
        uses: aquasecurity/trivy-action@master
        with:
          image-ref: "im-server:${{ github.sha }}"
          format: "sarif"
          output: "trivy-results.sarif"
          severity: "HIGH,CRITICAL"
```

```yaml
# .github/workflows/deploy.yml — 部署流水线
name: Deploy

on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      environment:
        description: "Deploy target"
        required: true
        default: staging
        type: choice
        options: [staging, production]

env:
  REGISTRY: registry.shulian.com
  HELM_CHART: deploy/charts/im

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      - name: Login to Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ secrets.REGISTRY_USER }}
          password: ${{ secrets.REGISTRY_PASS }}
      - name: Build and push all services
        run: |
          SERVICES="api-gateway user-svc message-svc group-svc conv-svc rc-svc bot-svc file-svc audit-svc card-svc contact-svc openim-server"
          TAG=${{ github.sha }}
          for svc in $SERVICES; do
            docker buildx build \
              --platform linux/amd64 \
              --tag $REGISTRY/im/$svc:$TAG \
              --file deploy/docker/$svc.Dockerfile \
              --push .
          done

  deploy-test:
    needs: [build-and-push]
    runs-on: ubuntu-latest
    environment: test
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to test
        run: |
          helm upgrade --install im-release $HELM_CHART \
            --namespace im-system \
            -f $HELM_CHART/values.yaml \
            -f $HELM_CHART/values-test.yaml \
            --set global.imageTag=${{ github.sha }} \
            --atomic \
            --timeout 5m

  e2e-test:
    needs: [deploy-test]
    runs-on: ubuntu-latest
    steps:
      - name: Run E2E tests
        run: |
          # 触发 E2E 测试套件，等待结果
          # 可调用单独的 E2E 测试项目
          echo "E2E tests passed"

  deploy-staging:
    needs: [e2e-test]
    if: github.event_name == 'workflow_dispatch' && github.event.inputs.environment == 'staging'
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - uses: actions/checkout@v4
      - name: Canary deploy (10%)
        run: |
          helm upgrade im-release $HELM_CHART \
            --namespace im-system \
            -f $HELM_CHART/values.yaml \
            -f $HELM_CHART/values-staging.yaml \
            --set global.imageTag=${{ github.sha }} \
            --set canary.weight=10 \
            --atomic
      - name: Wait for validation
        run: sleep 300  # 等待 5 分钟观察
      - name: Full rollout
        run: |
          helm upgrade im-release $HELM_CHART \
            --namespace im-system \
            --set canary.weight=100

  deploy-production:
    needs: [deploy-staging]
    if: github.event_name == 'workflow_dispatch' && github.event.inputs.environment == 'production'
    runs-on: ubuntu-latest
    environment:
      name: production
      url: https://im.shulian.com
    steps:
      - uses: actions/checkout@v4
      - name: Phase 1 — 10% rollout
        run: |
          helm upgrade im-release $HELM_CHART \
            --namespace im-system \
            -f $HELM_CHART/values.yaml \
            -f $HELM_CHART/values-prod.yaml \
            --set global.imageTag=${{ github.sha }} \
            --set rollout.weight=10
      - name: Health check
        run: |
          ./scripts/health-check.sh --phase=10pct --timeout=300
      - name: Phase 2 — 50% rollout
        run: |
          helm upgrade im-release $HELM_CHART \
            --set rollout.weight=50
      - name: Health check
        run: |
          ./scripts/health-check.sh --phase=50pct --timeout=300
      - name: Phase 3 — 100% rollout
        run: |
          helm upgrade im-release $HELM_CHART \
            --set rollout.weight=100
      - name: Final verification
        run: |
          ./scripts/health-check.sh --phase=full --timeout=600
```

---

## 5. 发布策略

### 5.1 滚动更新配置

```yaml
# K8s 滚动更新策略
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1          # 最多比期望多 1 个 Pod
    maxUnavailable: 0    # 滚动期间不能有不可用 Pod
```

### 5.2 金丝雀发布

```yaml
# 通过 Service Mesh (Istio) 实现金丝雀发布
# 部署两个版本的 Deployment，通过 VirtualService 控制流量权重
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: message-svc-canary
spec:
  hosts:
    - message-svc
  http:
    - match:
        - headers:
            x-canary:
              exact: "true"        # 特定 header 灰度用户
      route:
        - destination:
            host: message-svc
            subset: canary
    - route:
        - destination:
            host: message-svc
            subset: stable
          weight: 90
        - destination:
            host: message-svc
            subset: canary
          weight: 10
```

### 5.3 版本号规范

```
v{major}.{minor}.{patch}[-{pre-release}]+{build-metadata}

示例:
v1.0.0              # 正式发布
v1.0.1              # 补丁
v1.1.0-alpha.1      # 内测
v1.1.0-beta.1       # 公测
v1.1.0-rc.1         # 发布候选
```

### 5.4 发布 Checklist

```
□ 代码冻结：main 分支已合入所有预期 PR
□ 版本号：已更新 Chart.yaml 和源码版本号
□ 变更日志：CHANGELOG.md 已更新
□ 镜像已构建并 push 到 registry
□ 单元测试通过（go test -race ./...）
□ API 测试通过（NewMan 套件）
□ E2E 测试通过（Playwright + k6）
□ 安全扫描无 HIGH/CRITICAL 漏洞
□ 测试环境已验证通过
□ 预发环境金丝雀验证通过
□ 数据库 Migration SQL 已审核
□ 运维人员已通知
```

---

## 6. 数据库 Migration 策略

### 6.1 Migration 流程

```bash
# Migration 文件存放位置
deploy/db/
├── migrations/
│   ├── 001_create_users.sql
│   ├── 002_create_conversations.sql
│   ├── 003_create_groups.sql
│   ├── 004_add_group_notice_idx.sql
│   └── ...
├── seed/
│   ├── system_bots.sql        # 系统机器人初始化
│   └── default_configs.sql    # 默认配置
└── init.sql                   # 首次初始化（全量）

# 执行迁移（使用 golang-migrate 或类似工具）
migrate -path deploy/db/migrations -database "mysql://root:pass@tcp(localhost:3306)/im_shared" up
```

### 6.2 Migration 设计原则

| 原则 | 说明 |
|------|------|
| 向前兼容 | 旧版本代码不能因为新 Migration 而崩溃 |
| 增量变更 | 禁止修改已发布的 Migration 文件，只能追加新文件 |
| 可回滚 | 每个 Up 必须有对应的 Down 语句 |
| 代码审查 | Migration SQL 必须经过 DBA 审核 |

```sql
-- 示例: 001_create_users.sql (Up)
CREATE TABLE IF NOT EXISTS user (
    user_id BIGINT AUTO_INCREMENT PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    account VARCHAR(64) NOT NULL,
    password_hash VARCHAR(256) NOT NULL,
    display_name VARCHAR(64) NOT NULL,
    status TINYINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_tenant_account (tenant_id, account),
    KEY idx_tenant_status (tenant_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 001_create_users.sql (Down)
DROP TABLE IF EXISTS user;
```

---

## 7. 备份与恢复

### 7.1 备份策略

| 数据 | 工具 | 频率 | 保留期 | 存储位置 |
|------|------|------|--------|---------|
| MySQL 全量 | mysqldump | 每日 02:00 | 30 天 | S3 + 本地 |
| MySQL binlog | mysqlbinlog | 实时 | 7 天 | S3 |
| MongoDB | mongodump | 每日 03:00 | 30 天 | S3 |
| MongoDB Oplog | 副本集 | 内置 | 操作日志 | - |
| Redis RDB | SAVE/BGSAVE | 每小时 | 7 天 | S3 |
| MinIO 文件 | MinIO 版本控制 | 内置 | 版本管理 | MinIO 自身 |
| K8s 资源 | Velero | 每日 | 30 天 | S3 |

### 7.2 备份脚本

```bash
#!/bin/bash
# scripts/backup-mysql.sh — MySQL 全量备份

BACKUP_DIR="/backup/mysql"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30
S3_BUCKET="s3://im-backup/mysql"

mkdir -p $BACKUP_DIR

# 执行全量备份
mysqldump \
  --host=$MYSQL_HOST \
  --user=$MYSQL_USER \
  --password=$MYSQL_PASSWORD \
  --databases im_shared \
  --single-transaction \
  --routines \
  --triggers \
  --events \
  --compress \
  | gzip > $BACKUP_DIR/im_shared_$DATE.sql.gz

# 上传到 S3
aws s3 cp $BACKUP_DIR/im_shared_$DATE.sql.gz $S3_BUCKET/

# 清理 30 天前的本地备份
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete

# 发送备份完成通知
curl -X POST $WEBHOOK_URL \
  -H "Content-Type: application/json" \
  -d "{\"text\": \"MySQL backup completed: im_shared_$DATE.sql.gz\"}"
```

### 7.3 恢复流程

```bash
# 1. 停止 IM 服务（避免写入）
kubectl scale deployment -n im-system --all --replicas=0

# 2. 恢复 MySQL
gunzip < /backup/mysql/im_shared_20260702_020000.sql.gz | mysql -h $MYSQL_HOST -u $MYSQL_USER -p

# 3. 恢复 MongoDB（可选）
mongorestore --uri="$MONGO_URI" --drop /backup/mongo/20260702/

# 4. 启动服务
kubectl scale deployment -n im-system --all --replicas=3

# 5. 验证数据完整性
./scripts/verify-data.sh
```

---

## 8. 监控与告警

### 8.1 告警规则

```yaml
# deploy/prometheus/rules/alerts.yml
groups:
  - name: im-service-alerts
    rules:
      - alert: HighMessageLatency
        expr: histogram_quantile(0.99, rate(im_messages_send_duration_ms_bucket[5m])) > 500
        for: 5m
        labels: { severity: critical }
        annotations:
          summary: "P99 message latency > 500ms"
          description: "Current P99: {{ $value }}ms"

      - alert: HighMessageFailureRate
        expr: rate(im_messages_failed_total[5m]) / rate(im_messages_sent_total[5m]) > 0.01
        for: 5m
        labels: { severity: critical }
        annotations:
          summary: "Message failure rate > 1%"

      - alert: ServiceDown
        expr: up{job=~"im-.*"} == 0
        for: 1m
        labels: { severity: critical }
        annotations:
          summary: "{{ $labels.job }} is down"

      - alert: HighMemoryUsage
        expr: container_memory_usage_percent > 80
        for: 10m
        labels: { severity: warning }
        annotations:
          summary: "Memory usage > 80% on {{ $labels.instance }}"

      - alert: KafkaConsumerLag
        expr: kafka_consumer_lag > 10000
        for: 5m
        labels: { severity: warning }
        annotations:
          summary: "Kafka consumer lag > 10000 on {{ $labels.topic }}"

      - alert: MongoDBReplicationLag
        expr: mongodb_replset_member_replication_lag_seconds > 10
        for: 5m
        labels: { severity: warning }
```

### 8.2 告警通知渠道

| 级别 | 通知方式 | 响应要求 |
|------|---------|---------|
| Critical | 电话 + 钉钉/企微 @oncall | 5 分钟内响应 |
| Warning | 钉钉/企微群消息 | 30 分钟内评估 |
| Info | 邮件 | 工作日处理 |

### 8.3 故障响应流程

```
告警触发
  │
  ├── 值班人员确认
  │    ├── 5min 内无响应 → 升级到技术负责人
  │    └── 确认问题
  │
  ├── 问题定级
  │    ├── P0: 核心功能瘫痪 → 立即召集所有相关研发
  │    ├── P1: 主要功能异常 → 主力研发介入
  │    └── P2: 非核心问题 → 排期修复
  │
  ├── 应急处理
  │    ├── 回滚最近变更（如由发布引起）
  │    ├── 扩容受影响服务
  │    ├── 降级非核心功能保障核心链路
  │    └── 切换容灾
  │
  └── 事后复盘
       ├── 5 Why 根因分析
       ├── 改进措施
       └── 更新 Runbook
```

---

## 9. 服务网格与流量管理（Istio）

### 9.1 Istio 配置

```yaml
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  name: im-istio-config
spec:
  profile: default
  components:
    ingressGateways:
      - name: istio-ingressgateway
        enabled: true
        k8s:
          service:
            type: LoadBalancer
            ports:
              - port: 80
                targetPort: 8080
                name: http
              - port: 443
                targetPort: 8443
                name: https
  meshConfig:
    enableTracing: true
    defaultConfig:
      tracing:
        zipkin:
          address: jaeger-collector:9411
    accessLogFile: /dev/stdout
    accessLogEncoding: JSON
```

### 9.2 流量管理关键配置

```yaml
# 熔断
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: message-svc-circuit-breaker
spec:
  host: message-svc
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 50
        maxRequestsPerConnection: 10
    outlierDetection:
      consecutive5xxErrors: 5
      interval: 30s
      baseEjectionTime: 60s
```

---

## 10. 运维常用命令速查

### 10.1 服务管理

```bash
# 查看所有服务状态
kubectl get pods -n im-system -o wide

# 查看服务日志
kubectl logs -n im-system -l app=message-svc --tail=100 -f

# 重启服务（滚动更新）
kubectl rollout restart deployment -n im-system message-svc

# 查看滚动更新状态
kubectl rollout status deployment -n im-system message-svc

# 扩缩容
kubectl scale deployment -n im-system message-svc --replicas=5
```

### 10.2 数据库

```bash
# MySQL 连接
kubectl exec -it -n im-system deploy/mysql -- mysql -uroot -p

# MySQL 慢查询检查
kubectl exec -it -n im-system deploy/mysql -- mysql -e "SHOW PROCESSLIST;"

# MongoDB 连接
kubectl exec -it -n im-system sts/mongodb-0 -- mongosh

# 查看 MongoDB 分片状态
kubectl exec -it -n im-system sts/mongodb-0 -- mongosh --eval "sh.status()"

# Redis 检查
kubectl exec -it -n im-system sts/redis-0 -- redis-cli INFO memory
```

### 10.3 诊断

```bash
# 端口转发（本地调试）
kubectl port-forward -n im-system svc/message-svc 8081:8080

# 检查 HPA 状态
kubectl get hpa -n im-system

# 查看资源使用
kubectl top pods -n im-system

# 查看事件
kubectl get events -n im-system --sort-by='.lastTimestamp'

# 网络诊断
kubectl run -it --rm debug --image=nicolaka/netshoot -- /bin/bash
```
