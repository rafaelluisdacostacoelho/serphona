# Estimativa de Custos - Infraestrutura Serphona

## 📋 Sumário Executivo

Este documento apresenta uma estimativa detalhada de custos de infraestrutura para a plataforma Serphona em dois cenários:
- **Cenário 1**: 100 clientes, 5 agentes de atendimento 24/7
- **Cenário 2**: 1000 clientes, 5 agentes de atendimento 24/7

## 🎯 Premissas

### Cálculo de Chamadas Concorrentes
- **5 agentes atendendo 24/7**
- Tempo médio de atendimento: 5 minutos
- Capacidade por agente: 12 atendimentos/hora
- Total: 60 atendimentos/hora (5 agentes × 12)
- **Pico de chamadas concorrentes estimado: 15-20 chamadas simultâneas**

### Cálculo de Recursos por Cliente
- Média de 10-15 usuários ativos por cliente
- Chamadas por cliente/mês: ~1.000
- Armazenamento de áudio: 500MB/cliente/mês
- Dados analíticos: 200MB/cliente/mês

---

## 💰 CENÁRIO 1: 100 Clientes + 5 Agentes 24/7

### 📊 Dimensionamento de Infraestrutura

#### 1. Kubernetes Cluster
| Componente | Especificação | Quantidade | Justificativa |
|------------|---------------|------------|---------------|
| Control Plane | 4 vCPU, 8GB RAM | 3 nós | Alta disponibilidade |
| Worker Nodes | 16 vCPU, 32GB RAM | 5 nós | Carga de aplicação |
| **Total K8s** | **92 vCPU, 184GB RAM** | **8 nós** | - |

#### 2. Databases & Storage

**PostgreSQL (Patroni HA)**
- 3 nós: 8 vCPU, 64GB RAM, 500GB SSD cada
- 2 PgBouncer: 2 vCPU, 4GB RAM cada
- 2 HAProxy: 2 vCPU, 4GB RAM cada
- **Total**: 32 vCPU, 204GB RAM, 1.5TB storage

**ClickHouse (Analytics)**
- 3 nós: 8 vCPU, 32GB RAM, 1TB SSD cada
- **Total**: 24 vCPU, 96GB RAM, 3TB storage

**Redis (Cache/Sessions)**
- 3 nós Sentinel: 4 vCPU, 16GB RAM, 100GB SSD cada
- **Total**: 12 vCPU, 48GB RAM, 300GB storage

**MinIO (Object Storage)**
- 4 nós: 4 vCPU, 8GB RAM, 2TB HDD cada
- **Total**: 16 vCPU, 32GB RAM, 8TB storage

#### 3. Message Queue & Events

**Kafka Cluster**
- 3 brokers: 8 vCPU, 16GB RAM, 500GB SSD cada
- 3 Zookeeper: 2 vCPU, 4GB RAM, 50GB SSD cada
- **Total**: 30 vCPU, 60GB RAM, 1.65TB storage

#### 4. VoIP Infrastructure

**Asterisk (PBX)**
- 2 instâncias: 8 vCPU, 16GB RAM, 200GB SSD cada
- Capacidade: ~50 chamadas simultâneas por instância
- **Total**: 16 vCPU, 32GB RAM, 400GB storage

**RTPEngine (Media Processing)**
- 2 instâncias: 4 vCPU, 8GB RAM, 50GB SSD cada
- **Total**: 8 vCPU, 16GB RAM, 100GB storage

**Kamailio (SIP Proxy)**
- 2 instâncias: 4 vCPU, 8GB RAM, 50GB SSD cada
- **Total**: 8 vCPU, 16GB RAM, 100GB storage

#### 5. Observability Stack

**Monitoring & Logging**
- Prometheus: 4 vCPU, 16GB RAM, 500GB SSD
- Grafana: 2 vCPU, 4GB RAM, 50GB SSD
- Loki: 4 vCPU, 8GB RAM, 1TB SSD
- Tempo: 4 vCPU, 8GB RAM, 500GB SSD
- **Total**: 14 vCPU, 36GB RAM, 2.05TB storage

#### 6. Security & Management

**HashiCorp Vault**
- 3 nós: 2 vCPU, 4GB RAM, 50GB SSD cada
- **Total**: 6 vCPU, 12GB RAM, 150GB storage

