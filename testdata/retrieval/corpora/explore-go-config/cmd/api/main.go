package main

import (
	"context"
	"log"
	"net/http"

	"example.com/orders/config"
	"example.com/orders/storage"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	db, err := storage.OpenDatabase(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, newRouter(db)))
}
