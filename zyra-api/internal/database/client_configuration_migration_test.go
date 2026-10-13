package database_test

import (
	"fmt"
	"os"
	"testing"
	"time"
	"zyra-api/internal/database"
	"zyra-api/internal/models"
)

func TestPostgresLegacyClientConfigurationMigration(t *testing.T) {
	dsn := os.Getenv("ZYRA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set disposable PostgreSQL DSN")
	}
	root, err := database.Connect(dsn)
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := root.DB()
	defer pool.Close()
	schema := fmt.Sprintf("zyra_configuration_migration_%d", time.Now().UnixNano())
	if err := root.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	defer root.Exec("DROP SCHEMA " + schema + " CASCADE")
	db, err := database.Connect(dsn + " search_path=" + schema)
	if err != nil {
		t.Fatal(err)
	}
	connection, _ := db.DB()
	defer connection.Close()
	for _, statement := range []string{
		`CREATE TABLE databases (id text PRIMARY KEY,client_id text,name text,hostname text,ip text,notes text,archived boolean,created_at timestamptz)`,
		`INSERT INTO databases VALUES ('legacy-db','legacy-client','DB','host','','original notes',false,now())`,
		`CREATE TABLE email_sources (id text PRIMARY KEY,database_id text,sender text,subject text,timezone text,enabled boolean,schedules jsonb)`,
		`INSERT INTO email_sources VALUES ('legacy-source','legacy-db','report@example.test','daily','Europe/London',true,'[{"name":"AM","start":"06:00","end":"10:00"}]')`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := database.Migrate(db); err != nil {
			t.Fatal(err)
		}
	}
	var d models.Database
	if err := db.First(&d, "id = ?", "legacy-db").Error; err != nil {
		t.Fatal(err)
	}
	if d.Engine != "oracle" || d.MaxLagSequences != nil || d.ObservedUniqueID != "" || d.Notes != "original notes" {
		t.Fatalf("unsafe database migration %+v", d)
	}
	var source models.EmailSource
	if err := db.First(&source, "id = ?", "legacy-source").Error; err != nil {
		t.Fatal(err)
	}
	if source.AssessmentType != "daily_check" || source.Mailbox != "" || len(source.Weekdays) != 7 || source.EffectiveAt != nil || len(source.Schedules) != 1 || !source.Enabled {
		t.Fatalf("unsafe source migration %+v", source)
	}
}