**Bastion & Jump Hosts**
- 2 instâncias: 2 vCPU, 4GB RAM, 50GB SSD cada
- **Total**: 4 vCPU, 8GB RAM, 100GB storage

---

### 💵 Estimativa de Custos - 100 Clientes

#### Opção A: Infraestrutura On-Premise

| Categoria | Especificação | Investimento Inicial | Custo Mensal |
|-----------|---------------|---------------------|--------------|
| **Servidores Físicos** | 10× Dell R750 (2× Xeon Gold, 512GB RAM, 8TB SSD) | R$ 500.000 | R$ 5.000 (manutenção) |
| **Switches de Rede** | 2× Switch 48 portas 10Gbps | R$ 60.000 | R$ 500 |
| **Firewall** | 2× Firewall HA Fortinet | R$ 80.000 | R$ 800 |
| **Storage NAS** | 1× Storage NAS 50TB (backup) | R$ 100.000 | R$ 1.000 |
| **UPS/Nobreak** | 2× UPS 10kVA redundantes | R$ 60.000 | R$ 600 |
| **Rack & PDU** | 1× Rack 42U + PDUs | R$ 20.000 | R$ 200 |
| **Internet/Link** | Link dedicado 1Gbps redundante | - | R$ 8.000 |
| **Energia Elétrica** | ~20kW consumo médio | - | R$ 12.000 |
| **Datacenter/Colocation** | Espaço físico + refrigeração | - | R$ 15.000 |
| **Equipe Operacional** | 2× SysAdmin + 1× DevOps | - | R$ 45.000 |
| **Licenças de Software** | SO, Monitoring, Backup | R$ 50.000 | R$ 5.000 |
| **SUBTOTAL** | | **R$ 870.000** | **R$ 93.100/mês** |
| **Depreciação (36 meses)** | | | **R$ 24.167/mês** |
| **TOTAL MENSAL** | | | **≈ R$ 117.267/mês** |

**Custo por Cliente**: R$ 1.172,67/mês

#### Opção B: Cloud AWS

| Categoria | Recursos | Quantidade | Custo Unit. (USD) | Custo Total (USD) | Custo Total (BRL)* |
|-----------|----------|------------|-------------------|-------------------|-------------------|
| **EKS Control Plane** | Cluster gerenciado | 1 | $144 | $144 | R$ 720 |
| **EC2 - K8s Workers** | m6i.4xlarge (16vCPU, 64GB) | 5 | $495 | $2.475 | R$ 12.375 |
| **RDS Aurora PostgreSQL** | db.r6g.2xlarge (8vCPU, 64GB) | 3 | $825 | $2.475 | R$ 12.375 |
| **EC2 - ClickHouse** | r6i.2xlarge (8vCPU, 32GB) | 3 | $413 | $1.239 | R$ 6.195 |
| **ElastiCache Redis** | cache.r6g.xlarge (4vCPU, 16GB) | 3 | $247 | $741 | R$ 3.705 |
| **MSK Kafka** | kafka.m5.large | 3 | $219 | $657 | R$ 3.285 |
| **EC2 - Asterisk** | c6i.2xlarge (8vCPU, 16GB) | 2 | $247 | $494 | R$ 2.470 |
| **EC2 - RTPEngine** | c6i.xlarge (4vCPU, 8GB) | 2 | $124 | $248 | R$ 1.240 |
| **ALB/NLB** | Load Balancers | 3 | $30 | $90 | R$ 450 |
| **EBS Storage** | gp3 SSD | 20 TB | $80/TB | $1.600 | R$ 8.000 |
| **S3 Storage** | Standard | 10 TB | $23/TB | $230 | R$ 1.150 |
| **Data Transfer** | Saída Internet | 5 TB | $90/TB | $450 | R$ 2.250 |
| **CloudWatch** | Logs + Métricas | - | - | $200 | R$ 1.000 |
| **Route53** | DNS gerenciado | - | - | $50 | R$ 250 |
| **Backup** | AWS Backup | - | - | $150 | R$ 750 |
| **Support Business** | 10% do total | - | - | $1.114 | R$ 5.570 |
| **TOTAL MENSAL** | | | | **$12.357** | **≈ R$ 61.785/mês** |

