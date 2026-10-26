package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/contractvalidation"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/formaters"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// requestValidator compiles each embedded request schema once for every
// handler and create operation in the process.
var requestValidator = contractvalidation.New()

// resourceCreator is what one Resource Data type means: how a create checks
// its parent, decodes, validates and names it, and how the result is encoded.
// Each is declared beside its HTTP handler; CreateOperations runs them all the
// same way for HTTP and MQTT, and MQTT publishes with the same Codec HTTP GET
// uses.
type resourceCreator struct {
	// Kind is the pubsub capability resource type.
	Kind string
	// Name labels HTTP errors, for example "Datastream not found".
	Name     string
	Contract contractvalidation.Resource
	Codec    resourceCodec
	// ParentField is a read-only association ID in the body. When present it
	// must name the parent given by the URL or topic.
	ParentField string
	// Batch accepts a JSON array of resources in one HTTP request.
	Batch bool
	// Create persists one validated body inside tx and returns its path.
	Create func(ctx context.Context, tx *repository.Repositories, in createInput) (path string, err error)
}

// mediaTypes lists the representations the capability catalog advertises for
// the kind, default first. Creates accept the same media types.
func (c resourceCreator) mediaTypes() []string {
	capability, _ := pubsub.CapabilityForResource(c.Kind)
	return capability.MediaTypes
}

// createInput is one resource body that passed the shared checks.
type createInput struct {
	Parent string
	Media  string
	// Body is the raw JSON for formatter decoders; Value is the same JSON
	// decoded with json.Number.
	Body  []byte
	Value map[string]any
	Links *formaters.LinkResolver
}

// resourceCreators lists every resource clients may create over MQTT.
var resourceCreators = []resourceCreator{systemCreator, datastreamCreator, controlStreamCreator, observationCreator, commandCreator, commandStatusCreator, commandResultCreator, systemEventCreator}

var creatorsByKind = func() map[string]resourceCreator {
	creators, err := indexCreators(resourceCreators)
	if err != nil {
		panic(err)
	}
	return creators
}()

// indexCreators keys creators by kind. It checks that every capability
// offering client creates has exactly one, and that its codec encodes every
// media type the capability advertises.
func indexCreators(creators []resourceCreator) (map[string]resourceCreator, error) {
	byKind := make(map[string]resourceCreator, len(creators))
	for _, c := range creators {
		if _, duplicate := byKind[c.Kind]; duplicate {
			return nil, fmt.Errorf("duplicate creator for %s", c.Kind)
		}
		capability, ok := pubsub.CapabilityForResource(c.Kind)
		if !ok || !capability.ClientCreates {
			return nil, fmt.Errorf("creator %s has no client-create capability", c.Kind)
		}
		if c.Codec == nil {
			return nil, fmt.Errorf("creator %s has no codec", c.Kind)
		}
		for _, media := range capability.MediaTypes {
			if !slices.Contains(c.Codec.ContentTypes(), media) {
				return nil, fmt.Errorf("creator %s cannot encode %s", c.Kind, media)
			}
		}
		byKind[c.Kind] = c
	}
	for _, capability := range pubsub.Capabilities() {
		if _, ok := byKind[capability.ResourceType]; capability.ClientCreates && !ok {
			return nil, fmt.Errorf("capability %s offers client creates but has no creator", capability.Name)
		}
	}
	return byKind, nil
}

// CreateOperations runs resource creates for HTTP and for the broker's
// pre-distribution ingress. The resource write, system history, validation and
// canonical serialization share one transaction.
type CreateOperations struct {
	cfg       *config.Config
	repos     *repository.Repositories
	validator *contractvalidation.Validator
	logger    *zap.Logger
	publisher *pubsub.Publisher
}

func NewCreateOperations(cfg *config.Config, repos *repository.Repositories, publisher *pubsub.Publisher, logger *zap.Logger) *CreateOperations {
	return &CreateOperations{cfg: cfg, repos: repos, validator: requestValidator, logger: logger, publisher: publisher}
}

type createdResource struct {
	Path   string
	Change pubsub.Change
}

// errRepresentationTooLarge rejects a resource whose canonical representation
// cannot be published within the advertised MQTT message limit.
var errRepresentationTooLarge = errors.New("serialized resource exceeds the MQTT message limit")

func invalid(err error) error { return &repository.ValidationError{Err: err} }

