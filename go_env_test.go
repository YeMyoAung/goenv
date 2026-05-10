package goenv

import (
	"log"
	"testing"
	"time"
)

type Config struct {
	Name      string            `json:"name" validate:"required"`
	Foods     []string          `json:"foods" validate:"required"`
	Age       int               `json:"age" validate:"required"`
	IsStudent bool              `json:"is_student"`
	Height    float64           `json:"height" validate:"required"`
	TTL       time.Duration     `json:"ttl" validate:"required"`
	Headers   map[string]string `json:"headers" validate:"required"`
}

func TestGoEnvLoader(t *testing.T) {
	config, err := Load[Config](&Args{
		FileName: ".env.example",
	})
	if err != nil {
		t.Fatal(err)
	}
	if config == nil {
		t.Fatal("Config was null")
	}

	log.Println(config)
}