*Câmbio: USD 1 = BRL 5,00 (estimado)

**Custo por Cliente**: R$ 617,85/mês

#### Opção C: Cloud GCP

| Categoria | Recursos | Quantidade | Custo (USD) | Custo (BRL)* |
|-----------|----------|------------|-------------|--------------|
| **GKE Cluster** | Standard + Workers | 5 nodes | $2.800 | R$ 14.000 |
| **Cloud SQL PostgreSQL** | High Availability | 3 instances | $2.400 | R$ 12.000 |
| **Compute Engine - ClickHouse** | n2-highmem-8 | 3 | $1.200 | R$ 6.000 |
| **Memorystore Redis** | M3 | 3 | $720 | R$ 3.600 |
| **Pub/Sub** | Message streaming | - | $300 | R$ 1.500 |
| **Compute - VoIP** | c2-standard-8 | 4 | $950 | R$ 4.750 |
| **Load Balancer** | HTTP/TCP LB | 3 | $90 | R$ 450 |
| **Persistent Disk SSD** | 20 TB | - | $1.700 | R$ 8.500 |
| **Cloud Storage** | Standard | 10 TB | $200 | R$ 1.000 |
| **Network Egress** | 5 TB | - | $450 | R$ 2.250 |
| **Cloud Logging** | Logs retention | - | $180 | R$ 900 |
| **Cloud Monitoring** | Métricas | - | $120 | R$ 600 |
| **Support Standard** | 4% do total | - | $456 | R$ 2.280 |
| **TOTAL MENSAL** | | | **$11.566** | **≈ R$ 57.830/mês** |

**Custo por Cliente**: R$ 578,30/mês

---

## 💰 CENÁRIO 2: 1.000 Clientes + 5 Agentes 24/7

### 📊 Dimensionamento de Infraestrutura

#### Escala Aumentada (10x do Cenário 1)

**Principais Mudanças:**
- Kubernetes: 15 worker nodes (vs 5)
- PostgreSQL: Maior IOPS e storage
- ClickHouse: 6 nós sharded (vs 3)
- Kafka: Maior retenção e throughput
- Asterisk: Mesma configuração (20 chamadas simultâneas suficiente)
- Storage: 80TB para áudio e dados

### 💵 Estimativa de Custos - 1.000 Clientes

#### Opção A: Infraestrutura On-Premise

| Categoria | Especificação | Investimento Inicial | Custo Mensal |
|-----------|---------------|---------------------|--------------|
| **Servidores Físicos** | 25× Dell R750 (2× Xeon Gold, 512GB RAM, 8TB SSD) | R$ 1.250.000 | R$ 12.500 |
| **Switches de Rede** | 4× Switch 48 portas 10Gbps | R$ 120.000 | R$ 1.000 |
| **Firewall** | 2× Firewall HA Fortinet (maior capacidade) | R$ 150.000 | R$ 1.500 |
| **Storage NAS** | 1× Storage NAS 200TB (backup) | R$ 300.000 | R$ 3.000 |
| **UPS/Nobreak** | 4× UPS 10kVA redundantes | R$ 120.000 | R$ 1.200 |
| **Rack & PDU** | 2× Rack 42U + PDUs | R$ 40.000 | R$ 400 |
| **Internet/Link** | Link dedicado 10Gbps redundante | - | R$ 25.000 |
| **Energia Elétrica** | ~50kW consumo médio | - | R$ 30.000 |
| **Datacenter/Colocation** | Espaço físico + refrigeração | - | R$ 35.000 |
| **Equipe Operacional** | 4× SysAdmin + 2× DevOps + 1× Manager | - | R$ 120.000 |
| **Licenças de Software** | SO, Monitoring, Backup | R$ 150.000 | R$ 15.000 |
| **SUBTOTAL** | | **R$ 2.130.000** | **R$ 244.600/mês** |
| **Depreciação (36 meses)** | | | **R$ 59.167/mês** |
| **TOTAL MENSAL** | | | **≈ R$ 303.767/mês** |

**Custo por Cliente**: R$ 303,77/mês

#### Opção B: Cloud AWS

