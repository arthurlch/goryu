package main

import (
	"encoding/json"
	"fmt"
	"github.com/arthurlch/goryu/config"
)

func main() {
	// Test direct JSON unmarshaling
	jsonStr := `{"app":{"name":"test"},"server":{"host":"localhost","port":8080},"environment":"development","custom":{"database":{"driver":"sqlite3","path":"./test.db"}}}`

	var cfg config.Config
	if err := _ = json.Unmarshal([]byte(jsonStr), &cfg); err != nil {
		fmt.Println("JSON unmarshal error:", err)
		return
	}

	fmt.Printf("Direct unmarshal - Custom: %+v\n", cfg.Custom)

	// Test file loading
	fileCfg, err := config.LoadConfigWithFile("test_config.json")
	if err != nil {
		fmt.Println("File load error:", err)
		return
	}

	fmt.Printf("File load - Custom: %+v\n", fileCfg.Custom)
}
