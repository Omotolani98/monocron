package monocrond

import (
    "encoding/json"
    "google.golang.org/grpc/encoding"
)

// jsonCodec enables JSON payloads over gRPC without protobufs.
type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }
func (jsonCodec) Marshal(v any) ([]byte, error)   { return json.Marshal(v) }
func (jsonCodec) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

func JsonCodecForClient() encoding.Codec { return jsonCodec{} }

