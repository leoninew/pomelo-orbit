package codec

import (
	"fmt"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var MarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: true,
}

var UnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: false,
}

type ProtoJSON struct {
	Message proto.Message
}

func (r ProtoJSON) Render(w http.ResponseWriter) error {
	data, err := MarshalProtoJSON(r.Message)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (r ProtoJSON) WriteContentType(w http.ResponseWriter) {
	w.Header()["Content-Type"] = []string{"application/json; charset=utf-8"}
}

func MarshalProtoJSON(message proto.Message) ([]byte, error) {
	data, err := MarshalOptions.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal proto json: %w", err)
	}
	return data, nil
}

func UnmarshalProtoJSON(data []byte, message proto.Message) error {
	if err := UnmarshalOptions.Unmarshal(data, message); err != nil {
		return fmt.Errorf("unmarshal proto json: %w", err)
	}
	return nil
}
