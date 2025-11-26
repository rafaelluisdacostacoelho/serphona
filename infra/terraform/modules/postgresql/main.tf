# PostgreSQL Module - Multi-Tenant Database with HA
# Supports: On-Premise (Patroni), AWS RDS, GCP Cloud SQL

terraform {
  required_version = ">= 1.6.0"
}

# =============================================================================
# Variables
# =============================================================================

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
}

variable "region" {
  description = "Deployment region/datacenter"
  type        = string
}

variable "deployment_target" {
  description = "Deployment target: on-premise, aws, gcp"
  type        = string
  default     = "on-premise"
}

variable "cluster_name" {
  description = "PostgreSQL cluster name"
  type        = string
  default     = "pg-data"
}

variable "postgres_version" {
  description = "PostgreSQL version"
  type        = string
  default     = "16"
}

variable "node_count" {
  description = "Number of PostgreSQL nodes (1 primary + N-1 replicas)"
  type        = number
  default     = 3
}

variable "storage_size_gb" {
  description = "Storage size per node in GB"
  type        = number
  default     = 500
}

variable "enable_ha" {
  description = "Enable high availability"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Backup retention in days"
  type        = number
  default     = 30
}

variable "enable_encryption" {
  description = "Enable encryption at rest"
  type        = bool
  default     = true
}

variable "subnet_id" {
  description = "Subnet ID for deployment"
  type        = string
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to connect"
  type        = list(string)
  default     = ["10.2.0.0/24"]  # App network
}

# On-Premise specific
variable "vm_cpu_cores" {
  description = "CPU cores per VM (on-premise)"
  type        = number
  default     = 8
}

variable "vm_memory_mb" {
  description = "Memory in MB per VM (on-premise)"
  type        = number
  default     = 65536  # 64GB
}

variable "vm_template" {
  description = "VM template name (on-premise)"
  type        = string
  default     = "ubuntu-22.04-template"
}

# AWS specific
variable "aws_instance_class" {
  description = "RDS instance class"
  type        = string
  default     = "db.r6g.2xlarge"
}

# GCP specific
variable "gcp_tier" {
  description = "Cloud SQL tier"
  type        = string
  default     = "db-custom-8-65536"
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  name_prefix = "${var.cluster_name}-${var.environment}-${var.region}"
  
  # Multi-tenant database configuration
  databases = {
    app = {
      name        = "app_db"
      description = "Main application database"
      extensions  = ["uuid-ossp", "pgcrypto", "pg_stat_statements"]
    }
    analytics = {
      name        = "analytics_db"
      description = "Analytics staging database"
      extensions  = ["uuid-ossp", "pgcrypto"]
    }
  }

  # Connection pooling settings
  pgbouncer_settings = {
    pool_mode                = "transaction"
    max_client_conn          = 10000
    default_pool_size        = 100
    min_pool_size            = 10
    reserve_pool_size        = 25
    reserve_pool_timeout     = 5
    max_db_connections       = 200
    max_user_connections     = 200
    server_reset_query       = "DISCARD ALL"
    server_check_query       = "SELECT 1"
    server_check_delay       = 30
    query_timeout            = 120
    client_idle_timeout      = 600
  }

  common_tags = {
    Environment = var.environment
    Region      = var.region
    ManagedBy   = "terraform"
    Project     = "voicecustomer"
    Component   = "postgresql"
  }

  # PostgreSQL configuration parameters
  pg_params = {
    # Memory
    shared_buffers             = "16GB"
    effective_cache_size       = "48GB"
    work_mem                   = "256MB"
    maintenance_work_mem       = "2GB"
    
    # Checkpoints
    checkpoint_completion_target = "0.9"
    wal_buffers                = "64MB"
    min_wal_size               = "1GB"
    max_wal_size               = "4GB"
    
    # Query planning
    random_page_cost           = "1.1"
    effective_io_concurrency   = "200"
    
    # Connections
    max_connections            = "500"
    
    # Logging
    log_statement              = "all"
    log_connections            = "on"
    log_disconnections         = "on"
    log_lock_waits             = "on"
    log_min_duration_statement = "1000"  # Log queries > 1s
    
    # Row-level security
    row_security               = "on"
    
    # SSL
    ssl                        = "on"
    ssl_min_protocol_version   = "TLSv1.3"
  }
}

# =============================================================================
# On-Premise PostgreSQL (Patroni Cluster)
# =============================================================================

# Note: On-premise deployment uses Ansible for actual provisioning
# This generates the configuration for Ansible to consume

