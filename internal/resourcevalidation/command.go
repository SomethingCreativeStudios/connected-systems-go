package resourcevalidation

import (
	"encoding/json"
	"fmt"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
)

// ValidateCommandAgainstControlStreamSchema validates JSON command parameters
// against the matching schema, including SWE component constraints.
func ValidateCommandAgainstControlStreamSchema(cmd *domains.Command, cs *domains.ControlStream) error {
	schema := cs.Schemas.ForFormat("application/json")
	if schema == nil {
		schema = cs.Schema
	}
	if schema == nil {
		return fmt.Errorf("parameters: control stream has no command schema")
	}
	component := schema.ParametersSchema
	if component == nil {
		component = schema.RecordSchema
	}
	if component == nil {
		return fmt.Errorf("parameters: control stream has no parameters schema")
	}
	var value any
	if err := json.Unmarshal(cmd.Parameters, &value); err != nil {
		return fmt.Errorf("parameters: invalid JSON: %w", err)
	}
	return validateDataComponentValue(component, value, "parameters")
}
