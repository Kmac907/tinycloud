package storage

import (
	"database/sql"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

type Handler struct {
	store *state.Store
	cfg   config.Config
}

func NewHandler(store *state.Store, cfg config.Config) *Handler {
	return &Handler{store: store, cfg: cfg}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /{account}", h.listContainers)
	mux.HandleFunc("PUT /{account}/{container}", h.createContainer)
	mux.HandleFunc("GET /{account}/{container}", h.listBlobs)
	mux.HandleFunc("PUT /{account}/{container}/{blob...}", h.putBlob)
	mux.HandleFunc("GET /{account}/{container}/{blob...}", h.getBlob)
	mux.HandleFunc("HEAD /{account}/{container}/{blob...}", h.headBlob)
	mux.HandleFunc("DELETE /{account}/{container}/{blob...}", h.deleteBlob)
}

func (h *Handler) listContainers(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("comp") != "list" {
		http.NotFound(w, r)
		return
	}

	containers, err := h.store.ListBlobContainers(r.PathValue("account"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type containerEntry struct {
		Name string `xml:"Name"`
	}
	body := struct {
		XMLName    xml.Name         `xml:"EnumerationResults"`
		ServiceURL string           `xml:"ServiceEndpoint,attr,omitempty"`
		Containers []containerEntry `xml:"Containers>Container"`
	}{
		ServiceURL: h.cfg.BlobURL() + "/" + r.PathValue("account"),
	}
	for _, container := range containers {
		body.Containers = append(body.Containers, containerEntry{Name: container.Name})
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	writeXML(w, http.StatusOK, body)
}

func (h *Handler) createContainer(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("restype") != "container" {
		http.NotFound(w, r)
		return
	}

	container, created, err := h.store.CreateBlobContainer(r.PathValue("account"), r.PathValue("container"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "container already exists", http.StatusConflict)
		return
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	w.Header().Set("ETag", "\""+container.UpdatedAt+"\"")
	w.Header().Set("Last-Modified", formatHTTPTime(container.UpdatedAt))
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) listBlobs(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("restype") != "container" || r.URL.Query().Get("comp") != "list" {
		http.NotFound(w, r)
		return
	}

	blobs, err := h.store.ListBlobs(r.PathValue("account"), r.PathValue("container"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type blobEntry struct {
		Name string `xml:"Name"`
	}
	body := struct {
		XMLName xml.Name    `xml:"EnumerationResults"`
		Blobs   []blobEntry `xml:"Blobs>Blob"`
	}{}
	for _, blob := range blobs {
		body.Blobs = append(body.Blobs, blobEntry{Name: blob.Name})
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	writeXML(w, http.StatusOK, body)
}

func (h *Handler) putBlob(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read request body", http.StatusBadRequest)
		return
	}

	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	blob, err := h.store.PutBlob(
		r.PathValue("account"),
		r.PathValue("container"),
		r.PathValue("blob"),
		contentType,
		body,
	)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "container not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	w.Header().Set("ETag", blob.ETag)
	w.Header().Set("Last-Modified", formatHTTPTime(blob.UpdatedAt))
	w.Header().Set("x-ms-blob-type", "BlockBlob")
	w.Header().Set("x-ms-request-server-encrypted", "true")
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) getBlob(w http.ResponseWriter, r *http.Request) {
	blob, err := h.store.GetBlob(r.PathValue("account"), r.PathValue("container"), r.PathValue("blob"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("ETag", blob.ETag)
	w.Header().Set("Last-Modified", formatHTTPTime(blob.UpdatedAt))
	w.Header().Set("Content-Length", intToString(len(blob.Body)))
	w.Header().Set("x-ms-blob-type", "BlockBlob")
	w.Header().Set("x-ms-request-server-encrypted", "true")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(blob.Body)
}

func (h *Handler) headBlob(w http.ResponseWriter, r *http.Request) {
	blob, err := h.store.GetBlob(r.PathValue("account"), r.PathValue("container"), r.PathValue("blob"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	w.Header().Set("Content-Type", blob.ContentType)
	w.Header().Set("ETag", blob.ETag)
	w.Header().Set("Last-Modified", formatHTTPTime(blob.UpdatedAt))
	w.Header().Set("Content-Length", intToString(len(blob.Body)))
	w.Header().Set("x-ms-blob-type", "BlockBlob")
	w.Header().Set("x-ms-request-server-encrypted", "true")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) deleteBlob(w http.ResponseWriter, r *http.Request) {
	err := h.store.DeleteBlob(r.PathValue("account"), r.PathValue("container"), r.PathValue("blob"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setStorageHeaders(w, r.Header.Get("x-ms-version"))
	w.WriteHeader(http.StatusAccepted)
}

func writeXML(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(body)
}

func setStorageHeaders(w http.ResponseWriter, requestVersion string) {
	version := strings.TrimSpace(requestVersion)
	if version == "" {
		version = "2024-01-01"
	}
	w.Header().Set("x-ms-version", version)
}

func formatHTTPTime(value string) string {
	timestamp, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return ""
	}
	return timestamp.UTC().Format(http.TimeFormat)
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
