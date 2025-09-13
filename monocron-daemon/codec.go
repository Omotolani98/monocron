package monocrond

import (
    "encoding/json"
    "google.golang.org/grpc/encoding"
)

// jsonCodec allows us to use JSON payloads over gRPC
// without requiring protobuf code generation.
type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }
func (jsonCodec) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (jsonCodec) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

// JsonCodecForClient returns the JSON codec registered with grpc/encoding.
func JsonCodecForClient() encoding.Codec { return jsonCodec{} }

