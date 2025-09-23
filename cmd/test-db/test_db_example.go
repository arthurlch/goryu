package main

import (
	"fmt"
	"github.com/arthurlch/goryu/config"
	"github.com/arthurlch/goryu/db"
)

func main() {
	cfg, _ := config.LoadConfigWithFile("test_config.json")
	conn, err := db.Connect(cfg)
	if err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("Success! Connected to:", conn.Driver)
		if err := conn.Close(); err != nil {
			fmt.Printf("Error closing connection: %v\n", err)
		}
	}
}
