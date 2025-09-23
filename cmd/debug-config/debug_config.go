package main

import (
	"fmt"
	"github.com/arthurlch/goryu/config"
)

func main() {
	cfg, err := config.LoadConfigWithFile("test_config.json")
	if err != nil {
		fmt.Println("Load error:", err)
		return
	}
	fmt.Printf("Config: %+v\n", cfg)
	fmt.Printf("Custom: %+v\n", cfg.Custom)
	if db, ok := cfg.Custom["database"]; ok {
		fmt.Printf("Database config found: %+v\n", db)
	} else {
		fmt.Println("No database config found in custom")
	}
}
