package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"gorm.io/gorm/clause"
)

// MQTTCreateOperation is committed in the resource's transaction. Records are
// intentionally independent of resource foreign keys and survive deletion.
type MQTTCreateOperation struct {
	ScopeKey     string `gorm:"primaryKey;type:char(64)"`
	APIContext   string
	Principal    string
	OperationKey string
	Fingerprint  string
	ResourcePath string
	MediaType    string
	Response     []byte
	CreatedAt    time.Time
}

func (MQTTCreateOperation) TableName() string { return "mqtt_create_operations" }

func NewMQTTOperation(apiContext, principal, key, fingerprint string) MQTTCreateOperation {
	scope, _ := json.Marshal([]string{apiContext, principal, key})
	sum := sha256.Sum256(scope)
	return MQTTCreateOperation{ScopeKey: hex.EncodeToString(sum[:]), APIContext: apiContext, Principal: principal, OperationKey: key, Fingerprint: fingerprint}
}

// ClaimMQTTOperation must run inside Transaction. PostgreSQL's unique-key
// conflict waits for a concurrent claimant to commit or roll back.
func (r *Repositories) ClaimMQTTOperation(ctx context.Context, operation *MQTTCreateOperation) (bool, error) {
	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(operation)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}
	err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(operation, "scope_key = ?", operation.ScopeKey).Error
	return false, err
}

func (r *Repositories) CompleteMQTTOperation(ctx context.Context, operation *MQTTCreateOperation) error {
	return r.db.WithContext(ctx).Save(operation).Error
}
