package main

import (
	// Standard libraries
	"log"
	"time"

	// Third-party libraries
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// BudgetDecision represents a decision made on a FinOps expense request.
type BudgetDecision struct {
	ID          uint   `gorm:"primaryKey"`
	AgentID     string `gorm:"index"`
	ChatMessage string
	Status      string // "PENDING", "APPROVED", "REJECTED"
	DecisionAI  string // AI reasoning or n8n output
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var DB *gorm.DB

// InitDB initializes the SQLite database and migrates the schema.
func InitDB(dsn string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("[Database] Connected to SQLite database.")

	// Auto Migrate the schema
	err = DB.AutoMigrate(&BudgetDecision{})
	if err != nil {
		log.Fatalf("Failed to auto-migrate database schema: %v", err)
	}
	log.Println("[Database] Schema auto-migrated successfully.")
}

// SaveBudgetDecision saves or updates a decision in the database.
func SaveBudgetDecision(decision *BudgetDecision) error {
	if DB == nil {
		log.Println("[Database Error] DB is not initialized")
		return nil
	}

	result := DB.Save(decision)
	if result.Error != nil {
		log.Printf("[Database Error] Failed to save budget decision: %v", result.Error)
		return result.Error
	}
	return nil
}
