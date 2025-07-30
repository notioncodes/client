package client

import (
	"sync"
	"time"

	"github.com/notioncodes/types"
)

// ExportJobResult contains the results of an export operation.
type ExportJobResult struct {
	Start    time.Time                `json:"start"`
	End      time.Time                `json:"end"`
	Success  map[types.ObjectType]int `json:"success"`
	Requests int                      `json:"requests"`
	Errors   []ExportError            `json:"errors,omitempty"`
	mu       sync.RWMutex
}

// ExportError represents an error that occurred during export.
type ExportError struct {
	ObjectType types.ObjectType `json:"object_type"`
	ObjectID   string           `json:"object_id"`
	Error      string           `json:"error"`
	Timestamp  time.Time        `json:"timestamp"`
}

// NewExportResult creates a new ExportResult.
func NewExportResult() *ExportJobResult {
	return &ExportJobResult{
		Start:   time.Now(),
		Success: make(map[types.ObjectType]int),
		Errors:  []ExportError{},
	}
}

// Errored adds an error to the export result.
//
// Arguments:
// - objectType: The type of object that errored.
// - objectID: The ID of the object that errored.
// - err: The error that occurred.
func (e *ExportJobResult) Errored(objectType types.ObjectType, objectID string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Errors = append(e.Errors, ExportError{
		ObjectType: objectType,
		ObjectID:   objectID,
		Error:      err.Error(),
		Timestamp:  time.Now(),
	})
}

// Successful adds a successful export to the export result.
//
// Arguments:
// - objectType: The type of object that was exported.
// - r: The export result for the object.
func (e *ExportJobResult) Successful(objectType types.ObjectType, r *ExportJobResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.Success[objectType] += r.Success[objectType]
	e.Errors = append(e.Errors, r.Errors...)
}

// Total returns the total number of objects exported.
//
// Returns:
// - The total number of objects exported.
func (e *ExportJobResult) Total() int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var total int
	for _, count := range e.Success {
		total += count
	}

	return total + len(e.Errors)
}
