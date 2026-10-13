package repository

import "github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"

func testNumericObservationSchema() *domains.DatastreamSchema {
	return &domains.DatastreamSchema{ObsFormat: "application/json", ResultSchema: &domains.DatastreamDataComponent{Type: "Quantity", Definition: "http://example.test/value", Label: "Value", UOM: &domains.DatastreamUOM{Code: "1"}}}
}
func testNumericCommandSchema() *domains.ControlStreamSchema {
	return &domains.ControlStreamSchema{CommandFormat: "application/json", ParametersSchema: &domains.DatastreamDataComponent{Type: "Quantity", Definition: "http://example.test/setpoint", Label: "Set point", UOM: &domains.DatastreamUOM{Code: "1"}}}
}
