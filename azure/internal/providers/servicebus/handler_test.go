package servicebus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tinycloud/internal/config"
	"tinycloud/internal/state"
)

func TestHandlerRoundTrip(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	cfg := config.FromEnv()
	NewHandler(store, cfg).Register(mux)

	createNamespaceReq := httptest.NewRequest(http.MethodPost, "/namespaces", strings.NewReader(`{"name":"local-messaging"}`))
	createNamespaceRec := httptest.NewRecorder()
	mux.ServeHTTP(createNamespaceRec, createNamespaceReq)
	if createNamespaceRec.Code != http.StatusCreated {
		t.Fatalf("create namespace status = %d, want %d", createNamespaceRec.Code, http.StatusCreated)
	}
	if !strings.Contains(createNamespaceRec.Body.String(), cfg.ServiceBusURL()+"/namespaces/local-messaging") {
		t.Fatalf("create namespace body = %q, want namespace id", createNamespaceRec.Body.String())
	}

	listNamespacesReq := httptest.NewRequest(http.MethodGet, "/namespaces", nil)
	listNamespacesRec := httptest.NewRecorder()
	mux.ServeHTTP(listNamespacesRec, listNamespacesReq)
	if listNamespacesRec.Code != http.StatusOK {
		t.Fatalf("list namespaces status = %d, want %d", listNamespacesRec.Code, http.StatusOK)
	}

	createQueueReq := httptest.NewRequest(http.MethodPost, "/namespaces/local-messaging/queues", strings.NewReader(`{"name":"jobs"}`))
	createQueueRec := httptest.NewRecorder()
	mux.ServeHTTP(createQueueRec, createQueueReq)
	if createQueueRec.Code != http.StatusCreated {
		t.Fatalf("create queue status = %d, want %d", createQueueRec.Code, http.StatusCreated)
	}

	listQueuesReq := httptest.NewRequest(http.MethodGet, "/namespaces/local-messaging/queues", nil)
	listQueuesRec := httptest.NewRecorder()
	mux.ServeHTTP(listQueuesRec, listQueuesReq)
	if listQueuesRec.Code != http.StatusOK {
		t.Fatalf("list queues status = %d, want %d", listQueuesRec.Code, http.StatusOK)
	}
	if !strings.Contains(listQueuesRec.Body.String(), `"name":"jobs"`) {
		t.Fatalf("list queues body = %q, want queue name", listQueuesRec.Body.String())
	}

	sendReq := httptest.NewRequest(http.MethodPost, "/namespaces/local-messaging/queues/jobs/messages", strings.NewReader(`{"body":"{\"job\":\"sync\"}"}`))
	sendRec := httptest.NewRecorder()
	mux.ServeHTTP(sendRec, sendReq)
	if sendRec.Code != http.StatusCreated {
		t.Fatalf("send status = %d, want %d", sendRec.Code, http.StatusCreated)
	}

	receiveReq := httptest.NewRequest(http.MethodPost, "/namespaces/local-messaging/queues/jobs/messages/receive?maxMessages=1&visibilityTimeout=30", nil)
	receiveRec := httptest.NewRecorder()
	mux.ServeHTTP(receiveRec, receiveReq)
	if receiveRec.Code != http.StatusOK {
		t.Fatalf("receive status = %d, want %d", receiveRec.Code, http.StatusOK)
	}
	if !strings.Contains(receiveRec.Body.String(), `"body":"{\"job\":\"sync\"}"`) {
		t.Fatalf("receive body = %q, want message body", receiveRec.Body.String())
	}
	messageID := extractJSONValue(receiveRec.Body.String(), `"id":"`)
	lockToken := extractJSONValue(receiveRec.Body.String(), `"lockToken":"`)
	if messageID == "" || lockToken == "" {
		t.Fatalf("receive body = %q, want message id and lock token", receiveRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/namespaces/local-messaging/queues/jobs/messages/"+messageID+"?lockToken="+lockToken, nil)
	deleteRec := httptest.NewRecorder()
	mux.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteRec.Code, http.StatusNoContent)
	}
}

func TestCreateQueueRequiresExistingNamespace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store, err := state.NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mux := http.NewServeMux()
	NewHandler(store, config.FromEnv()).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/namespaces/missing/queues", strings.NewReader(`{"name":"jobs"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func extractJSONValue(body, prefix string) string {
	startIndex := strings.Index(body, prefix)
	if startIndex == -1 {
		return ""
	}
	startIndex += len(prefix)
	endIndex := strings.Index(body[startIndex:], `"`)
	if endIndex == -1 {
		return ""
	}
	return body[startIndex : startIndex+endIndex]
}
