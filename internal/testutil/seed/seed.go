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

func RemoveSQLiteExportedGatewaySeed(t testing.TB, database *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		"DELETE FROM route WHERE id = '01M01ZNW6CPQCB7P5PN669HWJ9'",
		"DELETE FROM service_component WHERE id = '01M01RHDXW3ZXC7YKNT8GDNQNB'",
		"DELETE FROM service WHERE id = '01M01RHDXW3ZXC7YKNT54RWM1M'",
		"DELETE FROM version_component_mount WHERE component_id = '01M01MP096R73Z3MG28P91CSPN'",
		"DELETE FROM version_component_endpoint WHERE component_id = '01M01MP096R73Z3MG28P91CSPN'",
		"DELETE FROM version_component WHERE id = '01M01MP096R73Z3MG28P91CSPN'",
		"DELETE FROM gateway_config WHERE application_id = '01M01MP0950ECGK2DS1FWYNC0B'",
		"DELETE FROM version WHERE id = '01M01MP0950ECGK2DS1J4N4P50'",
		"DELETE FROM application WHERE id = '01M01MP0950ECGK2DS1FWYNC0B'",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("remove exported gateway seed: %v", err)
		}
	}
}
