package agentes

import (
	"fmt"
	"strings"
	"time"

	"mid_night/llm"
	"mid_night/models"
	"mid_night/system"
)

const MAX_PASSOS_INVESTIGACAO = 12
const MAX_PASSOS_REPARO = 12

func agora() string { return time.Now().Format("15:04:05") }

type Orquestrador struct {
	Modelo        string
	UrlOllama     string
	Autonomo      bool
	MemoriaSessao string
}

func NovoOrquestrador(modelo, url string, autonomo bool) *Orquestrador {
	return &Orquestrador{Modelo: modelo, UrlOllama: url, Autonomo: autonomo}
}

func (o *Orquestrador) Iniciar() {
	fmt.Println("==================================================")
	fmt.Println("🤖 MID_NIGHT: AGENTES AUTÔNOMOS (Modo Interativo)")
	fmt.Println("   Digite '/bye' para sair.")
	fmt.Println("==================================================")
	fmt.Printf("   Modo Autônomo (Write/Fix/Search): %v\n\n", o.Autonomo)

	snapshot := system.ColetarContextoAmbiente()
	historicoGlobalInicial := &models.HistoricoAgente{NomeAgente: "Interprete"}
	perfil, err := o.executarAgente0a(snapshot, historicoGlobalInicial)
	if err != nil {
		fmt.Printf("❌ Falha crítica ao ler ambiente inicial: %v\n", err)
		return
	}

	for {
		entrada := system.LerEntradaDoUsuario()
		if entrada == "" {
			continue
		}

		if strings.ToLower(entrada) == "/bye" {
			fmt.Println("\n👋 Sessão encerrada. Memória limpa. Até logo!")
			break
		}

		historicoGlobal := &models.HistoricoAgente{NomeAgente: "Interprete"}
		missao, err := o.executarAgente0b(perfil, entrada, historicoGlobal)
		if err != nil {
			fmt.Printf("   ⚠️ Falha de comunicação com a IA: %v\n", err)
			continue
		}

		historicoInvestigador := &models.HistoricoAgente{NomeAgente: "Investigador"}
		dossie, err := o.executarAgente1(missao, perfil, historicoInvestigador)
		if err != nil {
			fmt.Printf("   ⚠️ Investigação interrompida por erro: %v\n", err)
			continue
		}

		analise, err := o.executarAgente2(missao, perfil, dossie, historicoInvestigador)
		if err != nil {
			fmt.Printf("   ⚠️ Erro ao formular análise estruturada: %v\n", err)
			continue
		}

		if analise.TemCorrecao {
			deveExecutar := o.Autonomo
			if !o.Autonomo {
				mensagem := fmt.Sprintf("Plano de ação crítico:\n\n%s\n\nDeseja executar?", analise.Plano)
				deveExecutar = system.LerConfirmacaoUsuario(mensagem)
			}

			if deveExecutar {
				err := o.executarAgente3_Reparador(perfil, analise.Plano)
				if err != nil {
					fmt.Printf("   ⚠️ Reparo interrompido por erro: %v\n", err)
				}
			} else {
				fmt.Println("\n   🚫 Execução cancelada.")
			}
		} else {
			fmt.Println("\n   ✅ Análise concluída. Sem correções pendentes.")
		}

		o.MemoriaSessao += fmt.Sprintf("-> Usuário: %s\n-> Resultado: %s\n\n",
			entrada,
			strings.ReplaceAll(models.Truncar(analise.Explicacao, 150), "\n", " "))

		fmt.Println("\n--------------------------------------------------")
	}
}

