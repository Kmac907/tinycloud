package config

import runtimeconfig "tinycloud/runtime/tinycloudconfig"

type Config = runtimeconfig.Config
type Service = runtimeconfig.Service
type ServiceDescriptor = runtimeconfig.ServiceDescriptor
type ServiceSelection = runtimeconfig.ServiceSelection

const (
	ServiceManagement = runtimeconfig.ServiceManagement
	ServiceBlob       = runtimeconfig.ServiceBlob
	ServiceQueue      = runtimeconfig.ServiceQueue
	ServiceTable      = runtimeconfig.ServiceTable
	ServiceKeyVault   = runtimeconfig.ServiceKeyVault
	ServiceServiceBus = runtimeconfig.ServiceServiceBus
	ServiceAppConfig  = runtimeconfig.ServiceAppConfig
	ServiceCosmos     = runtimeconfig.ServiceCosmos
	ServiceDNS        = runtimeconfig.ServiceDNS
	ServiceEventHubs  = runtimeconfig.ServiceEventHubs
)

var (
	FromEnv               = runtimeconfig.FromEnv
	ParseServiceSelection = runtimeconfig.ParseServiceSelection
)
