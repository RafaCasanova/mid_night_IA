package llm

import (
	"mid_night/models"
)

// EnviarComHistorico é a forma correta de chamar a LLM.
// Recebe o histórico acumulado (system + turnos anteriores),
// adiciona a nova mensagem do usuário, envia tudo para o Ollama,
// e já registra a resposta de volta no histórico.
//
// Padrão baseado em:
// https://github.com/ollama/ollama/blob/main/api/examples/chat/main.go
//
// IMPORTANTE: a nova mensagem do usuário é adicionada ANTES do envio,
// garantindo que o system sempre seja o primeiro elemento da lista.
func EnviarComHistorico(urlOllama, modelo string, hist *models.HistoricoChat, novaMensagem string) (string, error) {
	// 1. Registra a fala do usuário no histórico
	hist.AdicionarUser(novaMensagem)

	// 2. Monta a requisição com todo o histórico (system + turnos + nova mensagem)
	req := models.RequestOllama{
		Model:    modelo,
		Stream:   false,
		Messages: hist.Snapshot(), // system está sempre no índice 0
	}

	// 3. Envia para o Ollama
	resp, err := EnviarParaOllama(urlOllama, req)
	if err != nil {
		// Remove a mensagem do usuário do histórico para não corromper o estado
		msgs := hist.Snapshot()
		hist.Mensagens = msgs[:len(msgs)-1]
		return "", err
	}

	// 4. Registra a resposta da IA no histórico para o próximo turno
	resposta := LimparConteudo(resp.Message.Content)
	hist.AdicionarAssistant(resposta)

	return resposta, nil
}
