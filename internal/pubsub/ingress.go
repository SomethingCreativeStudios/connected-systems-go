package pubsub

import "context"

// Publication is authenticated transport metadata, never client resource JSON.
// A nil IdempotencyKey distinguishes an absent property from an invalid empty one.
type Publication struct {
	Principal       string
	Topic           string
	ProtocolVersion int
	QoS             int
	Retained        bool
	ContentType     string
	IdempotencyKey  *string
	Payload         []byte
}

type CreateResult struct {
	Payload     []byte
	ContentType string
	ResourceURL string
	Replayed    bool
	// Publish emits the lifecycle event and alternate representations. The
	// broker calls it after distributing Payload, so subscribers receive the
	// data before its event. It is nil for replays.
	Publish func()
}

// Ingress separates broker mechanics from resource interpretation and storage.
type Ingress interface {
	Authorize(context.Context, string, string, string) error
	Create(context.Context, Publication) (CreateResult, error)
}

// IngressError carries an operation category without depending on an HTTP handler
// or broker library. Status uses the corresponding CS API HTTP status code.
type IngressError struct {
	Status  int
	Message string
}

func (e *IngressError) Error() string { return e.Message }
