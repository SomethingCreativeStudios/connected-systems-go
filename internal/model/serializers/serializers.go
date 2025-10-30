package serializers

import "context"

type Serializer[Output any, Input any] interface {
	Serialize(ctx context.Context, item Input) (Output, error)
	SerializeAll(ctx context.Context, items []Input) ([]Output, error)
}

type SerializerCollection[Output any, Input any] struct {
	serializers map[string]Serializer[Output, Input]
}

func NewSerializerCollection[Output any, Input any](serializers map[string]Serializer[Output, Input]) *SerializerCollection[Output, Input] {
	return &SerializerCollection[Output, Input]{serializers: serializers}
}

func (sc *SerializerCollection[Output, Input]) GetSerializer(key string) Serializer[Output, Input] {
	serializer, exists := sc.serializers[key]

	if !exists {
		return sc.serializers["default"]
	}

	return serializer
}

func (sc *SerializerCollection[Output, Input]) SerializeAll(key string, items []Input) ([]Output, error) {
	serializer := sc.GetSerializer(key)
	return serializer.SerializeAll(context.Background(), items)
}

func (sc *SerializerCollection[Output, Input]) Serialize(key string, item Input) (Output, error) {
	serializer := sc.GetSerializer(key)
	return serializer.Serialize(context.Background(), item)
}
