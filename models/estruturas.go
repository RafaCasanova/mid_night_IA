package models

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------
// ESTRUTURAS DE DADOS E MÉTODOS DE DOMÍNIO
// ---------------------------------------------------------

type Mensagem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestOllama struct {
	Model    string     `json:"model"`
	Messages []Mensagem `json:"messages"`
	Stream   bool       `json:"stream"`
	Format   string     `json:"format,omitempty"`
}

type ResponseOllama struct {
	Message Mensagem `json:"message"`
}

type Decisao struct {
	Acao       string `json:"acao"`
	Dados      string `json:"dados"`
	Raciocinio string `json:"raciocinio"`
}

type RegistroPasso struct {
	Passo      int
	Acao       string
	Dados      string
	Raciocinio string
	Resultado  string
	Timestamp  string
}

type HistoricoAgente struct {
	NomeAgente string
	Passos     []RegistroPasso
}

func (h *HistoricoAgente) Adicionar(r RegistroPasso) {
	h.Passos = append(h.Passos, r)
}

func (h *HistoricoAgente) Formatar() string {
	if len(h.Passos) == 0 {
		return "Nenhuma ação tomada ainda nesta sessão.\n"
	}
	var sb strings.Builder
	for _, p := range h.Passos {
		sb.WriteString(fmt.Sprintf(
			"[Passo %d @ %s]\n  Raciocínio: %s\n  Ação: %s\n  Dados: %s\n  Resultado:\n%s\n\n",
			p.Passo, p.Timestamp, p.Raciocinio, p.Acao, p.Dados,
			Indent(Truncar(p.Resultado, 400)),
		))
	}
	return sb.String()
}

func (h *HistoricoAgente) ComandosJaUsados() string {
	if len(h.Passos) == 0 {
		return "Nenhum"
	}
	var sb strings.Builder
	for _, p := range h.Passos {
		sb.WriteString(fmt.Sprintf("- [Passo %d] %s: %s\n", p.Passo, p.Acao, p.Dados))
	}
	return sb.String()
}

// ---------------------------------------------------------
// FUNÇÕES AUXILIARES EXPORTADAS
// ---------------------------------------------------------

// Indent adiciona espaçamento para formatação de logs.
func Indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = "    " + l
	}
	return strings.Join(lines, "\n")
}

// Truncar limita o tamanho das strings para evitar estouro de contexto.
func Truncar(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + fmt.Sprintf("\n... [truncado: %d chars omitidos]", len(s)-max)
}

// AnaliseProblema é o formato de saída do Agente 2 (Planejador)
type AnaliseProblema struct {
	Explicacao  string `json:"explicacao"`
	TemCorrecao bool   `json:"tem_correcao"`
	Plano       string `json:"plano"`
}

