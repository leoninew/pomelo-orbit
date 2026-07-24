package deploymentsvc

import (
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

func normalizeInstanceKey(instanceKey string) (string, error) {
	instanceKey = strings.TrimSpace(instanceKey)
	if instanceKey == "" {
		return "default", nil
	}
	if instanceKey != "default" {
		return "", apperror.New(apperror.KindValidation, "instance_key must be default (single runtime per application)")
	}
	return instanceKey, nil
}
