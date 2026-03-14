package common_shared

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// ContactInfo holds nested contact details such as phone and address.
type ContactInfo struct {
	Phone               *Phone   `json:"phone,omitempty"`
	Address             *Address `json:"address,omitempty"`
	Website             string   `json:"website,omitempty"`
	HoursOfService      string   `json:"hoursOfService,omitempty"`
	ContactInstructions string   `json:"contactInstructions,omitempty"`
}

// Phone models simple phone contact information.
type Phone struct {
	Voice     string `json:"voice,omitempty"`
	Facsimile string `json:"facsimile,omitempty"`
}

// Address models address fields used by the OpenAPI contacts schema.
type Address struct {
	DeliveryPoint         string `json:"deliveryPoint,omitempty"`
	City                  string `json:"city,omitempty"`
	AdministrativeArea    string `json:"administrativeArea,omitempty"`
	PostalCode            string `json:"postalCode,omitempty"`
	Country               string `json:"country,omitempty"`
	ElectronicMailAddress string `json:"electronicMailAddress,omitempty"`
}

// ContactPersonOrg represents the common contact variant with individual or
// organization details.
type ContactPersonOrg struct {
	IndividualName   string       `json:"individualName,omitempty"`
	OrganisationName string       `json:"organisationName,omitempty"`
	PositionName     string       `json:"positionName,omitempty"`
	ContactInfo      *ContactInfo `json:"contactInfo,omitempty"`
	Role             string       `json:"role,omitempty"` // semantic role URI
}

// ContactLink represents the alternate contact variant that provides a
// reference link to contact information.
type ContactLink struct {
	Role string `json:"role,omitempty"`
	Name string `json:"name,omitempty"`
	Link Link   `json:"link"`
}

// ContactWrapper detects and holds whichever contact variant was present in
// the payload. It preserves the raw JSON and exposes typed fields when
// unmarshalled.
type ContactWrapper struct {
	Person  *ContactPersonOrg `json:"-"`
	LinkRef *ContactLink      `json:"-"`
	Raw     json.RawMessage   `json:"-"`
}

// UnmarshalJSON inspects the payload to choose the correct variant.
func (c *ContactWrapper) UnmarshalJSON(b []byte) error {
	c.Raw = append([]byte(nil), b...)

	// Quick probe for keys - we'll try to detect by presence of "link" or
	// the "individualName"/"organisationName" fields.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(b, &probe); err != nil {
		return err
	}

	if _, hasLink := probe["link"]; hasLink {
		var cl ContactLink
		if err := json.Unmarshal(b, &cl); err != nil {
			return err
		}
		c.LinkRef = &cl
		return nil
	}

	// Fallback to person/org variant
	var cp ContactPersonOrg
	if err := json.Unmarshal(b, &cp); err != nil {
		return err
	}
	c.Person = &cp
	return nil
}

// GetDisplayName returns the best available human-readable name for the
// contact (individual > organisation > link name).
func (c *ContactWrapper) GetDisplayName() string {
	if c == nil {
		return ""
	}
	if c.Person != nil {
		if c.Person.IndividualName != "" {
			return c.Person.IndividualName
		}
		if c.Person.OrganisationName != "" {
			return c.Person.OrganisationName
		}
	}
	if c.LinkRef != nil {
		return c.LinkRef.Name
	}
	return ""
}

// MarshalJSON implements json.Marshaler to serialize the correct variant.
func (c ContactWrapper) MarshalJSON() ([]byte, error) {
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	if c.Person != nil {
		return json.Marshal(c.Person)
	}
	if c.LinkRef != nil {
		return json.Marshal(c.LinkRef)
	}
	return []byte("{}"), nil
}

// ContactWrappers is a JSONB-backed slice of ContactWrapper for GORM
type ContactWrappers []ContactWrapper

// Value implements driver.Valuer for JSONB storage
func (c ContactWrappers) Value() (driver.Value, error) {
	if c == nil {
		return []byte("[]"), nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Scan implements sql.Scanner for JSONB retrieval
func (c *ContactWrappers) Scan(value interface{}) error {
	if value == nil {
		*c = nil
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan type %T into ContactWrappers", value)
	}
	return json.Unmarshal(b, c)
}

// Gorm data type hints
func (ContactWrappers) GormDataType() string { return "jsonb" }

func (ContactWrappers) GormDBDataType(db *gorm.DB, _ *schema.Field) string { return "jsonb" }
