package servicebus

import (
	"database/sql"
	"encoding/json"
	"errors"
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
	mux.HandleFunc("GET /namespaces", h.listNamespaces)
	mux.HandleFunc("POST /namespaces", h.createNamespace)
	mux.HandleFunc("GET /namespaces/{namespace}/queues", h.listQueues)
	mux.HandleFunc("POST /namespaces/{namespace}/queues", h.createQueue)
	mux.HandleFunc("GET /namespaces/{namespace}/topics", h.listTopics)
	mux.HandleFunc("POST /namespaces/{namespace}/topics", h.createTopic)
	mux.HandleFunc("GET /namespaces/{namespace}/topics/{topic}/subscriptions", h.listSubscriptions)
	mux.HandleFunc("POST /namespaces/{namespace}/topics/{topic}/subscriptions", h.createSubscription)
	mux.HandleFunc("POST /namespaces/{namespace}/queues/{queue}/messages", h.sendMessage)
	mux.HandleFunc("POST /namespaces/{namespace}/queues/{queue}/messages/receive", h.receiveMessages)
	mux.HandleFunc("DELETE /namespaces/{namespace}/queues/{queue}/messages/{messageId}", h.deleteMessage)
	mux.HandleFunc("POST /namespaces/{namespace}/topics/{topic}/messages", h.publishTopicMessage)
	mux.HandleFunc("POST /namespaces/{namespace}/topics/{topic}/subscriptions/{subscription}/messages/receive", h.receiveSubscriptionMessages)
	mux.HandleFunc("DELETE /namespaces/{namespace}/topics/{topic}/subscriptions/{subscription}/messages/{messageId}", h.deleteSubscriptionMessage)
}

