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
	priors, err := app.LoadIntentPriors(cfg.ModelPriorsPath)
	if err != nil {
		log.Printf("failed to load priors from %s, fallback to default predictor: %v", cfg.ModelPriorsPath, err)
	}
	if len(priors) > 0 {
		log.Printf("loaded intent priors from %s", cfg.ModelPriorsPath)
	}

	model := app.NewModel(priors)
	h := api.NewHandler(model)
	mux := http.NewServeMux()
	h.Register(mux)

	log.Printf("inference service listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}
