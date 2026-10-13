package domains

import (
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/mediatype"
	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/common_shared"
	"strings"
)

// RecomputeSchemaMetadata keeps stored query fields and response metadata in sync.
func (d *Datastream) RecomputeSchemaMetadata() {
	d.ObservedProperties = deriveDatastreamObservedProperties(d)
	d.ResultType = deriveDatastreamResultType(d)
	d.Formats = deriveDatastreamFormats(d)
}

func (c *ControlStream) RecomputeSchemaMetadata() {
	c.ControlledProperties = deriveControlStreamControlledProperties(c)
	c.Formats = deriveControlStreamFormats(c)
}

// definedComponent pairs a schema component carrying a semantic definition
// with its effective name (DatastreamNamedComponent shadows the embedded Name).
type definedComponent struct {
	component *DatastreamDataComponent
	name      string
}

// collectDefinedComponents recursively gathers every data component carrying a
// semantic definition URI from a schema component tree.
func collectDefinedComponents(component *DatastreamDataComponent, name string, out *[]definedComponent) {
	if component == nil {
		return
	}
	if component.Name != "" {
		name = component.Name
	}
	if component.Definition != "" {
		*out = append(*out, definedComponent{component: component, name: name})
	}
	for i := range component.Fields {
		collectDefinedComponents(&component.Fields[i].DatastreamDataComponent, component.Fields[i].Name, out)
	}
	for i := range component.Coordinates {
		collectDefinedComponents(&component.Coordinates[i].DatastreamDataComponent, component.Coordinates[i].Name, out)
	}
	for i := range component.Items {
		collectDefinedComponents(&component.Items[i].DatastreamDataComponent, component.Items[i].Name, out)
	}
	if component.ElementType != nil {
		collectDefinedComponents(&component.ElementType.DatastreamDataComponent, component.ElementType.Name, out)
	}
}

// datastreamResultComponent returns the observation result component of a
// datastream schema (JSON or SWE branch).
func datastreamResultComponent(schema *DatastreamSchema) *DatastreamDataComponent {
	if schema == nil {
		return nil
	}
	if schema.ResultSchema != nil {
		return schema.ResultSchema
	}
	return schema.RecordSchema
}

// deriveDatastreamObservedProperties computes the read-only, required
// "observedProperties" field from the semantic definitions declared in the
// result schemas of all registered schemas. Nil (-> null) when none declared.
func deriveDatastreamObservedProperties(datastream *Datastream) *DatastreamObservedProperties {
	var components []definedComponent
	for i := range datastream.Schemas {
		collectDefinedComponents(datastreamResultComponent(&datastream.Schemas[i]), "", &components)
	}
	// Legacy rows predate the schema registry.
	if len(components) == 0 {
		collectDefinedComponents(datastreamResultComponent(datastream.Schema), "", &components)
	}

	var props DatastreamObservedProperties
	seen := map[string]bool{}
	for _, dc := range components {
		c := dc.component
		if seen[c.Definition] {
			continue
		}
		seen[c.Definition] = true
		label := c.Label
		if label == "" {
			label = dc.name
		}
		props = append(props, DatastreamObservedProperty{
			Definition:  c.Definition,
			Label:       label,
			Description: c.Description,
		})
	}
	if len(props) == 0 {
		return nil
	}
	return &props
}

// deriveDatastreamResultType computes the read-only, required "resultType"
// field from the root component of the observation result schema.
// Nil (-> null) when no schema declares a result component.
func deriveDatastreamResultType(datastream *Datastream) *string {
	component := datastreamResultComponent(datastream.Schema)
	for i := range datastream.Schemas {
		if component != nil {
			break
		}
		component = datastreamResultComponent(&datastream.Schemas[i])
	}
	if component == nil {
		return nil
	}
	var resultType string
	switch strings.ToLower(component.Type) {
	case "quantity", "count":
		resultType = "measure"
	case "vector":
		resultType = "vector"
	case "datarecord":
		resultType = "record"
	case "dataarray", "matrix":
		resultType = "coverage"
	default:
		resultType = "complex"
	}
	return &resultType
}

// deriveDatastreamFormats computes the read-only, required "formats" field
// from the observation formats of all registered schemas.
func deriveDatastreamFormats(datastream *Datastream) common_shared.StringArray {
	var formats common_shared.StringArray
	seen := map[string]bool{}
	add := func(format string) {
		key := NormalizeMediaType(format)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		formats = append(formats, format)
	}
	for i := range datastream.Schemas {
		add(datastream.Schemas[i].ObsFormat)
	}
	// Legacy rows predate the schema registry.
	if datastream.Schema != nil {
		add(datastream.Schema.ObsFormat)
	}
	if len(formats) == 0 {
		formats = common_shared.StringArray{mediatype.JSON}
	}
	return formats
}

// controlStreamParamsComponent returns the command parameters component of a
// control stream schema (JSON or SWE branch).
func controlStreamParamsComponent(schema *ControlStreamSchema) *DatastreamDataComponent {
	if schema == nil {
		return nil
	}
	if schema.ParametersSchema != nil {
		return schema.ParametersSchema
	}
	return schema.RecordSchema
}

// deriveControlStreamControlledProperties computes the read-only, required
// "controlledProperties" field from the semantic definitions declared in the
// parameter schemas of all registered schemas. Nil (-> null) when none declared.
func deriveControlStreamControlledProperties(cs *ControlStream) *ControlStreamControlledProperties {
	var components []definedComponent
	for i := range cs.Schemas {
		collectDefinedComponents(controlStreamParamsComponent(&cs.Schemas[i]), "", &components)
	}
	// Legacy rows predate the schema registry.
	if len(components) == 0 {
		collectDefinedComponents(controlStreamParamsComponent(cs.Schema), "", &components)
	}

	var props ControlStreamControlledProperties
	seen := map[string]bool{}
	for _, dc := range components {
		c := dc.component
		if seen[c.Definition] {
			continue
		}
		seen[c.Definition] = true
		label := c.Label
		if label == "" {
			label = dc.name
		}
		props = append(props, ControlStreamControlledProperty{
			Definition:  c.Definition,
			Label:       label,
			Description: c.Description,
		})
	}
	if len(props) == 0 {
		return nil
	}
	return &props
}

// deriveControlStreamFormats computes the read-only, required "formats" field
// from the command formats of all registered schemas.
func deriveControlStreamFormats(cs *ControlStream) common_shared.StringArray {
	var formats common_shared.StringArray
	seen := map[string]bool{}
	add := func(format string) {
		key := NormalizeMediaType(format)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		formats = append(formats, format)
	}
	for i := range cs.Schemas {
		add(cs.Schemas[i].CommandFormat)
	}
	// Legacy rows predate the schema registry.
	if cs.Schema != nil {
		add(cs.Schema.CommandFormat)
	}
	if len(formats) == 0 {
		formats = common_shared.StringArray{mediatype.JSON}
	}
	return formats
}
