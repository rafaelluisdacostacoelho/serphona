# ==============================================================================
# Serphona - Development Environment
# ==============================================================================

terraform {
  required_version = ">= 1.6"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23"
    }
  }

  # Backend configuration for state storage
  # backend "s3" {
  #   bucket = "serphona-terraform-state"
  #   key    = "dev/terraform.tfstate"
  #   region = "us-east-1"
  # }
}

# ==============================================================================
# Modules
# ==============================================================================

module "network" {
  source = "../../modules/network"

  environment = "dev"
  vpc_cidr    = "10.0.0.0/16"
}

module "postgresql" {
  source = "../../modules/postgresql"

  environment     = "dev"
  instance_class  = "db.t3.medium"
  storage_size_gb = 100
}
