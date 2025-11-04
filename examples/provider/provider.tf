terraform {
  required_providers {
    victoriametricscloud = {
      source  = "VictoriaMetrics/victoriametricscloud"
      version = "0.0.1"
    }
  }
}

provider "victoriametricscloud" {
  api_key = var.api_key
}