| Categoria | Recursos | Quantidade | Custo (USD) | Custo (BRL)* |
|-----------|----------|------------|-------------|--------------|
| **EKS Control Plane** | Cluster gerenciado | 1 | $144 | R$ 720 |
| **EC2 - K8s Workers** | m6i.4xlarge (16vCPU, 64GB) | 15 | $7.425 | R$ 37.125 |
| **RDS Aurora PostgreSQL** | db.r6g.4xlarge (16vCPU, 128GB) | 3 | $4.950 | R$ 24.750 |
| **EC2 - ClickHouse** | r6i.4xlarge (16vCPU, 64GB) | 6 | $4.956 | R$ 24.780 |
| **ElastiCache Redis** | cache.r6g.2xlarge (8vCPU, 32GB) | 3 | $1.482 | R$ 7.410 |
| **MSK Kafka** | kafka.m5.2xlarge | 3 | $2.628 | R$ 13.140 |
| **EC2 - Asterisk** | c6i.2xlarge (8vCPU, 16GB) | 2 | $494 | R$ 2.470 |
| **EC2 - RTPEngine** | c6i.xlarge (4vCPU, 8GB) | 2 | $248 | R$ 1.240 |
| **ALB/NLB** | Load Balancers | 5 | $150 | R$ 750 |
| **EBS Storage** | gp3 SSD | 80 TB | $6.400 | R$ 32.000 |
| **S3 Storage** | Standard | 100 TB | $2.300 | R$ 11.500 |
| **Data Transfer** | Saída Internet | 20 TB | $1.800 | R$ 9.000 |
| **CloudWatch** | Logs + Métricas | - | $800 | R$ 4.000 |
| **Route53** | DNS gerenciado | - | $100 | R$ 500 |
| **Backup** | AWS Backup | - | $600 | R$ 3.000 |
| **Support Business** | 10% do total | $3.448 | R$ 17.240 |
| **TOTAL MENSAL** | | **$37.925** | **≈ R$ 189.625/mês** |

**Custo por Cliente**: R$ 189,63/mês

#### Opção C: Cloud GCP

| Categoria | Recursos | Quantidade | Custo (USD) | Custo (BRL)* |
|-----------|----------|------------|-------------|--------------|
| **GKE Cluster** | Standard + Workers | 15 nodes | $8.400 | R$ 42.000 |
| **Cloud SQL PostgreSQL** | High Availability | 3 instances | $7.200 | R$ 36.000 |
| **Compute Engine - ClickHouse** | n2-highmem-16 | 6 | $7.200 | R$ 36.000 |
| **Memorystore Redis** | M5 | 3 | $2.160 | R$ 10.800 |
| **Pub/Sub** | Message streaming | - | $1.200 | R$ 6.000 |
| **Compute - VoIP** | c2-standard-8 | 4 | $950 | R$ 4.750 |
| **Load Balancer** | HTTP/TCP LB | 5 | $150 | R$ 750 |
| **Persistent Disk SSD** | 80 TB | - | $6.800 | R$ 34.000 |
| **Cloud Storage** | Standard | 100 TB | $2.000 | R$ 10.000 |
| **Network Egress** | 20 TB | - | $1.800 | R$ 9.000 |
| **Cloud Logging** | Logs retention | - | $720 | R$ 3.600 |
| **Cloud Monitoring** | Métricas | - | $480 | R$ 2.400 |
| **Support Standard** | 4% do total | $1.564 | R$ 7.820 |
| **TOTAL MENSAL** | | **$40.624** | **≈ R$ 203.120/mês** |

**Custo por Cliente**: R$ 203,12/mês

---

## 📊 Comparação de Custos

### Cenário 1: 100 Clientes

| Opção | Custo Mensal Total | Custo por Cliente | Investimento Inicial |
|-------|-------------------|-------------------|---------------------|
| **On-Premise** | R$ 117.267 | R$ 1.172,67 | R$ 870.000 |
| **AWS** | R$ 61.785 | R$ 617,85 | R$ 0 |
| **GCP** | R$ 57.830 | R$ 578,30 | R$ 0 |

### Cenário 2: 1.000 Clientes

| Opção | Custo Mensal Total | Custo por Cliente | Investimento Inicial |
|-------|-------------------|-------------------|---------------------|
| **On-Premise** | R$ 303.767 | R$ 303,77 | R$ 2.130.000 |
| **AWS** | R$ 189.625 | R$ 189,63 | R$ 0 |
| **GCP** | R$ 203.120 | R$ 203,12 | R$ 0 |

