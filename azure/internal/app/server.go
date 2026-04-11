package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"tinycloud/internal/api/admin"
	"tinycloud/internal/api/arm"
	"tinycloud/internal/auth"
	"tinycloud/internal/config"
	"tinycloud/internal/httpx"
	"tinycloud/internal/identity"
	"tinycloud/internal/metadata"
	"tinycloud/internal/providers/appconfig"
	"tinycloud/internal/providers/cosmos"
	"tinycloud/internal/providers/dns"
	"tinycloud/internal/providers/eventhubs"
	"tinycloud/internal/providers/keyvault"
	"tinycloud/internal/providers/queue"
	"tinycloud/internal/providers/servicebus"
	"tinycloud/internal/providers/storage"
	"tinycloud/internal/providers/table"
	"tinycloud/internal/state"
	"tinycloud/internal/telemetry"
)

type Server struct {
	cfg    config.Config
	store  *state.Store
	logger *telemetry.Logger
}

func NewServer(cfg config.Config, store *state.Store, logger *telemetry.Logger) *Server {
	return &Server{
		cfg:    cfg,
		store:  store,
		logger: logger,
	}
}

func (s *Server) Run(ctx context.Context) error {
	if err := s.cfg.Validate(); err != nil {
		return err
	}
	if err := s.cfg.RequireServices(); err != nil {
		return err
	}
	if err := s.store.Init(); err != nil {
		return err
	}
	if s.cfg.ServiceEnabled(config.ServiceManagement) {
		if err := EnsureManagementTLSCertFiles(s.cfg); err != nil {
			return err
		}
	}

	var server *http.Server
	var managementTLSServer *http.Server
	var blobServer *http.Server
	var queueServer *http.Server
	var tableServer *http.Server
	var serviceBusServer *http.Server
	var keyVaultServer *http.Server
	var appConfigServer *http.Server
	var cosmosServer *http.Server
	var eventHubsServer *http.Server
	var dnsServer *dns.Server

	if s.cfg.ServiceEnabled(config.ServiceManagement) {
		mux := http.NewServeMux()
		authService := auth.NewService(s.cfg)
		admin.NewHandler(s.store, s.cfg).Register(mux)
		arm.NewHandler(s.store, s.cfg).Register(mux)
		auth.NewHandler(authService).Register(mux)
		identity.NewHandler(s.cfg, authService).Register(mux)
		metadata.NewHandler(s.cfg).Register(mux)
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			httpx.WriteJSON(w, http.StatusOK, map[string]string{
				"name":   "TinyCloud",
				"status": "running",
			})
		})

		handler := chain(
			mux,
			withRequestID,
			withLogging(s.logger),
			withRecovery(s.logger),
			withCORS,
			withAzureHeaders,
			withAPIVersion,
		)
		server = &http.Server{
			Addr:              s.cfg.ManagementAddr(),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		}
		managementTLSServer = &http.Server{
			Addr:              s.cfg.ManagementTLSAddr(),
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceBlob) {
		blobMux := http.NewServeMux()
		storage.NewHandler(s.store, s.cfg).Register(blobMux)
		blobServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.Blob,
			Handler:           chain(blobMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceQueue) {
		queueMux := http.NewServeMux()
		queue.NewHandler(s.store, s.cfg).Register(queueMux)
		queueServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.Queue,
			Handler:           chain(queueMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceTable) {
		tableMux := http.NewServeMux()
		table.NewHandler(s.store, s.cfg).Register(tableMux)
		tableServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.Table,
			Handler:           chain(tableMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceServiceBus) {
		serviceBusMux := http.NewServeMux()
		servicebus.NewHandler(s.store, s.cfg).Register(serviceBusMux)
		serviceBusServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.ServiceBus,
			Handler:           chain(serviceBusMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceKeyVault) {
		keyVaultMux := http.NewServeMux()
		keyvault.NewHandler(s.store, s.cfg).Register(keyVaultMux)
		keyVaultServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.KeyVault,
			Handler:           chain(keyVaultMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceAppConfig) {
		appConfigMux := http.NewServeMux()
		appconfig.NewHandler(s.store, s.cfg).Register(appConfigMux)
		appConfigServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.AppConfig,
			Handler:           chain(appConfigMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceCosmos) {
		cosmosMux := http.NewServeMux()
		cosmos.NewHandler(s.store, s.cfg).Register(cosmosMux)
		cosmosServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.Cosmos,
			Handler:           chain(cosmosMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceEventHubs) {
		eventHubsMux := http.NewServeMux()
		eventhubs.NewHandler(s.store, s.cfg).Register(eventHubsMux)
		eventHubsServer = &http.Server{
			Addr:              s.cfg.ListenHost + ":" + s.cfg.EventHubs,
			Handler:           chain(eventHubsMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
			ReadHeaderTimeout: 5 * time.Second,
		}
	}

	if s.cfg.ServiceEnabled(config.ServiceDNS) {
		dnsServer = dns.NewServer(s.store, s.cfg, s.logger)
	}

	logFields := map[string]any{
		"dataRoot":         s.cfg.DataRoot,
		"enabledServices":  s.cfg.EnabledServices(),
		"disabledServices": s.cfg.DisabledServices(),
	}
	if server != nil {
		logFields["addr"] = s.cfg.ManagementAddr()
		logFields["managementTLSAddr"] = s.cfg.ManagementTLSAddr()
		logFields["managementTLSCert"] = s.cfg.ManagementTLSCertPath()
	}
	if blobServer != nil {
		logFields["blobAddr"] = s.cfg.ListenHost + ":" + s.cfg.Blob
	}
	if queueServer != nil {
		logFields["queueAddr"] = s.cfg.ListenHost + ":" + s.cfg.Queue
	}
	if tableServer != nil {
		logFields["tableAddr"] = s.cfg.ListenHost + ":" + s.cfg.Table
	}
	if serviceBusServer != nil {
		logFields["serviceBusAddr"] = s.cfg.ListenHost + ":" + s.cfg.ServiceBus
	}
	if keyVaultServer != nil {
		logFields["keyVaultAddr"] = s.cfg.ListenHost + ":" + s.cfg.KeyVault
	}
	if appConfigServer != nil {
		logFields["appConfigAddr"] = s.cfg.ListenHost + ":" + s.cfg.AppConfig
	}
	if cosmosServer != nil {
		logFields["cosmosAddr"] = s.cfg.ListenHost + ":" + s.cfg.Cosmos
	}
	if eventHubsServer != nil {
		logFields["eventHubsAddr"] = s.cfg.ListenHost + ":" + s.cfg.EventHubs
	}
	if dnsServer != nil {
		logFields["dnsAddr"] = s.cfg.ListenHost + ":" + s.cfg.DNS
	}
	s.logger.Info("tinycloud server starting", logFields)

	errCh := make(chan error, 11)
	if server != nil {
		go func() {
			errCh <- server.ListenAndServe()
		}()
	}
	if managementTLSServer != nil {
		go func() {
			errCh <- managementTLSServer.ListenAndServeTLS(s.cfg.ManagementTLSCertPath(), s.cfg.ManagementTLSKeyPath())
		}()
	}
	if blobServer != nil {
		go func() {
			errCh <- blobServer.ListenAndServe()
		}()
	}
	if queueServer != nil {
		go func() {
			errCh <- queueServer.ListenAndServe()
		}()
	}
	if tableServer != nil {
		go func() {
			errCh <- tableServer.ListenAndServe()
		}()
	}
	if serviceBusServer != nil {
		go func() {
			errCh <- serviceBusServer.ListenAndServe()
		}()
	}
	if keyVaultServer != nil {
		go func() {
			errCh <- keyVaultServer.ListenAndServe()
		}()
	}
	if appConfigServer != nil {
		go func() {
			errCh <- appConfigServer.ListenAndServe()
		}()
	}
	if cosmosServer != nil {
		go func() {
			errCh <- cosmosServer.ListenAndServe()
		}()
	}
	if eventHubsServer != nil {
		go func() {
			errCh <- eventHubsServer.ListenAndServe()
		}()
	}
	if dnsServer != nil {
		go func() {
			errCh <- dnsServer.ListenAndServe()
		}()
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.logger.Info("tinycloud server stopping", nil)
		if err := shutdownHTTPServer(keyVaultServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(managementTLSServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(appConfigServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(cosmosServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(eventHubsServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownDNSServer(dnsServer); err != nil {
			return err
		}
		if err := shutdownHTTPServer(queueServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(tableServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(serviceBusServer, shutdownCtx); err != nil {
			return err
		}
		if err := shutdownHTTPServer(blobServer, shutdownCtx); err != nil {
			return err
		}
		return shutdownHTTPServer(server, shutdownCtx)
	case err := <-errCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = shutdownHTTPServer(keyVaultServer, shutdownCtx)
		_ = shutdownHTTPServer(managementTLSServer, shutdownCtx)
		_ = shutdownHTTPServer(appConfigServer, shutdownCtx)
		_ = shutdownHTTPServer(cosmosServer, shutdownCtx)
		_ = shutdownHTTPServer(eventHubsServer, shutdownCtx)
		_ = shutdownDNSServer(dnsServer)
		_ = shutdownHTTPServer(queueServer, shutdownCtx)
		_ = shutdownHTTPServer(tableServer, shutdownCtx)
		_ = shutdownHTTPServer(serviceBusServer, shutdownCtx)
		_ = shutdownHTTPServer(blobServer, shutdownCtx)
		_ = shutdownHTTPServer(server, shutdownCtx)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func shutdownHTTPServer(server *http.Server, shutdownCtx context.Context) error {
	if server == nil {
		return nil
	}
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func shutdownDNSServer(server *dns.Server) error {
	if server == nil {
		return nil
	}
	if err := server.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}