// decodeError keeps a missing referenced resource a 404 and makes any other
// decoding failure a validation error.
func decodeError(err error) error {
	if repository.IsNotFound(err) {
		return err
	}
	return invalid(err)
}

// parentError reports a missing parent the way nested repository creates do.
func parentError(kind, id string, err error) error {
	if repository.IsNotFound(err) {
		return fmt.Errorf("%w: %s %q", repository.ErrParentNotFound, kind, id)
	}
	return err
}

func objectBody(body []byte) (map[string]any, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if value == nil {
		return nil, fmt.Errorf("payload must be one resource object")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("payload must contain exactly one JSON resource")
	}
	return value, nil
}

// createOne validates and persists one decoded body inside tx. With MQTT
// enabled it also resolves the committed resource's canonical change.
func (s *CreateOperations) createOne(ctx context.Context, tx *repository.Repositories, c resourceCreator, parent, media string, body []byte, value map[string]any) (createdResource, error) {
	var out createdResource
	if !slices.Contains(c.mediaTypes(), media) {
		return out, invalid(fmt.Errorf("unsupported content type %q", media))
	}
	if c.ParentField != "" {
		if supplied, ok := value[c.ParentField]; ok && supplied != parent {
			return out, invalid(fmt.Errorf("%s does not match the parent resource %q", c.ParentField, parent))
		}
	}
	if err := s.validator.Validate(c.Contract, media, body); err != nil {
		return out, invalid(err)
	}
	links := newLinkResolver(s.cfg, tx)
	path, err := c.Create(ctx, tx, createInput{Parent: parent, Media: media, Body: body, Value: value, Links: links})
	if err != nil {
		return out, err
	}
	out.Path = path
	if !s.cfg.MQTT.Enabled {
		return out, nil
	}
	// Re-read through the same canonical resolver used for HTTP changes, inside
	// the transaction, so ownership and all advertised encodings are checked.
	resolver := pubsub.NewRepositoryChangeResolver(s.cfg.API.BaseURL, tx, representer(links))
	out.Change, err = resolver.Resolve(ctx, path, pubsub.OperationCreate)
	if err != nil {
		return out, err
	}
	limit := s.cfg.MQTT.MaxMessageBytes
	if limit <= 0 {
		limit = 10 << 20
	}
	for _, r := range out.Change.Representations {
		if int64(len(r.Payload)) > limit {
			return out, errRepresentationTooLarge
		}
	}
	return out, nil
}

// HTTP creates the resource, or for a batch creator the array of resources,
// in the request body under parent.
func (s *CreateOperations) HTTP(w http.ResponseWriter, r *http.Request, c resourceCreator, parent string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeDeserializeError(w, r, err)
		return
	}
	media := mediatype.Normalize(r.Header.Get("Content-Type"))
	if media == "" || media == mediatype.JSON && !slices.Contains(c.mediaTypes(), media) {
		media = c.mediaTypes()[0]
	}
	bodies := [][]byte{body}
	if c.Batch && bytes.HasPrefix(bytes.TrimSpace(body), []byte("[")) {
		var values []json.RawMessage
		if err = json.Unmarshal(body, &values); err != nil || len(values) == 0 {
			writeValidationError(w, r, errors.New("at least one valid resource is required"))
			return
		}
		bodies = nil
		for _, value := range values {
			bodies = append(bodies, value)
		}
	}
	var created []createdResource
	err = s.repos.Transaction(r.Context(), func(tx *repository.Repositories) error {
		for _, body := range bodies {
			value, err := objectBody(body)
			if err != nil {
				return invalid(err)
			}
			resource, err := s.createOne(r.Context(), tx, c, parent, media, body, value)
			if err != nil {
				return err
			}
			created = append(created, resource)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errRepresentationTooLarge) {
			writeError(w, r, http.StatusRequestEntityTooLarge, err.Error())
			return
		}
		writeRepositoryError(w, r, s.logger, err, c.Name, "Failed to create "+strings.ToLower(c.Name))
		return
	}
	w.Header().Set("Location", strings.TrimRight(s.cfg.API.BaseURL, "/")+created[0].Path)
	w.WriteHeader(http.StatusCreated)
	// The changes were resolved inside the transaction, so publish them here and
	// tell the event middleware not to resolve the request again.
	if s.cfg.MQTT.Enabled && s.publisher != nil {
		for _, resource := range created {
			s.publisher.PublishChange(resource.Change)
		}
		pubsub.MarkPublished(r.Context())
	}
}