func (h *Handler) listNamespaces(w http.ResponseWriter, _ *http.Request) {
	namespaces, err := h.store.ListServiceBusNamespaces()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(namespaces))
	for _, namespace := range namespaces {
		value = append(value, map[string]any{
			"name": namespace.Name,
			"id":   h.namespaceID(namespace.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createNamespace(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	namespace, created, err := h.store.CreateServiceBusNamespace(body.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "namespace already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": namespace.Name,
		"id":   h.namespaceID(namespace.Name),
	})
}

func (h *Handler) listQueues(w http.ResponseWriter, r *http.Request) {
	queues, err := h.store.ListServiceBusQueues(r.PathValue("namespace"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(queues))
	for _, queue := range queues {
		value = append(value, map[string]any{
			"name": queue.Name,
			"id":   h.queueID(queue.Namespace, queue.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createQueue(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	queue, created, err := h.store.CreateServiceBusQueue(r.PathValue("namespace"), body.Name)
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

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": queue.Name,
		"id":   h.queueID(queue.Namespace, queue.Name),
	})
}

func (h *Handler) listTopics(w http.ResponseWriter, r *http.Request) {
	topics, err := h.store.ListServiceBusTopics(r.PathValue("namespace"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(topics))
	for _, topic := range topics {
		value = append(value, map[string]any{
			"name": topic.Name,
			"id":   h.topicID(topic.Namespace, topic.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createTopic(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	topic, created, err := h.store.CreateServiceBusTopic(r.PathValue("namespace"), body.Name)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "topic already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": topic.Name,
		"id":   h.topicID(topic.Namespace, topic.Name),
	})
}

func (h *Handler) listSubscriptions(w http.ResponseWriter, r *http.Request) {
	subscriptions, err := h.store.ListServiceBusSubscriptions(r.PathValue("namespace"), r.PathValue("topic"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		value = append(value, map[string]any{
			"name": subscription.Name,
			"id":   h.subscriptionID(subscription.Namespace, subscription.TopicName, subscription.Name),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) createSubscription(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	subscription, created, err := h.store.CreateServiceBusSubscription(r.PathValue("namespace"), r.PathValue("topic"), body.Name)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !created {
		http.Error(w, "subscription already exists", http.StatusConflict)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"name": subscription.Name,
		"id":   h.subscriptionID(subscription.Namespace, subscription.TopicName, subscription.Name),
	})
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	message, err := h.store.SendServiceBusMessage(r.PathValue("namespace"), r.PathValue("queue"), body.Body)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, messageResponse(message))
}

func (h *Handler) receiveMessages(w http.ResponseWriter, r *http.Request) {
	maxMessages, visibilityTimeout, ok := receiveOptions(w, r)
	if !ok {
		return
	}

	messages, err := h.store.ReceiveServiceBusMessages(r.PathValue("namespace"), r.PathValue("queue"), maxMessages, visibilityTimeout)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		value = append(value, messageResponse(message))
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) publishTopicMessage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "request body must be valid JSON", http.StatusBadRequest)
		return
	}

	message, err := h.store.PublishServiceBusTopicMessage(r.PathValue("namespace"), r.PathValue("topic"), body.Body)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, subscriptionMessageResponse(message))
}

func (h *Handler) receiveSubscriptionMessages(w http.ResponseWriter, r *http.Request) {
	maxMessages, visibilityTimeout, ok := receiveOptions(w, r)
	if !ok {
		return
	}

	messages, err := h.store.ReceiveServiceBusSubscriptionMessages(r.PathValue("namespace"), r.PathValue("topic"), r.PathValue("subscription"), maxMessages, visibilityTimeout)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	value := make([]map[string]any, 0, len(messages))
	for _, message := range messages {
		value = append(value, subscriptionMessageResponse(message))
	}
	writeJSON(w, http.StatusOK, map[string]any{"value": value})
}

func (h *Handler) deleteSubscriptionMessage(w http.ResponseWriter, r *http.Request) {
	lockToken := strings.TrimSpace(r.URL.Query().Get("lockToken"))
	if lockToken == "" {
		http.Error(w, "lockToken is required", http.StatusBadRequest)
		return
	}

	err := h.store.DeleteServiceBusSubscriptionMessage(r.PathValue("namespace"), r.PathValue("topic"), r.PathValue("subscription"), r.PathValue("messageId"), lockToken)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) deleteMessage(w http.ResponseWriter, r *http.Request) {
	lockToken := strings.TrimSpace(r.URL.Query().Get("lockToken"))
	if lockToken == "" {
		http.Error(w, "lockToken is required", http.StatusBadRequest)
		return
	}

	err := h.store.DeleteServiceBusMessage(r.PathValue("namespace"), r.PathValue("queue"), r.PathValue("messageId"), lockToken)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) namespaceID(namespace string) string {
	return h.cfg.ServiceBusURL() + "/namespaces/" + namespace
}

func (h *Handler) queueID(namespace, queue string) string {
	return h.namespaceID(namespace) + "/queues/" + queue
}

func (h *Handler) topicID(namespace, topic string) string {
	return h.namespaceID(namespace) + "/topics/" + topic
}

func (h *Handler) subscriptionID(namespace, topic, subscription string) string {
	return h.topicID(namespace, topic) + "/subscriptions/" + subscription
}

func messageResponse(message state.ServiceBusMessage) map[string]any {
	return map[string]any{
		"id":            message.ID,
		"body":          message.Body,
		"lockToken":     message.LockToken,
		"deliveryCount": message.DeliveryCount,
	}
}

func subscriptionMessageResponse(message state.ServiceBusSubscriptionMessage) map[string]any {
	return map[string]any{
		"id":            message.ID,
		"body":          message.Body,
		"lockToken":     message.LockToken,
		"deliveryCount": message.DeliveryCount,
	}
}

func receiveOptions(w http.ResponseWriter, r *http.Request) (int, time.Duration, bool) {
	maxMessages := 1
	if raw := strings.TrimSpace(r.URL.Query().Get("maxMessages")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			http.Error(w, "maxMessages must be a positive integer", http.StatusBadRequest)
			return 0, 0, false
		}
		maxMessages = value
	}

	visibilityTimeout := 30 * time.Second
	if raw := strings.TrimSpace(r.URL.Query().Get("visibilityTimeout")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			http.Error(w, "visibilityTimeout must be a non-negative integer", http.StatusBadRequest)
			return 0, 0, false
		}
		visibilityTimeout = time.Duration(value) * time.Second
	}

	return maxMessages, visibilityTimeout, true
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
