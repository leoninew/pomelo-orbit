package cihandler

import (
	"encoding/json"

	pomeloorbit "gitee.com/leoninew/pomelo-orbit/internal/gen/proto/orbit"

	"google.golang.org/protobuf/encoding/protojson"
)

type repositoryWebhookUpdateReq struct {
	Body      *pomeloorbit.RepositoryWebhookUpdateReq
	BranchSet bool
}

func (r *repositoryWebhookUpdateReq) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded := &pomeloorbit.RepositoryWebhookUpdateReq{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, decoded); err != nil {
		return err
	}
	r.Body = decoded
	_, r.BranchSet = raw["branch_filter"]
	return nil
}
