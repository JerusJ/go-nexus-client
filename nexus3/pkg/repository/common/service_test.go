package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	"github.com/stretchr/testify/assert"
)

// TestRepository is a simple struct for testing
type TestRepository struct {
	Name   string `json:"name"`
	Online bool   `json:"online"`
}

func newTestClient(url string, retryMaxAttempts, retryWaitMs *int) *client.Client {
	return client.NewClient(client.Config{
		URL:              url,
		Username:         "admin",
		Password:         "admin123",
		Insecure:         true,
		RetryMaxAttempts: retryMaxAttempts,
		RetryWaitMs:      retryWaitMs,
	})
}

func intPtr(i int) *int {
	return &i
}

func TestGet_SuccessOnFirstAttempt(t *testing.T) {
	repo := TestRepository{Name: "test-repo", Online: true}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(repo)
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(3), intPtr(10))
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	result, err := service.Get("test-repo")

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-repo", result.Name)
	assert.Equal(t, true, result.Online)
}

func TestGet_RetriesOnError_ThenSucceeds(t *testing.T) {
	repo := TestRepository{Name: "test-repo", Online: true}
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt < 3 {
			// First 2 attempts return 404
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Repository not found"}`))
			return
		}
		// Third attempt succeeds
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(repo)
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(5), intPtr(10)) // 10ms wait for fast tests
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	result, err := service.Get("test-repo")

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test-repo", result.Name)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "Should have made 3 attempts")
}

func TestGet_FailsAfterMaxRetries(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Repository not found"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(3), intPtr(10))
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	result, err := service.Get("test-repo")

	assert.NotNil(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "could not read repository")
	assert.Contains(t, err.Error(), "404")
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "Should have made 3 attempts")
}

func TestGet_RetriesOn500Error(t *testing.T) {
	repo := TestRepository{Name: "test-repo", Online: true}
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message": "Internal server error"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(repo)
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(3), intPtr(10))
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	result, err := service.Get("test-repo")

	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(2), atomic.LoadInt32(&attemptCount), "Should have made 2 attempts")
}

func TestUpdate_SuccessOnFirstAttempt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(3), intPtr(10))
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	repo := TestRepository{Name: "test-repo", Online: true}
	err := service.Update("test-repo", repo)

	assert.Nil(t, err)
}

func TestUpdate_RetriesOnError_ThenSucceeds(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		if attempt < 3 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Repository not found"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(5), intPtr(10))
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	repo := TestRepository{Name: "test-repo", Online: true}
	err := service.Update("test-repo", repo)

	assert.Nil(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "Should have made 3 attempts")
}

func TestUpdate_FailsAfterMaxRetries(t *testing.T) {
	var attemptCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"message": "Service unavailable"}`))
	}))
	defer server.Close()

	c := newTestClient(server.URL, intPtr(3), intPtr(10))
	service := NewRepositoryService[TestRepository]("service/rest/v1/repositories/test", c)

	repo := TestRepository{Name: "test-repo", Online: true}
	err := service.Update("test-repo", repo)

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "could not update repository")
	assert.Contains(t, err.Error(), "503")
	assert.Equal(t, int32(3), atomic.LoadInt32(&attemptCount), "Should have made 3 attempts")
}

func TestRetryConfig_Defaults(t *testing.T) {
	c := client.NewClient(client.Config{
		URL:      "http://localhost:8081",
		Username: "admin",
		Password: "admin123",
		// RetryMaxAttempts and RetryWaitMs not set - should use defaults
	})

	assert.Equal(t, 5, c.RetryMaxAttempts(), "Default RetryMaxAttempts should be 5")
	assert.Equal(t, 500, c.RetryWaitMs(), "Default RetryWaitMs should be 500")
}

func TestRetryConfig_CustomValues(t *testing.T) {
	c := client.NewClient(client.Config{
		URL:              "http://localhost:8081",
		Username:         "admin",
		Password:         "admin123",
		RetryMaxAttempts: intPtr(10),
		RetryWaitMs:      intPtr(1000),
	})

	assert.Equal(t, 10, c.RetryMaxAttempts(), "Custom RetryMaxAttempts should be 10")
	assert.Equal(t, 1000, c.RetryWaitMs(), "Custom RetryWaitMs should be 1000")
}
