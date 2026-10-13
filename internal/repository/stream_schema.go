package repository

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"slices"
	"strings"

	"github.com/SomethingCreativeStudios/connected-systems-go/internal/model/domains"
	"gorm.io/gorm"
)

// Schema equality ignores JSON object key order, while retaining array order.
func schemasEqual(a, b any) bool {

	rawA, err := json.Marshal(a)
	if err != nil {
		return false
	}
	rawB, err := json.Marshal(b)
	if err != nil {
		return false
	}

	decode := func(raw []byte) (any, error) {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		return canonicalSchemaValue(value), nil
	}
	decodedA, err := decode(rawA)
	if err != nil {
		return false
	}
	decodedB, err := decode(rawB)
	if err != nil {
		return false
	}
	return reflect.DeepEqual(decodedA, decodedB)
}

type schemaNumber string

// Normalize decimals exactly without float64 rounding or expanding exponents.
func canonicalSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			typed[key] = canonicalSchemaValue(item)
		}
	case []any:
		for i, item := range typed {
			typed[i] = canonicalSchemaValue(item)
		}
	case json.Number:
		coefficient, exponent, hasExponent := strings.Cut(strings.ToLower(string(typed)), "e")
		negative := strings.HasPrefix(coefficient, "-")
		coefficient = strings.TrimPrefix(coefficient, "-")
		integer, fraction, _ := strings.Cut(coefficient, ".")
		digits := strings.TrimLeft(integer+fraction, "0")
		if digits == "" {
			return schemaNumber("0")
		}
		trimmed := strings.TrimRight(digits, "0")
		power := new(big.Int)
		if hasExponent {
			power.SetString(exponent, 10)
		}
		power.Add(power, big.NewInt(int64(len(digits)-len(trimmed)-len(fraction))))
		sign := ""
		if negative {
			sign = "-"
		}
		return schemaNumber(sign + trimmed + "e" + power.String())
	}
	return value
}

func mergeDatastreamSchema(tx *gorm.DB, replacement, existing *domains.Datastream) error {
	registry := slices.Clone(existing.Schemas)
	if existing.Schema != nil && registry.ForFormat(existing.Schema.ObsFormat) == nil {
		registry = registry.Upsert(*existing.Schema)
	}
	if replacement.Schema == nil {
		replacement.Schema = existing.Schema
		if replacement.Schema == nil && len(registry) > 0 {
			replacement.Schema = registry.ForFormat("application/json")
			if replacement.Schema == nil {
				replacement.Schema = &registry[0]
			}
		}
	} else {
		incoming := *replacement.Schema
		incoming.ObsFormat = domains.NormalizeMediaType(incoming.ObsFormat)
		if incoming.ObsFormat == "" {
			incoming.ObsFormat = "application/json"
		}
		old := registry.ForFormat(incoming.ObsFormat)
		if old != nil {
			normalized := *old
			normalized.ObsFormat = domains.NormalizeMediaType(normalized.ObsFormat)
			if normalized.ObsFormat == "" {
				normalized.ObsFormat = "application/json"
			}
			old = &normalized
		}
		if !schemasEqual(old, &incoming) {
			if has, err := hasRows(tx, &domains.Observation{}, "datastream_id = ?", existing.ID); err != nil {
				return err
			} else if has {
				return ErrSchemaInUse
			}
		}
		replacement.Schema = &incoming
		registry = registry.Upsert(incoming)
	}
	replacement.Schemas = registry
	replacement.RecomputeSchemaMetadata()
	return nil
}

func mergeControlStreamSchema(tx *gorm.DB, replacement, existing *domains.ControlStream) error {
	registry := slices.Clone(existing.Schemas)
	if existing.Schema != nil && registry.ForFormat(existing.Schema.CommandFormat) == nil {
		registry = registry.Upsert(*existing.Schema)
	}
	if replacement.Schema == nil {
		replacement.Schema = existing.Schema
		if replacement.Schema == nil && len(registry) > 0 {
			replacement.Schema = registry.ForFormat("application/json")
			if replacement.Schema == nil {
				replacement.Schema = &registry[0]
			}
		}
	} else {
		incoming := *replacement.Schema
		incoming.CommandFormat = domains.NormalizeMediaType(incoming.CommandFormat)
		if incoming.CommandFormat == "" {
			incoming.CommandFormat = "application/json"
		}
		old := registry.ForFormat(incoming.CommandFormat)
		if old != nil {
			normalized := *old
			normalized.CommandFormat = domains.NormalizeMediaType(normalized.CommandFormat)
			if normalized.CommandFormat == "" {
				normalized.CommandFormat = "application/json"
			}
			old = &normalized
		}
		if !schemasEqual(old, &incoming) {
			if has, err := hasRows(tx, &domains.Command{}, "control_stream_id = ?", existing.ID); err != nil {
				return err
			} else if has {
				return ErrSchemaInUse
			}
		}
		replacement.Schema = &incoming
		registry = registry.Upsert(incoming)
	}
	replacement.Schemas = registry
	replacement.RecomputeSchemaMetadata()
	return nil
}
