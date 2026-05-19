package models

// HistoricoChat mantém a conversa acumulada no formato exato da API Ollama:
// system → user → assistant → user → assistant → ...
//
// É passado entre agentes para que cada chamada à LLM tenha contexto completo,
// exatamente como no exemplo oficial do Ollama:
// https://github.com/ollama/ollama/blob/main/api/examples/chat/main.go
type HistoricoChat struct {
	Mensagens []Mensagem
}

// NovoHistoricoChat cria um histórico já com o system prompt fixo do projeto.
// O system DEVE ser o primeiro elemento — antes de qualquer user/assistant.
func NovoHistoricoChat(systemPrompt string) *HistoricoChat {
	return &HistoricoChat{
		Mensagens: []Mensagem{
			{Role: "system", Content: systemPrompt},
		},
	}
}

// AdicionarUser acrescenta uma fala do usuário ao histórico.
func (h *HistoricoChat) AdicionarUser(conteudo string) {
	h.Mensagens = append(h.Mensagens, Mensagem{Role: "user", Content: conteudo})
}

// AdicionarAssistant acrescenta a resposta da IA ao histórico.
func (h *HistoricoChat) AdicionarAssistant(conteudo string) {
	h.Mensagens = append(h.Mensagens, Mensagem{Role: "assistant", Content: conteudo})
}

// Snapshot retorna uma cópia da lista de mensagens, segura para passar à API.
func (h *HistoricoChat) Snapshot() []Mensagem {
	copia := make([]Mensagem, len(h.Mensagens))
	copy(copia, h.Mensagens)
	return copia
}