resource "local_file" "patroni_inventory" {
  count = var.deployment_target == "on-premise" ? 1 : 0

  filename = "${path.module}/generated/patroni-inventory.yml"
  content  = yamlencode({
    all = {
      vars = {
        cluster_name      = var.cluster_name
        postgres_version  = var.postgres_version
        patroni_scope     = local.name_prefix
        etcd_cluster_name = "${local.name_prefix}-etcd"
      }
      children = {
        postgresql = {
          hosts = {
            for i in range(var.node_count) : "${var.cluster_name}-${var.environment}-${var.region}-${format("%02d", i + 1)}" => {
              ansible_host    = "10.3.0.${10 + i}"
              patroni_role    = i == 0 ? "primary" : "replica"
              replication_slot = i == 0 ? null : "replica_${i}"
            }
          }
        }
        pgbouncer = {
          hosts = {
            "${var.cluster_name}-pgbouncer-${var.environment}-${var.region}-01" = {
              ansible_host = "10.3.0.100"
            }
            "${var.cluster_name}-pgbouncer-${var.environment}-${var.region}-02" = {
              ansible_host = "10.3.0.101"
            }
          }
        }
        haproxy = {
          hosts = {
            "${var.cluster_name}-haproxy-${var.environment}-${var.region}-01" = {
              ansible_host   = "10.3.0.110"
              haproxy_role   = "primary"
            }
            "${var.cluster_name}-haproxy-${var.environment}-${var.region}-02" = {
              ansible_host   = "10.3.0.111"
              haproxy_role   = "backup"
            }
          }
        }
      }
    }
  })
}

resource "local_file" "patroni_config" {
  count = var.deployment_target == "on-premise" ? 1 : 0

  filename = "${path.module}/generated/patroni-config.yml"
  content  = yamlencode({
    scope        = local.name_prefix
    namespace    = "/service/"
    name         = var.cluster_name
    
    restapi = {
      listen         = "0.0.0.0:8008"
      connect_address = "$${HOSTNAME}:8008"
    }
    
    etcd3 = {
      hosts = "etcd-01:2379,etcd-02:2379,etcd-03:2379"
    }
    
    bootstrap = {
      dcs = {
        ttl                   = 30
        loop_wait             = 10
        retry_timeout         = 10
        maximum_lag_on_failover = 1048576
        postgresql = {
          use_pg_rewind = true
          use_slots     = true
          parameters    = local.pg_params
        }
      }
      initdb = [
        { encoding = "UTF8" },
        { "data-checksums" = true }
      ]
    }
    
    postgresql = {
      listen         = "0.0.0.0:5432"
      connect_address = "$${HOSTNAME}:5432"
      data_dir       = "/var/lib/postgresql/data"
      bin_dir        = "/usr/lib/postgresql/${var.postgres_version}/bin"
      authentication = {
        superuser = {
          username = "postgres"
        }
        replication = {
          username = "replicator"
        }
      }
      parameters = local.pg_params
      pg_hba = [
        "local   all             all                                     peer",
        "host    all             all             127.0.0.1/32            scram-sha-256",
        "host    all             all             10.0.0.0/8              scram-sha-256",
        "host    replication     replicator      10.3.0.0/24             scram-sha-256"
      ]
    }
    
    watchdog = {
      mode     = "automatic"
      device   = "/dev/watchdog"
      safety_margin = 5
    }
  })
}

# =============================================================================
# AWS RDS PostgreSQL
# =============================================================================

resource "aws_db_subnet_group" "postgresql" {
  count = var.deployment_target == "aws" ? 1 : 0

  name       = "${local.name_prefix}-subnet-group"
  subnet_ids = [var.subnet_id]

  tags = local.common_tags
}

