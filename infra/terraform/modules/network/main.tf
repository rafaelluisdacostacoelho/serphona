# Network Module - On-Premise & Cloud Portable
# Supports: Proxmox, VMware vSphere, AWS, GCP, Azure

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
  description = "Deployment target: on-premise, aws, gcp, azure"
  type        = string
  default     = "on-premise"
}

variable "network_cidr" {
  description = "Base network CIDR"
  type        = string
  default     = "10.0.0.0/16"
}

variable "vlans" {
  description = "VLAN configuration"
  type = map(object({
    id          = number
    name        = string
    cidr        = string
    gateway     = string
    description = string
  }))
  default = {
    dmz = {
      id          = 10
      name        = "dmz"
      cidr        = "10.0.0.0/24"
      gateway     = "10.0.0.1"
      description = "DMZ - Edge services, load balancers"
    }
    voip = {
      id          = 20
      name        = "voip"
      cidr        = "10.1.0.0/24"
      gateway     = "10.1.0.1"
      description = "VoIP - Asterisk, Kamailio, RTPEngine"
    }
    app = {
      id          = 30
      name        = "app"
      cidr        = "10.2.0.0/24"
      gateway     = "10.2.0.1"
      description = "Application - Kubernetes cluster"
    }
    data = {
      id          = 40
      name        = "data"
      cidr        = "10.3.0.0/24"
      gateway     = "10.3.0.1"
      description = "Data - Databases, storage"
    }
    mgmt = {
      id          = 50
      name        = "mgmt"
      cidr        = "10.4.0.0/24"
      gateway     = "10.4.0.1"
      description = "Management - Bastion, monitoring"
    }
  }
}

variable "dns_servers" {
  description = "DNS server addresses"
  type        = list(string)
  default     = ["10.4.0.10", "10.4.0.11"]
}

variable "ntp_servers" {
  description = "NTP server addresses"
  type        = list(string)
  default     = ["10.4.0.12"]
}

# =============================================================================
# Local Values
# =============================================================================

locals {
  name_prefix = "${var.environment}-${var.region}"
  
  # Firewall rules for inter-VLAN communication
  firewall_rules = {
    # DMZ to VoIP (SIP/RTP)
    dmz_to_voip = {
      name        = "dmz-to-voip"
      source      = var.vlans.dmz.cidr
      destination = var.vlans.voip.cidr
      ports       = ["5060/udp", "5061/tcp", "10000-20000/udp"]
      action      = "allow"
    }
    # DMZ to App (HTTP/HTTPS)
    dmz_to_app = {
      name        = "dmz-to-app"
      source      = var.vlans.dmz.cidr
      destination = var.vlans.app.cidr
      ports       = ["80/tcp", "443/tcp", "6443/tcp"]
      action      = "allow"
    }
    # App to Data (Database ports)
    app_to_data = {
      name        = "app-to-data"
      source      = var.vlans.app.cidr
      destination = var.vlans.data.cidr
      ports       = ["5432/tcp", "6379/tcp", "9000/tcp", "9092/tcp", "8123/tcp"]
      action      = "allow"
    }
    # VoIP to App (ARI, Kafka)
    voip_to_app = {
      name        = "voip-to-app"
      source      = var.vlans.voip.cidr
      destination = var.vlans.app.cidr
      ports       = ["8088/tcp", "9092/tcp"]
      action      = "allow"
    }
    # VoIP to Data (Recording storage)
    voip_to_data = {
      name        = "voip-to-data"
      source      = var.vlans.voip.cidr
      destination = var.vlans.data.cidr
      ports       = ["9000/tcp"]
      action      = "allow"
    }
    # Management to All
    mgmt_to_all = {
      name        = "mgmt-to-all"
      source      = var.vlans.mgmt.cidr
      destination = var.network_cidr
      ports       = ["22/tcp", "443/tcp"]
      action      = "allow"
    }
  }

  common_tags = {
    Environment = var.environment
    Region      = var.region
    ManagedBy   = "terraform"
    Project     = "voicecustomer"
  }
}

# =============================================================================
# On-Premise Network (Proxmox)
# =============================================================================

# This section would use the Proxmox provider for on-premise deployments
# resource "proxmox_network_vlan" "vlans" {
#   for_each = var.deployment_target == "on-premise" ? var.vlans : {}
#   
#   name    = "${local.name_prefix}-${each.value.name}"
#   vlan_id = each.value.id
#   bridge  = "vmbr0"
#   comment = each.value.description
# }

# =============================================================================
# AWS VPC (Cloud)
# =============================================================================

resource "aws_vpc" "main" {
  count = var.deployment_target == "aws" ? 1 : 0

  cidr_block           = var.network_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-vpc"
  })
}

resource "aws_subnet" "subnets" {
  for_each = var.deployment_target == "aws" ? var.vlans : {}

  vpc_id            = aws_vpc.main[0].id
  cidr_block        = each.value.cidr
  availability_zone = "${var.region}a"

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-${each.value.name}"
    Type = each.key
  })
}

resource "aws_internet_gateway" "main" {
  count = var.deployment_target == "aws" ? 1 : 0

  vpc_id = aws_vpc.main[0].id

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-igw"
  })
}

resource "aws_route_table" "public" {
  count = var.deployment_target == "aws" ? 1 : 0

  vpc_id = aws_vpc.main[0].id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main[0].id
  }

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-public-rt"
  })
}

resource "aws_security_group" "vlan_sg" {
  for_each = var.deployment_target == "aws" ? var.vlans : {}

  name        = "${local.name_prefix}-${each.value.name}-sg"
  description = each.value.description
  vpc_id      = aws_vpc.main[0].id

  tags = merge(local.common_tags, {
    Name = "${local.name_prefix}-${each.value.name}-sg"
  })
}

# =============================================================================
# GCP VPC (Cloud)
# =============================================================================

resource "google_compute_network" "main" {
  count = var.deployment_target == "gcp" ? 1 : 0

  name                    = "${local.name_prefix}-vpc"
  auto_create_subnetworks = false
  routing_mode            = "REGIONAL"
}

resource "google_compute_subnetwork" "subnets" {
  for_each = var.deployment_target == "gcp" ? var.vlans : {}

  name          = "${local.name_prefix}-${each.value.name}"
  ip_cidr_range = each.value.cidr
  region        = var.region
  network       = google_compute_network.main[0].id

  private_ip_google_access = true
}

resource "google_compute_firewall" "rules" {
  for_each = var.deployment_target == "gcp" ? local.firewall_rules : {}

  name    = "${local.name_prefix}-${each.value.name}"
  network = google_compute_network.main[0].name

  allow {
    protocol = split("/", each.value.ports[0])[1]
    ports    = [for p in each.value.ports : split("/", p)[0]]
  }

  source_ranges = [each.value.source]
  target_tags   = [each.key]
}

# =============================================================================
# Outputs
# =============================================================================

output "network_id" {
  description = "Network/VPC ID"
  value = coalesce(
    try(aws_vpc.main[0].id, ""),
    try(google_compute_network.main[0].id, ""),
    "on-premise-network"
  )
}

output "subnet_ids" {
  description = "Map of subnet IDs by name"
  value = var.deployment_target == "aws" ? {
    for k, v in aws_subnet.subnets : k => v.id
  } : var.deployment_target == "gcp" ? {
    for k, v in google_compute_subnetwork.subnets : k => v.id
  } : {
    for k, v in var.vlans : k => "vlan-${v.id}"
  }
}

output "vlan_config" {
  description = "VLAN configuration"
  value       = var.vlans
}

output "firewall_rules" {
  description = "Firewall rules configuration"
  value       = local.firewall_rules
}
