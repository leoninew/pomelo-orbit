package cihandler

import (
	"encoding/json"

	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/codec"
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
	if err := transportcodec.UnmarshalProtoJSON(data, decoded); err != nil {
		return err
	}
	r.Body = decoded
	_, r.BranchSet = raw["branch_filter"]
	return nil
}