resource "aws_security_group" "postgresql" {
  count = var.deployment_target == "aws" ? 1 : 0

  name        = "${local.name_prefix}-sg"
  description = "Security group for PostgreSQL RDS"
  vpc_id      = data.aws_subnet.selected[0].vpc_id

  ingress {
    description = "PostgreSQL from allowed networks"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = local.common_tags
}

data "aws_subnet" "selected" {
  count = var.deployment_target == "aws" ? 1 : 0
  id    = var.subnet_id
}

resource "aws_db_parameter_group" "postgresql" {
  count = var.deployment_target == "aws" ? 1 : 0

  family = "postgres${var.postgres_version}"
  name   = "${local.name_prefix}-params"

  parameter {
    name  = "log_statement"
    value = "all"
  }

  parameter {
    name  = "log_connections"
    value = "1"
  }

  parameter {
    name  = "log_disconnections"
    value = "1"
  }

  parameter {
    name  = "log_lock_waits"
    value = "1"
  }

  parameter {
    name  = "log_min_duration_statement"
    value = "1000"
  }

  parameter {
    name  = "shared_preload_libraries"
    value = "pg_stat_statements"
  }

  tags = local.common_tags
}

resource "aws_rds_cluster" "postgresql" {
  count = var.deployment_target == "aws" ? 1 : 0

  cluster_identifier = local.name_prefix
  engine             = "aurora-postgresql"
  engine_version     = var.postgres_version
  database_name      = "app_db"
  master_username    = "postgres"
  master_password    = random_password.db_password[0].result

  db_subnet_group_name   = aws_db_subnet_group.postgresql[0].name
  vpc_security_group_ids = [aws_security_group.postgresql[0].id]

  backup_retention_period = var.backup_retention_days
  preferred_backup_window = "02:00-03:00"
  
  storage_encrypted = var.enable_encryption
  
  enabled_cloudwatch_logs_exports = ["postgresql"]

  tags = local.common_tags
}

resource "aws_rds_cluster_instance" "postgresql" {
  count = var.deployment_target == "aws" ? var.node_count : 0

  identifier         = "${local.name_prefix}-${count.index + 1}"
  cluster_identifier = aws_rds_cluster.postgresql[0].id
  instance_class     = var.aws_instance_class
  engine             = "aurora-postgresql"

  tags = local.common_tags
}

resource "random_password" "db_password" {
  count = var.deployment_target != "on-premise" ? 1 : 0

  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# =============================================================================
# GCP Cloud SQL PostgreSQL
# =============================================================================

resource "google_sql_database_instance" "postgresql" {
  count = var.deployment_target == "gcp" ? 1 : 0

  name             = local.name_prefix
  database_version = "POSTGRES_${var.postgres_version}"
  region           = var.region

  settings {
    tier              = var.gcp_tier
    availability_type = var.enable_ha ? "REGIONAL" : "ZONAL"
    disk_size         = var.storage_size_gb
    disk_type         = "PD_SSD"
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
      start_time                     = "02:00"
      transaction_log_retention_days = 7
      backup_retention_settings {
        retained_backups = var.backup_retention_days
      }
    }

    ip_configuration {
      ipv4_enabled    = false
      private_network = var.subnet_id
      require_ssl     = true
    }

    database_flags {
      name  = "log_statement"
      value = "all"
    }

    database_flags {
      name  = "log_connections"
      value = "on"
    }

    database_flags {
      name  = "log_disconnections"
      value = "on"
    }

    insights_config {
      query_insights_enabled  = true
      record_application_tags = true
      record_client_address   = true
    }
  }

  deletion_protection = var.environment == "production"
}

resource "google_sql_database" "databases" {
  for_each = var.deployment_target == "gcp" ? local.databases : {}

  name     = each.value.name
  instance = google_sql_database_instance.postgresql[0].name
}

resource "google_sql_user" "postgres" {
  count = var.deployment_target == "gcp" ? 1 : 0

  name     = "postgres"
  instance = google_sql_database_instance.postgresql[0].name
  password = random_password.db_password[0].result
}

# =============================================================================
# Outputs
# =============================================================================

output "connection_string" {
  description = "PostgreSQL connection string"
  sensitive   = true
  value = var.deployment_target == "aws" ? (
    "postgresql://postgres:${random_password.db_password[0].result}@${aws_rds_cluster.postgresql[0].endpoint}:5432/app_db?sslmode=require"
  ) : var.deployment_target == "gcp" ? (
    "postgresql://postgres:${random_password.db_password[0].result}@${google_sql_database_instance.postgresql[0].private_ip_address}:5432/app_db?sslmode=require"
  ) : (
    "postgresql://postgres@${var.cluster_name}-haproxy.data.svc:5432/app_db?sslmode=require"
  )
}

output "endpoint" {
  description = "PostgreSQL endpoint"
  value = var.deployment_target == "aws" ? (
    aws_rds_cluster.postgresql[0].endpoint
  ) : var.deployment_target == "gcp" ? (
    google_sql_database_instance.postgresql[0].private_ip_address
  ) : (
    "${var.cluster_name}-haproxy-${var.environment}-${var.region}-01"
  )
}

output "port" {
  description = "PostgreSQL port"
  value       = 5432
}

output "database_names" {
  description = "List of database names"
  value       = [for db in local.databases : db.name]
}

output "patroni_inventory_path" {
  description = "Path to generated Patroni inventory (on-premise only)"
  value       = var.deployment_target == "on-premise" ? local_file.patroni_inventory[0].filename : null
}

output "pgbouncer_settings" {
  description = "PgBouncer configuration settings"
  value       = local.pgbouncer_settings
}
