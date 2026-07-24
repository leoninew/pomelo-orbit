package binding

import (
	"encoding/json"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	repositoryv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/repository"
)

type RepositoryWebhookUpdateReq struct {
	Body      *repositoryv1.RepositoryWebhookUpdateReq
	BranchSet bool
}

func (r *RepositoryWebhookUpdateReq) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded := &repositoryv1.RepositoryWebhookUpdateReq{}
	if err := codec.UnmarshalProtoJSON(data, decoded); err != nil {
		return err
	}
	r.Body = decoded
	_, r.BranchSet = raw["branch_filter"]
	return nil
}
