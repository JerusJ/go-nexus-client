package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/datadrivers/go-nexus-client/nexus3/pkg/client"
	"github.com/datadrivers/go-nexus-client/nexus3/pkg/tools"
)

type RepositoryService[R any] struct {
	endpoint string
	client   *client.Client
}

func NewRepositoryService[R any](ep string, c *client.Client) *RepositoryService[R] {
	return &RepositoryService[R]{
		endpoint: ep,
		client:   c,
	}
}

func (s *RepositoryService[R]) Create(repo R) error {
	data, err := tools.JsonMarshalInterfaceToIOReader(repo)
	if err != nil {
		return err
	}
	body, resp, err := s.client.Post(s.endpoint, data)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("could not create repository %v: HTTP: %d, %s", repo, resp.StatusCode, string(body))
	}
	return nil
}

func (s *RepositoryService[R]) Get(id string) (*R, error) {
	repo := new(R)

	maxRetries := s.client.RetryMaxAttempts()
	retryWait := time.Duration(s.client.RetryWaitMs()) * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		body, resp, err := s.client.Get(fmt.Sprintf("%s/%s", s.endpoint, id), nil)
		if err != nil {
			// Network error - retry
			lastErr = err
			if attempt < maxRetries-1 {
				time.Sleep(retryWait)
				retryWait *= 2
				continue
			}
			return nil, err
		}

		if resp.StatusCode == http.StatusOK {
			if err = json.Unmarshal(body, repo); err != nil {
				return nil, fmt.Errorf("could not unmarshal repository: %v", err)
			}
			return repo, nil
		}

		// Retry on any non-success status code
		lastErr = fmt.Errorf("could not read repository '%s': HTTP: %d, %s", id, resp.StatusCode, string(body))
		if attempt < maxRetries-1 {
			time.Sleep(retryWait)
			retryWait *= 2
			continue
		}
	}

	return nil, lastErr
}

func (s *RepositoryService[R]) Update(id string, repo R) error {
	maxRetries := s.client.RetryMaxAttempts()
	retryWait := time.Duration(s.client.RetryWaitMs()) * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		data, err := tools.JsonMarshalInterfaceToIOReader(repo)
		if err != nil {
			return err
		}

		body, resp, err := s.client.Put(fmt.Sprintf("%s/%s", s.endpoint, id), data)
		if err != nil {
			// Network error - retry
			lastErr = err
			if attempt < maxRetries-1 {
				time.Sleep(retryWait)
				retryWait *= 2
				continue
			}
			return err
		}

		if resp.StatusCode == http.StatusNoContent {
			return nil
		}

		// Retry on any non-success status code
		lastErr = fmt.Errorf("could not update repository '%s': HTTP: %d, %s", id, resp.StatusCode, string(body))
		if attempt < maxRetries-1 {
			time.Sleep(retryWait)
			retryWait *= 2
			continue
		}
	}

	return lastErr
}

func (s *RepositoryService[R]) Delete(id string) error {
	return DeleteRepository(s.client, id)
}
