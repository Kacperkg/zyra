// seed-demo adds fictional reports for local UI development. Run only against
// the disposable Compose database with development login explicitly enabled.
package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"zyra-api/internal/auth"
	"zyra-api/internal/config"
	"zyra-api/internal/database"
	"zyra-api/internal/models"
	"zyra-api/internal/repository"
	"zyra-api/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if !cfg.DevAdminLogin {
		log.Fatal("seed-demo requires local development admin mode")
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if cfg.Migrate {
		if err := database.Migrate(db); err != nil {
			log.Fatal(err)
		}
	}
	svc := services.New(repository.New(db), auth.New(cfg.JWTSecret))
	svc.DevAdminLogin = true
	ctx := context.Background()
	if err := svc.BootstrapDevelopmentAdmin(ctx); err != nil {
		log.Fatal(err)
	}
	pair, err := svc.Login(ctx, "admin", "admin")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_, sessionID, err := svc.Authenticate(ctx, pair.AccessToken)
		if err == nil {
			_ = svc.Logout(ctx, sessionID)
		}
	}()
	base := time.Now().UTC().Truncate(time.Hour)
	reportsPerDatabase := []int{4, 3, 3}
	for i, name := range []string{"Northstar Retail", "Harbour Energy", "Cobalt Health"} {
		client := models.Client{ID: fmt.Sprintf("demo-client-%d", i), Name: name, Notes: "Fictional client for local development.", CreatedAt: base}
		if err := db.Where("id = ?", client.ID).FirstOrCreate(&client).Error; err != nil {
			log.Fatal(err)
		}
		d := models.Database{ID: fmt.Sprintf("demo-database-%d", i), ClientID: client.ID, Name: []string{"IFSPRD", "HEPRD1", "CHCDB"}[i], Hostname: fmt.Sprintf("demo-oracle-%02d", i+1), IP: fmt.Sprintf("192.0.2.%d", i+10), Notes: "Demo database. Confirm the affected resources before closing an issue.", CreatedAt: base}
		if err := db.Where("id = ?", d.ID).FirstOrCreate(&d).Error; err != nil {
			log.Fatal(err)
		}
		free, used := 5.0, 90.0
		settings := models.DatabaseSettings{ID: d.ID, Settings: models.CheckSettings{Selected: map[string]bool{"tablespace": true, "backups": true, "filesystem": true, "missing_email": true}, Resources: map[string]map[string]models.ResourceRule{"tablespace": {"UNDOTBS1": {MinFreePercent: &free}}, "backups": {`E:\ORADATA\APEX01.DBF`: {}}, "filesystem": {"G:": {MaxUsedPercent: &used}}}}}
		if err := db.Where("id = ?", d.ID).FirstOrCreate(&settings).Error; err != nil {
			log.Fatal(err)
		}
		for run := 0; run < reportsPerDatabase[i]; run++ {
			messageID := fmt.Sprintf("zyra-demo-%d-%d", i, run)
			var existing int64
			if err := db.Model(&models.Assessment{}).Where("message_id = ?", messageID).Count(&existing).Error; err != nil {
				log.Fatal(err)
			}
			if existing > 0 {
				continue
			}
			when := base.Add(-time.Duration(run*12+i) * time.Hour)
			svc.Now = func() time.Time { return when }
			body := fmt.Sprintf("ReportOn: %s, PkgVersion: 2.5\nDatabase: %s\nDatabase checks\nBackups=\\\nDatafiles needing backup:\nRMAN E:\\ORADATA\\APEX01.DBF 22-SEP-2026 00:42:16\nTablespaces=\\\nUNDOTBS1 31744 31744 30900 2.7%%\nFilesystem Usage\nG: 94%% - 122.88Gb of 2048.00Gb free\nScript Info\nScript : E:\\oracle\\daily.cmd\nRun by : oracle@%s", when.Format("02-Jan-2006:15:04:05"), d.Name, d.Hostname)
			a, err := svc.SubmitReport(ctx, pair.User, d.ID, services.ReportInput{MessageID: messageID, Subject: d.Name + " - daily check", Sender: "reports@example.test", Body: body, ReceivedAt: when})
			if err != nil {
				log.Fatal(err)
			}
			if run%5 == 0 {
				var tickets []models.Ticket
				if err := svc.Store.Tickets().Find(ctx, &tickets, repository.Query{Where: map[string]any{"assessment_id": a.ID}}); err != nil {
					log.Fatal(err)
				}
				for _, ticket := range tickets {
					if _, err := svc.TicketAction(ctx, pair.User, ticket.ID, "comment_and_close", "Demo investigation complete. This is fictional test data."); err != nil {
						log.Fatal(err)
					}
				}
			}
		}
	}
	log.Print("Demo reports ready (30 tickets on a fresh database); existing demo reports and operator edits were preserved")
}
