package transport

import (
	"fmt"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoJSONMarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: true,
}

var protoJSONUnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: false,
}

type protoJSONRenderer struct {
	message proto.Message
}

func (r protoJSONRenderer) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)
	data, err := MarshalProtoJSON(r.message)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func (r protoJSONRenderer) WriteContentType(w http.ResponseWriter) {
	w.Header()["Content-Type"] = []string{"application/json; charset=utf-8"}
}

func MarshalProtoJSON(message proto.Message) ([]byte, error) {
	data, err := protoJSONMarshalOptions.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal proto json: %w", err)
	}
	return data, nil
}

func UnmarshalProtoJSON(data []byte, message proto.Message) error {
	if err := protoJSONUnmarshalOptions.Unmarshal(data, message); err != nil {
		return fmt.Errorf("unmarshal proto json: %w", err)
	}
	return nil
}
