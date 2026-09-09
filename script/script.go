package script

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	documentService "github.com/pln-colabora/colabora-be/modules/document/service"
	"gorm.io/gorm"
)

func Script(scriptName string, db *gorm.DB, documents documentService.DocumentService) error {
	switch scriptName {
	case "example_script":
		exampleScript := NewExampleScript(db)
		return exampleScript.Run()
	case "cleanup_orphan_documents":
		hours := 24
		if value, err := strconv.Atoi(os.Getenv("DOCUMENT_ORPHAN_TTL_HOURS")); err == nil && value > 0 {
			hours = value
		}
		deleted, err := documents.CleanupOrphans(context.Background(), time.Now().Add(-time.Duration(hours)*time.Hour), 1000)
		if err == nil {
			log.Printf("deleted %d expired unattached documents", deleted)
		}
		return err
	default:
		return errors.New("script not found")
	}
}
