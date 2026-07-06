package cihandler

import (
	"encoding/json"

	apiv1 "backend/internal/gen/orbit/api/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type repositoryWebhookUpdateReq struct {
	Body      *apiv1.RepositoryWebhookUpdateReq
	BranchSet bool
}

func (r *repositoryWebhookUpdateReq) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded := &apiv1.RepositoryWebhookUpdateReq{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, decoded); err != nil {
		return err
	}
	r.Body = decoded
	_, r.BranchSet = raw["branch_filter"]
	return nil
}
