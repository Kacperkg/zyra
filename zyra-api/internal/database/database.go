package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"zyra-api/internal/models"
)

func Connect(url string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(url), &gorm.Config{TranslateError: true})
}

// Migrate is explicitly enabled by the operator for this development implementation.
func Migrate(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(&models.User{}, &models.Client{}, &models.Database{}, &models.DatabaseSettings{}, &models.Session{}, &models.ResetToken{}, &models.EmailSource{}, &models.Assessment{}, &models.Ticket{}, &models.TicketEvent{}, &models.Comment{}, &models.SavedTicket{}, &models.CommentMention{}, &models.CommentNotificationDelivery{}, &models.Notification{}, &models.AccountLifecycle{}); err != nil {
			return err
		}
		if tx.Migrator().HasColumn("users", "disabled") {
			for _, sql := range []string{
				`UPDATE users SET status = CASE WHEN disabled THEN 'retired' ELSE 'active' END`,
				`UPDATE sessions SET revoked = true WHERE user_id IN (SELECT id FROM users WHERE status = 'retired')`,
				`UPDATE reset_tokens SET used = true WHERE user_id IN (SELECT id FROM users WHERE status = 'retired')`,
				`ALTER TABLE users DROP COLUMN disabled`,
			} {
				if err := tx.Exec(sql).Error; err != nil {
					return err
				}
			}
		}
		for _, sql := range []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_single_owner ON users (role) WHERE role = 'owner'`,
			`ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_account_status`,
			`ALTER TABLE users ADD CONSTRAINT chk_account_status CHECK (status IN ('active','retired') AND (role <> 'owner' OR status = 'active'))`,
			`INSERT INTO comments (id,ticket_id,user_id,schema_version,content,text,revision,created_at)
	   SELECT id,ticket_id,user_id,1,jsonb_build_object('blocks',jsonb_build_array(jsonb_build_object('type','paragraph','children',jsonb_build_array(jsonb_build_object('type','text','text',COALESCE(comment,'')))))),COALESCE(comment,''),1,created_at
	   FROM ticket_events WHERE type IN ('comment','comment_and_close') AND user_id <> '' AND COALESCE(comment_id,'') = '' ON CONFLICT (id) DO NOTHING`,
			`INSERT INTO ticket_events (id,ticket_id,user_id,type,comment,created_at,comment_id)
	   SELECT id || '-closure',ticket_id,user_id,'close','',created_at,'' FROM ticket_events WHERE type = 'comment_and_close' AND user_id <> '' AND COALESCE(comment_id,'') = '' ON CONFLICT (id) DO NOTHING`,
			`UPDATE ticket_events SET comment_id = id, comment = '', type = 'comment' WHERE type IN ('comment','comment_and_close') AND user_id <> '' AND COALESCE(comment_id,'') = ''`,
		} {
			if err := tx.Exec(sql).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
