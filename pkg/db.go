package pkg

import (
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/kenelite/gosphere-backend/model"
)

var db *gorm.DB

func InitDatabase() error {
	if db != nil {
		return nil
	}

	user := os.Getenv("MYSQL_USER")
	pass := os.Getenv("MYSQL_PASSWORD")
	host := os.Getenv("MYSQL_HOST")
	port := os.Getenv("MYSQL_PORT")
	name := os.Getenv("MYSQL_DATABASE")

	if port == "" {
		port = "3306"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, name)
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	// Assign after success
	db = conn

	// Auto-migrate core tables
	if err := db.AutoMigrate(&model.Tenant{}); err != nil {
		return err
	}
	return nil
}

func DB() *gorm.DB { return db }
