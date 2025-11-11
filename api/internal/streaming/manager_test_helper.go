//go:build integration
// +build integration

package streaming

import (
	"context"

	"github.com/stwalsh4118/hermes/internal/models"
)

// TriggerFirstBatchForTest triggers the first batch generation for testing purposes.
// This is a test helper that allows integration tests to manually trigger the first batch
// since the batch coordinator doesn't automatically handle nil batch cases.
func TriggerFirstBatchForTest(manager *StreamManager, session *models.StreamSession) error {
	return manager.generateNextBatch(context.Background(), session)
}
