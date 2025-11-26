package common_shared

import (
	"database/sql/driver"
	"encoding/json"
)

type LegalConstraint struct {
	AccessConstraints CodeLists `json:"accessConstraints"`
	UseConstraints    CodeLists `json:"useConstraints"`
	OtherConstraints  Terms     `json:"otherConstraints"`
	UserLimitations   *string   `json:"userLimitations,omitempty"`
}

type LegalConstraints []LegalConstraint

// Value implements driver.Valuer for JSONB storage
func (l LegalConstraints) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Scan implements sql.Scanner for JSONB retrieval
func (l *LegalConstraints) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, l)
}
