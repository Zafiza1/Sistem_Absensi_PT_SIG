// Package faceprofile stores each employee's enrolled face feature vector
// and serves it back to registered tablets so recognition can run
// on-device — the backend never receives or stores a face image, only the
// numeric vector the FaceRecognitionService on the tablet computed, per
// the spec's requirement to treat biometric data as sensitive and avoid
// sending raw images to the server when it can be avoided.
package faceprofile

import (
	"time"

	"github.com/google/uuid"
)

// FaceProfile is deliberately engine-agnostic: FeatureVector holds
// whatever numeric representation the tablet's active
// FaceRecognitionService produces (today: geometric landmark distances;
// tomorrow: a deep-learning embedding), unmarshaled from JSONB without any
// backend-side assumption about its length or meaning.
type FaceProfile struct {
	ID            uuid.UUID
	EmployeeID    uuid.UUID
	FeatureVector []float64
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// Denormalized fields for the tablet sync payload.
	EmployeeName   string
	EmployeeNumber string
}
