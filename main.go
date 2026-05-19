package main

import (
	"mid_night/agentes"
)

func main() {
	// Configurações principais do Laboratório
	//modelo := "qwen3-coder-next:cloud"
	modelo := "qwen3.5:2b"
	urlOllama := "http://localhost:11434/api/chat"
	autonomo := true
	// Instancia e roda a orquestração
	orquestrador := agentes.NovoOrquestrador(modelo, urlOllama,autonomo)
	orquestrador.Iniciar()
}