func (o *Orquestrador) executarAgente0a(snapshot string, hist *models.HistoricoAgente) (string, error) {
	fmt.Println("\n🌐 [AGENTE 0a] Analisando ambiente inicial...")
	prompt := fmt.Sprintf(`Analise os dados brutos abaixo e produza um PERFIL DE AMBIENTE de forma estrita e direta. Sem introduções.

=== DADOS BRUTOS ===
%s

SISTEMA OPERACIONAL:
ARQUITETURA:
AMBIENTE ESPECIAL:
SHELL PADRÃO:
USUÁRIO E DIRETÓRIO:
FERRAMENTAS CONFIRMADAS:
RESTRIÇÕES IDENTIFICADAS:`, snapshot)

	resp, err := llm.EnviarParaOllama(o.UrlOllama, models.RequestOllama{
		Model: o.Modelo, Stream: false, Messages: []models.Mensagem{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	perfil := llm.LimparConteudo(resp.Message.Content)
	hist.Adicionar(models.RegistroPasso{Passo: 1, Acao: "analisar", Dados: "snapshot", Resultado: perfil, Timestamp: agora()})
	return perfil, nil
}

func (o *Orquestrador) executarAgente0b(perfil, entrada string, hist *models.HistoricoAgente) (string, error) {
	fmt.Println("🧠 [AGENTE 0b] Formulando missão contextualizada...")
	prompt := fmt.Sprintf(`Reescreva a intenção do usuário como um objetivo técnico conciso e acionável. Zero enrolação.

=== MEMÓRIA DAS INTERAÇÕES ===
%s
(Use o contexto acima APENAS se a nova entrada for ambígua ou referenciar o passado).

=== PERFIL ===
%s
=== ENTRADA ===
"%s"`, o.MemoriaSessao, perfil, entrada)

	resp, err := llm.EnviarParaOllama(o.UrlOllama, models.RequestOllama{
		Model: o.Modelo, Stream: false, Messages: []models.Mensagem{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	missao := llm.LimparConteudo(resp.Message.Content)
	hist.Adicionar(models.RegistroPasso{Passo: 2, Acao: "formular", Dados: entrada, Resultado: missao, Timestamp: agora()})
	return missao, nil
}

func (o *Orquestrador) executarAgente1(missao, perfil string, hist *models.HistoricoAgente) (string, error) {
	fmt.Println("🔎 [AGENTE 1] Iniciando investigação de estado...")
	dossie, passo := "", 1

	for passo <= MAX_PASSOS_INVESTIGACAO {
		prompt := fmt.Sprintf(`Sua função é inspecionar o estado REAL do sistema para cumprir a missão.

=== MISSÃO ===
%s
=== HISTÓRICO ===
%s

REGRAS DE COMPORTAMENTO (CRÍTICO):
1. PROIBIDO TER PREGUIÇA: Nunca encerre com a ação "analisar" no seu primeiro passo. Você DEVE mapear o ambiente rodando comandos como 'ls -la', 'find .', 'cat', etc.
2. Se a missão envolver código ou problemas do sistema, vasculhe o diretório atual em busca de arquivos relevantes e leia o conteúdo deles.
3. Não altere nada, use apenas comandos de leitura.
4. Responda OBRIGATORIAMENTE usando o formato markdown JSON abaixo:

%sjson
{
  "raciocinio": "preciso listar os arquivos para entender o projeto",
  "acao": "comando",
  "dados": "ls -la"
}
%s`, missao, hist.Formatar(), "```", "```")

		resp, err := llm.EnviarParaOllama(o.UrlOllama, models.RequestOllama{
			Model: o.Modelo, Stream: false, Messages: []models.Mensagem{{Role: "user", Content: prompt}},
		})
		if err != nil {
			return dossie, err
		}

		decisao, err := llm.ExtrairJSON(resp.Message.Content)
		if err != nil {
			hist.Adicionar(models.RegistroPasso{Passo: passo, Acao: "erro_json", Resultado: err.Error(), Timestamp: agora()})
			passo++
			continue
		}

		// Trava de segurança no código: se ele tentar analisar no passo 1, a gente força um erro e manda ele tentar de novo
		if decisao.Acao == "analisar" && passo == 1 {
			erroForcado := "ERRO DE PROCESSO: Você não pode usar 'analisar' no passo 1. Execute um 'comando' para ler/listar arquivos primeiro."
			hist.Adicionar(models.RegistroPasso{Passo: passo, Acao: "comando", Dados: decisao.Dados, Resultado: erroForcado, Timestamp: agora()})
			passo++
			continue
		}

		if decisao.Acao == "analisar" {
			dossie += fmt.Sprintf("\n[CONCLUSÃO DA LEITURA]\n%s\n", decisao.Dados)
			break
		}

		resultado := ""
		if decisao.Acao == "comando" {
			resultado = system.ExecutarComando(decisao.Dados)
		} else if decisao.Acao == "pesquisar" {
			resultado = system.ExecutarPesquisa(decisao.Dados)
		}

		hist.Adicionar(models.RegistroPasso{Passo: passo, Acao: decisao.Acao, Dados: decisao.Dados, Resultado: resultado, Timestamp: agora()})
		dossie += fmt.Sprintf("\n[%s: %s]\n%s\n", strings.ToUpper(decisao.Acao), decisao.Dados, resultado)
		passo++
	}
	return dossie, nil
}

func (o *Orquestrador) executarAgente2(missao, perfil, dossie string, hist *models.HistoricoAgente) (models.AnaliseProblema, error) {
	fmt.Println("🧠 [AGENTE 2] Analisando criticamente e traçando plano...")

	// PROTEÇÃO CONTRA ESTOURO DE MEMÓRIA (Context Window)
	// Limita o dossiê para garantir que as instruções do prompt não sejam ignoradas.
	dossieSeguro := models.Truncar(dossie, 12000)

	prompt := fmt.Sprintf(`Você é um Engenheiro Sênior Decisor. Sua tarefa é determinar se o sistema precisa sofrer MUTAÇÃO (criação de arquivos, edição de código, instalação de pacotes).

=== MISSÃO ===
%s
=== DOSSIÊ DO ESTADO REAL ===
%s

REGRAS CRÍTICAS DE FORMATAÇÃO (SEU JSON VAI QUEBRAR SE VOCÊ IGNORAR):
1. Responda OBRIGATORIAMENTE com um bloco markdown JSON.
2. DENTRO DO JSON: Não use quebras de linha reais (Enter) dentro das strings. Se precisar quebrar linha, digite literalmente \\n.
3. Não use aspas duplas internas dentro das strings. Use apenas aspas simples (').

Formato EXATO esperado:
%sjson
{
  "explicacao": "Diagnóstico do estado real. Use aspas simples (') se precisar citar trechos.",
  "tem_correcao": true,
  "plano": "Comandos exatos (use \\n para separar múltiplos comandos, sem pular de linha fisicamente)."
}
%s`, missao, dossieSeguro, "```", "```")

	resp, err := llm.EnviarParaOllama(o.UrlOllama, models.RequestOllama{
		Model: o.Modelo, Stream: false, Messages: []models.Mensagem{{Role: "user", Content: prompt}},
	})
	if err != nil {
		return models.AnaliseProblema{}, err
	}

	analise, err := llm.ExtrairJSONAnalise(resp.Message.Content)
	if err != nil {
		return models.AnaliseProblema{Explicacao: "Falha ao gerar plano estruturado.", TemCorrecao: false}, err
	}

	fmt.Println("==================================================")
	fmt.Println("RELATÓRIO:")
	fmt.Println(models.Indent(models.Truncar(analise.Explicacao, 1000)))
	fmt.Println("==================================================")
	return analise, nil
}

func (o *Orquestrador) executarAgente3_Reparador(perfil, plano string) error {
	fmt.Println("\n🛠️  [AGENTE 3] Iniciando Reparador (Write/Fix/Search)...")
	historicoReparador := &models.HistoricoAgente{NomeAgente: "Reparador"}
	passo := 1

	for passo <= MAX_PASSOS_REPARO {
		prompt := fmt.Sprintf(`Execute e VERIFIQUE a correção. Você tem alta autonomia.

=== PLANO ===
%s
=== HISTÓRICO ===
%s

REGRAS CRÍTICAS:
1. Modifique arquivos usando ferramentas de terminal.
2. VERIFIQUE a modificação rodando o código.
3. Responda OBRIGATORIAMENTE usando o bloco markdown JSON.

%sjson
{
  "raciocinio": "foco no erro",
  "acao": "comando|pesquisar|analisar",
  "dados": "..."
}
%s`, plano, historicoReparador.Formatar(), "```", "```")

		resp, err := llm.EnviarParaOllama(o.UrlOllama, models.RequestOllama{
			Model: o.Modelo, Stream: false, Messages: []models.Mensagem{{Role: "user", Content: prompt}},
		})
		if err != nil {
			return err
		}

		decisao, err := llm.ExtrairJSON(resp.Message.Content)
		if err != nil {
			historicoReparador.Adicionar(models.RegistroPasso{Passo: passo, Acao: "erro_json", Resultado: err.Error(), Timestamp: agora()})
			passo++
			continue
		}

		fmt.Printf("   [%d/%d] %s: %s\n", passo, MAX_PASSOS_REPARO, strings.ToUpper(decisao.Acao), decisao.Dados)

		if decisao.Acao == "analisar" {
			fmt.Println("\n✅ CONCLUSÃO DO REPARO:")
			fmt.Println(models.Indent(decisao.Dados))
			break
		}

		resultado := ""
		if decisao.Acao == "comando" {
			resultado = system.ExecutarComando(decisao.Dados)
		} else if decisao.Acao == "pesquisar" {
			resultado = system.ExecutarPesquisa(decisao.Dados)
		} else {
			resultado = fmt.Sprintf("Ação inválida: %s", decisao.Acao)
		}

		historicoReparador.Adicionar(models.RegistroPasso{Passo: passo, Acao: decisao.Acao, Dados: decisao.Dados, Resultado: resultado, Timestamp: agora()})
		passo++
	}

	if passo > MAX_PASSOS_REPARO {
		fmt.Printf("\n   ⚠️ Reparador esgotou %d tentativas.\n", MAX_PASSOS_REPARO)
	}
	return nil
}
