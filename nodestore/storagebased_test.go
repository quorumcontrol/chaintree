package nodestore

import (
	"testing"

	"github.com/quorumcontrol/storage"
)

func TestStorageBased(t *testing.T) {
	sbs := NewStorageBasedStore(storage.NewMemStorage())
	SubtestAll(t, sbs)
}
