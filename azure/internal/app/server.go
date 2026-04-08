package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"tinycloud/internal/api/admin"
	"tinycloud/internal/api/arm"
	"tinycloud/internal/auth"
	"tinycloud/internal/config"
	"tinycloud/internal/httpx"
	"tinycloud/internal/identity"
	"tinycloud/internal/metadata"
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
	if err := s.store.Init(); err != nil {
		return err
	}

	mux := http.NewServeMux()
	authService := auth.NewService(s.cfg)
	admin.NewHandler(s.store, s.cfg.DataRoot).Register(mux)
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
	server := &http.Server{
		Addr:              s.cfg.ManagementAddr(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	blobMux := http.NewServeMux()
	storage.NewHandler(s.store, s.cfg).Register(blobMux)
	blobServer := &http.Server{
		Addr:              s.cfg.ListenHost + ":" + s.cfg.Blob,
		Handler:           chain(blobMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
		ReadHeaderTimeout: 5 * time.Second,
	}

	queueMux := http.NewServeMux()
	queue.NewHandler(s.store, s.cfg).Register(queueMux)
	queueServer := &http.Server{
		Addr:              s.cfg.ListenHost + ":" + s.cfg.Queue,
		Handler:           chain(queueMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
		ReadHeaderTimeout: 5 * time.Second,
	}

	tableMux := http.NewServeMux()
	table.NewHandler(s.store, s.cfg).Register(tableMux)
	tableServer := &http.Server{
		Addr:              s.cfg.ListenHost + ":" + s.cfg.Table,
		Handler:           chain(tableMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
		ReadHeaderTimeout: 5 * time.Second,
	}

	serviceBusMux := http.NewServeMux()
	servicebus.NewHandler(s.store, s.cfg).Register(serviceBusMux)
	serviceBusServer := &http.Server{
		Addr:              s.cfg.ListenHost + ":" + s.cfg.ServiceBus,
		Handler:           chain(serviceBusMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
		ReadHeaderTimeout: 5 * time.Second,
	}

	keyVaultMux := http.NewServeMux()
	keyvault.NewHandler(s.store, s.cfg).Register(keyVaultMux)
	keyVaultServer := &http.Server{
		Addr:              s.cfg.ListenHost + ":" + s.cfg.KeyVault,
		Handler:           chain(keyVaultMux, withRequestID, withLogging(s.logger), withRecovery(s.logger), withCORS, withAzureHeaders),
		ReadHeaderTimeout: 5 * time.Second,
	}

	s.logger.Info("tinycloud server starting", map[string]any{
		"addr":         s.cfg.ManagementAddr(),
		"blobAddr":     s.cfg.ListenHost + ":" + s.cfg.Blob,
		"queueAddr":    s.cfg.ListenHost + ":" + s.cfg.Queue,
		"tableAddr":    s.cfg.ListenHost + ":" + s.cfg.Table,
		"serviceBusAddr": s.cfg.ListenHost + ":" + s.cfg.ServiceBus,
		"keyVaultAddr": s.cfg.ListenHost + ":" + s.cfg.KeyVault,
		"dataRoot":     s.cfg.DataRoot,
	})

	errCh := make(chan error, 6)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	go func() {
		errCh <- blobServer.ListenAndServe()
	}()
	go func() {
		errCh <- queueServer.ListenAndServe()
	}()
	go func() {
		errCh <- tableServer.ListenAndServe()
	}()
	go func() {
		errCh <- serviceBusServer.ListenAndServe()
	}()
	go func() {
		errCh <- keyVaultServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.logger.Info("tinycloud server stopping", nil)
		if err := keyVaultServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		if err := queueServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		if err := tableServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		if err := serviceBusServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		if err := blobServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = keyVaultServer.Shutdown(shutdownCtx)
		_ = queueServer.Shutdown(shutdownCtx)
		_ = tableServer.Shutdown(shutdownCtx)
		_ = serviceBusServer.Shutdown(shutdownCtx)
		_ = blobServer.Shutdown(shutdownCtx)
		_ = server.Shutdown(shutdownCtx)
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
