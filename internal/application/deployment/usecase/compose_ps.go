package deploymentsvc

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
)

// composePsRow is the docker compose ps --format json record schema used by
// the runtime command.
type composePsRow struct {
	Id      string `json:"ID"`
	Name    string `json:"Name"`
	Service string `json:"Service"`
	State   string `json:"State"`
	Status  string `json:"Status"`
	Health  string `json:"Health"`
	Image   string `json:"Image"`
}

func parseComposePsOutput(raw string) ([]deploymentdto.RuntimeContainer, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return []deploymentdto.RuntimeContainer{}, nil
	}

	var rows []composePsRow
	var err error
	switch text[0] {
	case '[':
		err = json.Unmarshal([]byte(text), &rows)
	case '{':
		rows, err = decodeComposePsRecords(text)
	default:
		return nil, fmt.Errorf("compose ps output must be JSON")
	}
	if err != nil {
		return nil, fmt.Errorf("decode compose ps output: %w", err)
	}
	containers := make([]deploymentdto.RuntimeContainer, 0, len(rows))
	for _, row := range rows {
		containers = append(containers, deploymentdto.RuntimeContainer{
			Id:      row.Id,
			Name:    row.Name,
			Service: row.Service,
			State:   row.State,
			Status:  row.Status,
			Health:  row.Health,
			Image:   row.Image,
		})
	}
	return containers, nil
}

func decodeComposePsRecords(text string) ([]composePsRow, error) {
	decoder := json.NewDecoder(strings.NewReader(text))
	rows := make([]composePsRow, 0)
	for {
		var row composePsRow
		if err := decoder.Decode(&row); err != nil {
			if err == io.EOF {
				return rows, nil
			}
			return nil, err
		}
		rows = append(rows, row)
	}
}