---

## 💡 Custos Adicionais (Todos os Cenários)

### Serviços de IA/STT/TTS (não incluídos acima)

| Serviço | Provider | Custo Estimado |
|---------|----------|----------------|
| **Speech-to-Text** | Google/Azure/AWS | $0,006/minuto = ~R$ 0,03/min |
| **Text-to-Speech** | Google/ElevenLabs | $0,016/1K chars = ~R$ 0,08/resposta |
| **LLM (GPT-4)** | OpenAI | $0,03/1K tokens = ~R$ 0,45/interação |
| **LLM (Claude)** | Anthropic | $0,025/1K tokens = ~R$ 0,38/interação |

**Estimativa por Chamada (5 min):**
- STT: R$ 0,15
- TTS: R$ 0,32 (4 respostas)
- LLM: R$ 1,80 (4 interações)
- **Total: ~R$ 2,27 por chamada**

**Cenário 1 (100 clientes):**
- 1.000 chamadas/cliente/mês = 100.000 chamadas/mês
- Custo IA: R$ 227.000/mês

**Cenário 2 (1.000 clientes):**
- 1.000 chamadas/cliente/mês = 1.000.000 chamadas/mês
- Custo IA: R$ 2.270.000/mês

### Outros Custos Mensais

- **Telefonia (Trunks SIP)**: R$ 0,05-0,15/minuto
- **Números DIDs**: R$ 30-50/número/mês
- **SMS (notificações)**: R$ 0,10-0,25/SMS
- **Certificados SSL**: R$ 0-500/mês (Let's Encrypt grátis)
- **CDN (se necessário)**: R$ 200-2.000/mês
- **Disaster Recovery/Backup externo**: R$ 1.000-5.000/mês

---

## 🎯 Recomendações

### Para 100 Clientes:
✅ **Recomendado: AWS ou GCP**
- Menor investimento inicial
- Custo por cliente mais competitivo
- Maior flexibilidade para escalar
- Menor overhead operacional

### Para 1.000 Clientes:
✅ **Recomendado: Híbrido (On-Premise + Cloud)**
- On-premise para workloads estáveis (K8s, databases)
- Cloud para picos de demanda e IA services
- Melhor custo-benefício em escala
- Break-even vs cloud puro em ~7-8 meses

### Estratégia de Crescimento:
1. **0-100 clientes**: 100% Cloud
2. **100-500 clientes**: Avaliar ROI on-premise
3. **500-1000 clientes**: Migração gradual para híbrido
4. **1000+ clientes**: Infraestrutura própria com cloud burst

---

## 📝 Notas Importantes

1. **Custos de IA dominam** a operação (70-90% do custo total operacional)
2. **Margem de erro**: ±20% nas estimativas
3. **Otimizações possíveis**:
   - Uso de LLMs open-source (Llama, Mistral) pode reduzir custos de IA em 80%
   - Cache de respostas comuns pode reduzir 30-40% das chamadas LLM
   - Spot instances/Preemptible VMs podem reduzir custo cloud em 60-70%
4. **Custos não incluídos**:
   - Desenvolvimento e manutenção de software
   - Marketing e vendas
   - Suporte ao cliente
   - Compliance e auditoria

---

## 📞 Resumo para Tomada de Decisão

| Métrica | 100 Clientes | 1.000 Clientes |
|---------|--------------|----------------|
| **Infraestrutura/mês** | R$ 58-117K | R$ 190-304K |
| **IA Services/mês** | R$ 227K | R$ 2.270K |
| **Telefonia/mês** | R$ 25K | R$ 250K |
| **TOTAL OPERACIONAL** | **R$ 310-369K/mês** | **R$ 2.710-2.824K/mês** |
| **Receita mínima necessária** | R$ 3.100/cliente | R$ 2.710/cliente |
| **(para 50% margem)** | R$ 6.200/cliente | R$ 5.420/cliente |

---

**Documento gerado em:** Dezembro 2024  
**Versão:** 1.0  
**Contato:** Para dúvidas ou ajustes nos cálculos, consulte a equipe de infraestrutura.
