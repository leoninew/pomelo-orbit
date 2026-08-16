package testseed

import (
	"database/sql"
	"testing"
)

func ApplySQLitePipelineDemo(t testing.TB, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
        INSERT OR IGNORE INTO repository (id, name, code, repository_url, project_id)
        VALUES (
            '01KNNRBH52BQJYT9487B2H8N62',
            'Pipeline Demo',
            'pipeline-demo',
            'https://example.invalid/pipeline-demo.git',
            '01KRRKK0K3T519ZQZES3M4QA9Z'
        )
    `)
	if err != nil {
		t.Fatalf("seed pipeline demo repository: %v", err)
	}
}
