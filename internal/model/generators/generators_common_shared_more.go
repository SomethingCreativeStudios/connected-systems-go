package generators

import (
	"encoding/json"
	"time"

	"github.com/yourusername/connected-systems-go/internal/model/common_shared"
)

// FakeDocument returns a Document with link
func FakeDocument() common_shared.Document {
	return common_shared.Document{
		Role:        f.Internet().URL(),
		Name:        f.Lorem().Word(),
		Description: f.Lorem().Sentence(2),
		Link:        FakeLink(),
	}
}

// FakeDocuments returns a Documents slice
func FakeDocuments() common_shared.Documents {
	return common_shared.Documents{FakeDocument()}
}

// FakeHistoryEvent returns a simple HistoryEvent
func FakeHistoryEvent() common_shared.HistoryEvent {
	now := time.Now()
	ht := common_shared.HistoryTime{Instant: &now}
	prop := FakeComponentWrapper()
	rawProps, _ := json.Marshal(prop)
	return common_shared.HistoryEvent{
		ID:            f.Lorem().Word(),
		Label:         f.Lorem().Word(),
		Description:   f.Lorem().Sentence(2),
		Definition:    f.Internet().URL(),
		Identifiers:   FakeTerms(),
		Classifiers:   FakeTerms(),
		Contacts:      []common_shared.ContactWrapper{FakeContactWrapper()},
		Documentation: FakeDocuments(),
		Time:          ht,
		Properties:    []common_shared.ComponentWrapper{prop},
		Configuration: rawProps,
	}
}

// FakeHistory returns a History slice
func FakeHistory() common_shared.History {
	return common_shared.History{FakeHistoryEvent()}
}

// FakeProperties returns a simple Properties map
func FakeProperties() common_shared.Properties {
	return common_shared.Properties{
		"example": f.Lorem().Word(),
		"number":  42,
	}
}

// FakeSecurityConstraints returns one SecurityConstraint
func FakeSecurityConstraints() common_shared.SecurityConstraints {
	sc := common_shared.SecurityConstraint{Type: f.Internet().URL()}
	sc.Extra = map[string]interface{}{"level": "public"}
	return common_shared.SecurityConstraints{sc}
}

// FakeLegalConstraints returns one LegalConstraint
func FakeLegalConstraints() common_shared.LegalConstraints {
	return common_shared.LegalConstraints{{
		AccessConstraints: common_shared.CodeLists{{CodeSpace: f.Internet().URL(), Value: "access"}},
		UseConstraints:    common_shared.CodeLists{{CodeSpace: f.Internet().URL(), Value: "use"}},
		OtherConstraints:  FakeTerms(),
	}}
}

// FakeIOList returns an IOList with one Component entry
func FakeIOList() common_shared.IOList {
	cw := FakeComponentWrapper()
	raw, _ := json.Marshal(cw)
	return common_shared.IOList{{Component: &cw, Raw: raw}}
}

// FakeMethod returns a Method
func FakeMethod() common_shared.Method {
	return common_shared.Method{Algorithm: f.Lorem().Word(), Description: f.Lorem().Sentence(2)}
}

// FakeSpatialFrame returns a SpatialFrame
func FakeSpatialFrame() common_shared.SpatialFrame {
	return common_shared.SpatialFrame{
		ID:          f.Lorem().Word(),
		Label:       f.Lorem().Word(),
		Description: f.Lorem().Sentence(2),
		Origin:      "urn:ogc:def:crs:EPSG::4326",
		Axes:        []common_shared.Axis{{Name: "X", Description: "Longitude"}, {Name: "Y", Description: "Latitude"}},
	}
}

// FakeTemporalFrame returns a TemporalFrame
func FakeTemporalFrame() common_shared.TemporalFrame {
	return common_shared.TemporalFrame{
		ID:          f.Lorem().Word(),
		Label:       f.Lorem().Word(),
		Description: f.Lorem().Sentence(2),
		Origin:      time.Now().Format(time.RFC3339),
	}
}

// FakeBoundingBox returns a BoundingBox
func FakeBoundingBox() common_shared.BoundingBox {
	return common_shared.BoundingBox{MinX: -180, MinY: -90, MaxX: 180, MaxY: 90}
}

// FakeExtent returns an Extent
func FakeExtent() common_shared.Extent {
	bb := FakeBoundingBox()
	return common_shared.Extent{Spatial: &bb, Temporal: FakeTimeRange()}
}

// FakeCodeLists returns one CodeList
func FakeCodeLists() common_shared.CodeLists {
	return common_shared.CodeLists{{CodeSpace: f.Internet().URL(), Value: f.Lorem().Word()}}
}
