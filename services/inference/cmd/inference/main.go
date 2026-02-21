package main

import (
	"log"
	"net/http"

	"github.com/dysania/meowlator/services/inference/internal/api"
	"github.com/dysania/meowlator/services/inference/internal/app"
	"github.com/dysania/meowlator/services/inference/internal/config"
)

func main() {
	cfg := config.Load()
	model := app.NewModel()
	h := api.NewHandler(model)
	mux := http.NewServeMux()
	h.Register(mux)

	log.Printf("inference service listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}
