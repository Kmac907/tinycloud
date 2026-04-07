terraform {
  required_version = ">= 1.6.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}
  subscription_id            = var.subscription_id
  tenant_id                  = var.tenant_id
  resource_provider_registrations = "none"
}

variable "subscription_id" {
  type    = string
  default = "11111111-1111-1111-1111-111111111111"
}

variable "tenant_id" {
  type    = string
  default = "00000000-0000-0000-0000-000000000001"
}

resource "azurerm_resource_group" "example" {
  name     = "tinycloud-rg"
  location = "westus2"

  tags = {
    environment = "local"
    managed_by  = "tinycloud"
  }
}
