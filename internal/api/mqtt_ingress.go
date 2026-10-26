package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/config"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/pubsub"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/repository"
	"go.uber.org/zap"
)

// NewMQTTIngress supplies direct operations to the embedded broker.
func NewMQTTIngress(cfg *config.Config, repos *repository.Repositories, publisher *pubsub.Publisher, logger *zap.Logger) pubsub.Ingress {
	return NewCreateOperations(cfg, repos, publisher, logger)
}

func (s *CreateOperations) Authorize(_ context.Context, _ string, access, topic string) error {
	allowed := false
	switch access {
	case "subscribe":
		allowed = pubsub.FilterOffered(s.cfg.MQTT.NodeID, topic, s.cfg.PubSub)
	case "read":
		allowed = pubsub.TopicOffered(s.cfg.MQTT.NodeID, topic, s.cfg.PubSub)
	case "write":
		target, err := pubsub.ParseDataTopic(s.cfg.MQTT.NodeID, topic)
		allowed = s.cfg.PubSub.ResourceData.Enabled && err == nil && target.Capability.ClientCreates
	}
	if !allowed {
		return &pubsub.IngressError{Status: http.StatusForbidden, Message: "topic or filter is not authorized by the advertised capabilities"}
	}
	return nil
}

// requestFingerprint identifies a create's input: its canonical collection,
// media type and JSON with sorted keys and number lexemes preserved.
func requestFingerprint(collection, media string, value map[string]any) (string, error) {
	normalized, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal([]any{collection, media, json.RawMessage(normalized)})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

func validatePublication(version, qos int, retained bool, media string, key string) error {
	if version != 4 && version != 5 {
		return fmt.Errorf("only MQTT 3.1.1 and 5.0 are supported")
	}
	if qos < 0 || qos > 2 || retained {
		return fmt.Errorf("invalid QoS or retained resource publication")
	}
	if version == 4 && key != "" {
		return fmt.Errorf("idempotency keys require MQTT 5")
	}
	if key != "" && (!utf8.ValidString(key) || len(key) > 128 || strings.ContainsRune(key, 0)) {
		return fmt.Errorf("idempotency-key must contain 1 to 128 UTF-8 bytes without NUL")
	}
	if qos == 2 && (version != 5 || key == "") {
		return fmt.Errorf("QoS 2 creates require MQTT 5 and an idempotency-key")
	}
	if version == 5 && media == "" {
		return fmt.Errorf("MQTT 5 Resource Data requires Content Type")
	}
	return nil
}

// ingressError gives the broker the CS API status category of a failed create.
// Server failures are not described.
func ingressError(err error) *pubsub.IngressError {
	var failure *pubsub.IngressError
	var validation *repository.ValidationError
	switch {
	case errors.As(err, &failure):
		return failure
	case errors.Is(err, errRepresentationTooLarge):
		return &pubsub.IngressError{Status: http.StatusRequestEntityTooLarge, Message: err.Error()}
	case errors.As(err, &validation):
		return &pubsub.IngressError{Status: http.StatusBadRequest, Message: validation.Error()}
	case errors.Is(err, repository.ErrParentNotFound):
		return &pubsub.IngressError{Status: http.StatusNotFound, Message: err.Error()}
	case repository.IsNotFound(err):
		return &pubsub.IngressError{Status: http.StatusNotFound, Message: "parent resource not found"}
	case repository.IsUniqueViolation(err):
		return &pubsub.IngressError{Status: http.StatusConflict, Message: "a resource with the same unique identifier already exists"}
	}
	return &pubsub.IngressError{Status: http.StatusInternalServerError, Message: "resource operation failed"}
}

// Create runs one client publication as a CS API create. The broker applies
// the ingest timeout and publishes the result's change after distributing it.
func (s *CreateOperations) Create(ctx context.Context, publication pubsub.Publication) (pubsub.CreateResult, error) {
	result, err := s.ingest(repository.WithLinkBaseURL(ctx, s.cfg.API.BaseURL), publication)
	if err != nil {
		s.logger.Warn("Rejected MQTT create", zap.String("topic", publication.Topic), zap.String("principal", publication.Principal), zap.Error(err))
		return result, ingressError(err)
	}
	return result, nil
}

func (s *CreateOperations) ingest(ctx context.Context, publication pubsub.Publication) (result pubsub.CreateResult, err error) {
	principal := publication.Principal
	if principal == "" {
		return result, &pubsub.IngressError{Status: http.StatusForbidden, Message: "missing authenticated principal"}
	}
	key := ""
	if publication.IdempotencyKey != nil {
		key = *publication.IdempotencyKey
		if key == "" {
			return result, &pubsub.IngressError{Status: http.StatusBadRequest, Message: "idempotency-key must not be empty"}
		}
	}
	version, qos := publication.ProtocolVersion, publication.QoS
	media := mediatype.Normalize(publication.ContentType)
	if err = validatePublication(version, qos, publication.Retained, media, key); err != nil {
		return result, invalid(err)
	}
	if !s.cfg.PubSub.ResourceData.Enabled {
		return result, &pubsub.IngressError{Status: http.StatusForbidden, Message: "Resource Data is disabled"}
	}
	target, err := pubsub.ParseDataTopic(s.cfg.MQTT.NodeID, publication.Topic)
	if err != nil {
		return result, invalid(err)
	}
	creator, ok := creatorsByKind[target.Capability.ResourceType]
	if !target.Capability.ClientCreates || !ok {
		return result, &pubsub.IngressError{Status: http.StatusForbidden, Message: "client creates are disabled for this capability"}
	}
	if version == 4 {
		media = target.MediaType
	}
	if media != target.MediaType {
		return result, invalid(fmt.Errorf("content type %q conflicts with topic representation %q", media, target.MediaType))
	}
	limit := s.cfg.MQTT.MaxMessageBytes
	if limit <= 0 {
		limit = 10 << 20
	}
	body := publication.Payload
	if int64(len(body)) > limit {
		return result, &pubsub.IngressError{Status: http.StatusRequestEntityTooLarge, Message: "resource exceeds message limit"}
	}
	value, err := objectBody(body)
	if err != nil {
		return result, invalid(err)
	}
	apiRoot := strings.TrimRight(s.cfg.API.BaseURL, "/")
	fingerprint, err := requestFingerprint(apiRoot+target.CanonicalCollection, media, value)
	if err != nil {
		return result, invalid(err)
	}
	var created createdResource
	var response []byte
	replayed := false
	err = s.repos.Transaction(ctx, func(tx *repository.Repositories) error {
		resolver := pubsub.NewRepositoryChangeResolver(s.cfg.API.BaseURL, tx, nil)
		if target.ParentID != "" {
			parent, err := resolver.ResolveIdentity(ctx, "/"+target.Capability.ParentCollection+"/"+url.PathEscape(target.ParentID))
			if err != nil {
				return err
			}
			parts := strings.Split(target.CollectionTopicPath, "/")
			if parent.TopicPath+"/"+parts[len(parts)-1] != target.CollectionTopicPath {
				return invalid(errors.New("topic ownership chain does not match the target collection"))
			}
		}
		operation := repository.NewMQTTOperation(apiRoot, principal, key, fingerprint)
		if key != "" {
			fresh, err := tx.ClaimMQTTOperation(ctx, &operation)
			if err != nil {
				return err
			}
			if !fresh {
				if operation.Fingerprint != fingerprint {
					return &pubsub.IngressError{Status: http.StatusConflict, Message: "idempotency-key was already used for different input"}
				}
				if _, err = resolver.ResolveIdentity(ctx, operation.ResourcePath); err != nil {
					if repository.IsNotFound(err) {
						return &pubsub.IngressError{Status: http.StatusGone, Message: "original resource is no longer available; the key cannot recreate it"}
					}
					return err
				}
				created.Path = operation.ResourcePath
				response = operation.Response
				replayed = true
				return nil
			}
		}
		var err error
		created, err = s.createOne(ctx, tx, creator, target.ParentID, media, body, value)
		if err != nil {
			return err
		}
		if created.Change.CollectionTopicPath != target.CollectionTopicPath {
			return fmt.Errorf("created resource does not belong to the publication topic")
		}
		for _, representation := range created.Change.Representations {
			if representation.MediaType == media {
				response = representation.Payload
			}
		}
		if len(response) == 0 {
			return fmt.Errorf("no canonical resource representation")
		}
		if key != "" {
			operation.ResourcePath = created.Path
			operation.MediaType = media
			operation.Response = response
			return tx.CompleteMQTTOperation(ctx, &operation)
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	result = pubsub.CreateResult{Payload: response, ContentType: media, ResourceURL: apiRoot + created.Path, Replayed: replayed}
	if !replayed && s.publisher != nil {
		change := created.Change
		change.SkipDataMediaType = media
		result.Publish = func() { s.publisher.PublishChange(change) }
	}
	return result, nil
}
