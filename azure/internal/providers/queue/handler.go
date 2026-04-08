package queue

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
	mux.HandleFunc("GET /{account}", h.listQueues)
	mux.HandleFunc("PUT /{account}/{queue}", h.createQueue)
	mux.HandleFunc("POST /{account}/{queue}/messages", h.putMessage)
	mux.HandleFunc("GET /{account}/{queue}/messages", h.getMessages)
	mux.HandleFunc("DELETE /{account}/{queue}/messages/{messageId}", h.deleteMessage)
}

func (h *Handler) listQueues(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("comp") != "list" {
		http.NotFound(w, r)
		return
	}
	if !h.accountExists(w, r, r.PathValue("account")) {
		return
	}

	queues, err := h.store.ListQueues(r.PathValue("account"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type queueEntry struct {
		Name string `xml:"Name"`
	}
	body := struct {
		XMLName    xml.Name     `xml:"EnumerationResults"`
		ServiceURL string       `xml:"ServiceEndpoint,attr,omitempty"`
		Queues     []queueEntry `xml:"Queues>Queue"`
	}{
		ServiceURL: h.cfg.QueueURL() + "/" + r.PathValue("account"),
	}
	for _, queue := range queues {
		body.Queues = append(body.Queues, queueEntry{Name: queue.Name})
	}

	setQueueHeaders(w, r.Header.Get("x-ms-version"))
	writeXML(w, http.StatusOK, body)
}

func (h *Handler) createQueue(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("restype") != "queue" {
		http.NotFound(w, r)
		return
	}

	queue, created, err := h.store.CreateQueue(r.PathValue("account"), r.PathValue("queue"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "queue already exists", http.StatusConflict)
		return
	}

	setQueueHeaders(w, r.Header.Get("x-ms-version"))
	w.Header().Set("ETag", "\""+queue.UpdatedAt+"\"")
	w.Header().Set("Last-Modified", formatHTTPTime(queue.UpdatedAt))
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) putMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		XMLName     xml.Name `xml:"QueueMessage"`
		MessageText string   `xml:"MessageText"`
	}
	if err := xml.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid XML", http.StatusBadRequest)
		return
	}

	message, err := h.store.EnqueueMessage(r.PathValue("account"), r.PathValue("queue"), body.MessageText)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := struct {
		XMLName         xml.Name `xml:"QueueMessagesList"`
		MessageID       string   `xml:"QueueMessage>MessageId"`
		InsertionTime   string   `xml:"QueueMessage>InsertionTime,omitempty"`
		ExpirationTime  string   `xml:"QueueMessage>ExpirationTime,omitempty"`
		PopReceipt      string   `xml:"QueueMessage>PopReceipt"`
		TimeNextVisible string   `xml:"QueueMessage>TimeNextVisible,omitempty"`
	}{
		MessageID:       message.ID,
		InsertionTime:   formatHTTPTime(message.CreatedAt),
		ExpirationTime:  "",
		PopReceipt:      message.PopReceipt,
		TimeNextVisible: formatHTTPTime(message.VisibleAt),
	}

	setQueueHeaders(w, r.Header.Get("x-ms-version"))
	writeXML(w, http.StatusCreated, response)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	maxMessages := 1
	if raw := strings.TrimSpace(r.URL.Query().Get("numofmessages")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			http.Error(w, "numofmessages must be a positive integer", http.StatusBadRequest)
			return
		}
		maxMessages = value
	}

	visibilityTimeout := 30 * time.Second
	if raw := strings.TrimSpace(r.URL.Query().Get("visibilitytimeout")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			http.Error(w, "visibilitytimeout must be a non-negative integer", http.StatusBadRequest)
			return
		}
		visibilityTimeout = time.Duration(value) * time.Second
	}

	messages, err := h.store.ReceiveMessages(r.PathValue("account"), r.PathValue("queue"), maxMessages, visibilityTimeout)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type queueMessageXML struct {
		MessageID       string `xml:"MessageId"`
		InsertionTime   string `xml:"InsertionTime,omitempty"`
		ExpirationTime  string `xml:"ExpirationTime,omitempty"`
		PopReceipt      string `xml:"PopReceipt"`
		TimeNextVisible string `xml:"TimeNextVisible,omitempty"`
		DequeueCount    int    `xml:"DequeueCount"`
		MessageText     string `xml:"MessageText"`
	}
	body := struct {
		XMLName  xml.Name          `xml:"QueueMessagesList"`
		Messages []queueMessageXML `xml:"QueueMessage"`
	}{}
	for _, message := range messages {
		body.Messages = append(body.Messages, queueMessageXML{
			MessageID:       message.ID,
			InsertionTime:   formatHTTPTime(message.CreatedAt),
			ExpirationTime:  "",
			PopReceipt:      message.PopReceipt,
			TimeNextVisible: formatHTTPTime(message.VisibleAt),
			DequeueCount:    message.DequeueCount,
			MessageText:     message.MessageText,
		})
	}

	setQueueHeaders(w, r.Header.Get("x-ms-version"))
	writeXML(w, http.StatusOK, body)
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	popReceipt := strings.TrimSpace(r.URL.Query().Get("popreceipt"))
	if popReceipt == "" {
		http.Error(w, "popreceipt is required", http.StatusBadRequest)
		return
	}

	err := h.store.DeleteMessage(r.PathValue("account"), r.PathValue("queue"), r.PathValue("messageId"), popReceipt)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	setQueueHeaders(w, r.Header.Get("x-ms-version"))
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) accountExists(w http.ResponseWriter, r *http.Request, accountName string) bool {
	if _, err := h.store.GetStorageAccountByName(accountName); errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return false
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	return true
}

func writeXML(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, xml.Header)
	_ = xml.NewEncoder(w).Encode(body)
}

func setQueueHeaders(w http.ResponseWriter, requestVersion string) {
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
