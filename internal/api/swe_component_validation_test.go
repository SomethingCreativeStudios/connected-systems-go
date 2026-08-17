package api

import (
	"testing"

	"github.com/yourusername/connected-systems-go/internal/model/domains"
)

func ptrBool(b bool) *bool { return &b }

func TestValidateSWEComponent(t *testing.T) {
	quantityNoUOM := &domains.DatastreamDataComponent{
		Type: "Quantity", Definition: "urn:x:speed", Label: "Speed",
	}
	quantityWithCode := &domains.DatastreamDataComponent{
		Type: "Quantity", Definition: "urn:x:speed", Label: "Speed",
		UOM: &domains.DatastreamUOM{Code: "m/s"},
	}
	quantityWithHref := &domains.DatastreamDataComponent{
		Type: "Quantity", Definition: "urn:x:speed", Label: "Speed",
		UOM: &domains.DatastreamUOM{Href: "http://units/ms"},
	}
	quantityUOMEmpty := &domains.DatastreamDataComponent{
		Type: "Quantity", Definition: "urn:x:speed", Label: "Speed",
		UOM: &domains.DatastreamUOM{},
	}
	categoryOK := &domains.DatastreamDataComponent{
		Type: "Category", Definition: "urn:x:action", Label: "Action",
	}
	categoryNoLabel := &domains.DatastreamDataComponent{
		Type: "Category", Definition: "urn:x:action",
	}

	tests := []struct {
		name    string
		comp    *domains.DatastreamDataComponent
		wantErr bool
	}{
		{"nil is ok", nil, false},
		{"quantity without uom rejected", quantityNoUOM, true},
		{"quantity with code accepted", quantityWithCode, false},
		{"quantity with href accepted", quantityWithHref, false},
		{"quantity with empty uom rejected", quantityUOMEmpty, true},
		{"category with definition+label accepted", categoryOK, false},
		{"category without label rejected", categoryNoLabel, true},
		{
			"datarecord requires fields",
			&domains.DatastreamDataComponent{Type: "DataRecord"},
			true,
		},
		{
			"datarecord field must have name",
			&domains.DatastreamDataComponent{Type: "DataRecord", Fields: []domains.DatastreamNamedComponent{
				{DatastreamDataComponent: *categoryOK},
			}},
			true,
		},
		{
			"datarecord with valid field accepted",
			&domains.DatastreamDataComponent{Type: "DataRecord", Fields: []domains.DatastreamNamedComponent{
				{Name: "action", DatastreamDataComponent: *categoryOK},
			}},
			false,
		},
		{
			"nested uom-less quantity rejected",
			&domains.DatastreamDataComponent{Type: "DataRecord", Fields: []domains.DatastreamNamedComponent{
				{Name: "speed", DatastreamDataComponent: *quantityNoUOM},
			}},
			true,
		},
		{
			"vector requires referenceFrame and coordinates",
			&domains.DatastreamDataComponent{Type: "Vector", Definition: "urn:x:v", Label: "V"},
			true,
		},
		{
			"dataarray requires elementType",
			&domains.DatastreamDataComponent{Type: "DataArray"},
			true,
		},
		{
			"dataarray with valid elementType accepted",
			&domains.DatastreamDataComponent{Type: "DataArray", ElementType: &domains.DatastreamNamedComponent{
				Name: "e", DatastreamDataComponent: *quantityWithCode,
			}},
			false,
		},
		{
			"geometry requires srs",
			&domains.DatastreamDataComponent{Type: "Geometry", Definition: "urn:x:g", Label: "G"},
			true,
		},
		{
			"unknown extension type accepted",
			&domains.DatastreamDataComponent{Type: "VendorThing"},
			false,
		},
		{
			"optional field still needs required attrs",
			&domains.DatastreamDataComponent{Type: "DataRecord", Fields: []domains.DatastreamNamedComponent{
				{Name: "speed", DatastreamDataComponent: domains.DatastreamDataComponent{
					Type: "Quantity", Definition: "urn:x:s", Label: "S", Optional: ptrBool(true),
				}},
			}},
			true, // optional Quantity still requires uom
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSWEComponent(tt.comp, "root")
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
