package api

import (
	"fmt"
	"strings"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

// validateSWEComponent structurally validates a SWE Common component definition
// tree, enforcing the required fields each component type mandates per the OGC
// SWE Common JSON schema. This validates the *schema* itself (e.g. a datastream
// recordSchema or a control stream parametersSchema), not runtime values —
// value validation lives in observation_schema_validation.go.
//
// path identifies the component within the tree for error messages
// (e.g. "parametersSchema.fields[0]").
func validateSWEComponent(c *domains.DatastreamDataComponent, path string) error {
	if c == nil {
		return nil
	}

	// Infer a record shape when the type is omitted but fields are present,
	// mirroring the leniency of the value validator.
	componentType := c.Type
	if componentType == "" && len(c.Fields) > 0 {
		componentType = "DataRecord"
	}
	if componentType == "" {
		return fmt.Errorf("%s: component is missing required \"type\"", path)
	}

	// Fields required by every component that carries semantics + a label.
	requireDefinitionLabel := func() error {
		if c.Definition == "" {
			return fmt.Errorf("%s: %s requires \"definition\"", path, componentType)
		}
		if c.Label == "" {
			return fmt.Errorf("%s: %s requires \"label\"", path, componentType)
		}
		return nil
	}

	switch strings.ToLower(componentType) {
	// Numeric/temporal scalars additionally require a unit of measure.
	case "quantity", "quantityrange", "time", "timerange":
		if err := requireDefinitionLabel(); err != nil {
			return err
		}
		if err := validateUOM(c, path, componentType); err != nil {
			return err
		}

	// Simple scalars: definition + label, no uom.
	case "boolean", "category", "categoryrange", "count", "countrange", "text":
		if err := requireDefinitionLabel(); err != nil {
			return err
		}

	case "datarecord":
		if len(c.Fields) == 0 {
			return fmt.Errorf("%s: DataRecord requires non-empty \"fields\"", path)
		}
		for i := range c.Fields {
			f := &c.Fields[i]
			if f.Name == "" {
				return fmt.Errorf("%s.fields[%d]: field requires \"name\"", path, i)
			}
			if err := validateSWEComponent(&f.DatastreamDataComponent, fmt.Sprintf("%s.fields[%d](%s)", path, i, f.Name)); err != nil {
				return err
			}
		}

	case "datachoice":
		if len(c.Items) == 0 {
			return fmt.Errorf("%s: DataChoice requires non-empty \"items\"", path)
		}
		for i := range c.Items {
			it := &c.Items[i]
			if it.Name == "" {
				return fmt.Errorf("%s.items[%d]: item requires \"name\"", path, i)
			}
			if err := validateSWEComponent(&it.DatastreamDataComponent, fmt.Sprintf("%s.items[%d](%s)", path, i, it.Name)); err != nil {
				return err
			}
		}

	case "vector":
		if err := requireDefinitionLabel(); err != nil {
			return err
		}
		if c.ReferenceFrame == "" {
			return fmt.Errorf("%s: Vector requires \"referenceFrame\"", path)
		}
		if len(c.Coordinates) == 0 {
			return fmt.Errorf("%s: Vector requires non-empty \"coordinates\"", path)
		}
		for i := range c.Coordinates {
			co := &c.Coordinates[i]
			if co.Name == "" {
				return fmt.Errorf("%s.coordinates[%d]: coordinate requires \"name\"", path, i)
			}
			if err := validateSWEComponent(&co.DatastreamDataComponent, fmt.Sprintf("%s.coordinates[%d](%s)", path, i, co.Name)); err != nil {
				return err
			}
		}

	case "dataarray", "matrix":
		if c.ElementType == nil {
			return fmt.Errorf("%s: %s requires \"elementType\"", path, componentType)
		}
		if err := validateSWEComponent(&c.ElementType.DatastreamDataComponent, path+".elementType"); err != nil {
			return err
		}

	case "geometry":
		if c.SRS == "" {
			return fmt.Errorf("%s: Geometry requires \"srs\"", path)
		}
		if err := requireDefinitionLabel(); err != nil {
			return err
		}

	default:
		// Unknown/extension component types are accepted.
	}

	return nil
}

// validateControlStreamSchema structurally validates the SWE Common components
// carried by a control stream schema (both the JSON and SWE Common branches).
func validateControlStreamSchema(s *domains.ControlStreamSchema) error {
	if s == nil {
		return nil
	}
	if err := validateSWEComponent(s.ParametersSchema, "parametersSchema"); err != nil {
		return err
	}
	if err := validateSWEComponent(s.ResultSchema, "resultSchema"); err != nil {
		return err
	}
	if err := validateSWEComponent(s.FeasibilityResultSchema, "feasibilityResultSchema"); err != nil {
		return err
	}
	if err := validateSWEComponent(s.RecordSchema, "recordSchema"); err != nil {
		return err
	}
	return nil
}

// validateDatastreamSchema structurally validates the SWE Common components
// carried by a datastream schema (both the JSON and SWE Common branches).
func validateDatastreamSchema(s *domains.DatastreamSchema) error {
	if s == nil {
		return nil
	}
	if err := validateSWEComponent(s.ParametersSchema, "parametersSchema"); err != nil {
		return err
	}
	if err := validateSWEComponent(s.ResultSchema, "resultSchema"); err != nil {
		return err
	}
	if err := validateSWEComponent(s.RecordSchema, "recordSchema"); err != nil {
		return err
	}
	return nil
}

// validateUOM enforces that a unit-bearing component carries a uom with either a
// UCUM "code" or a "href" (the SWE Common UnitReference anyOf).
func validateUOM(c *domains.DatastreamDataComponent, path, componentType string) error {
	if c.UOM == nil {
		return fmt.Errorf("%s: %s requires \"uom\"", path, componentType)
	}
	if c.UOM.Code == "" && c.UOM.Href == "" {
		return fmt.Errorf("%s: %s \"uom\" requires either \"code\" (UCUM) or \"href\"", path, componentType)
	}
	return nil
}
