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

// systemPromptBase é o contexto fixo injetado como PRIMEIRO elemento
// em TODAS as conversas com a LLM, antes de qualquer mensagem do usuário.
// Isso resolve o problema de alucinação: a IA sempre sabe onde está e o que é.

const systemPromptBase = `Você é um Agente Autónomo e Engenheiro de Sistemas Sénior do projeto mid_night_IA.
A sua função é operar via terminal para orquestrar, auditar e reparar ambientes de software.

REGRAS ABSOLUTAS — NUNCA VIOLE:
1. Você opera EXCLUSIVAMENTE dentro do diretório do projeto atual, a não ser em casos extremos, e isso tem que ter a permissão do usuário.
2. NUNCA modifique ou leia ficheiros do sistema (~/.bashrc, /etc/*, etc.) fora do escopo do projeto.
3. NUNCA execute comandos destrutivos sem confirmação explícita.
4. NUNCA assuma ou "alucine" a inexistência de um ficheiro/função sem antes realizar uma busca exaustiva (ex: find, grep recursivo).
5. CONFIE APENAS NAS PROVAS: Se um comando falhar ou retornar vazio, não desista. Formule uma nova hipótese e tente outro comando ou estratégia.
6. Ao investigar, consulte sempre o histórico desta conversa para evitar loops infinitos e passos repetidos.`

type Orquestrador struct {
	Modelo    string
	UrlOllama string
	Autonomo  bool
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

	// Histórico de sessão: acumula user/assistant ao longo da sessão inteira.
	// O system prompt é sempre o índice 0 — a IA nunca "esquece" o contexto.
	histSessao := models.NovoHistoricoChat(systemPromptBase)

	perfil, err := o.executarAgente0a(snapshot, histSessao)
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
			fmt.Println("\n👋 Sessão encerrada. Até logo!")
			break
		}

		// Cada agente recebe o histSessao, garantindo que a instrução original
		// e todo o contexto anterior estejam presentes na chamada à LLM.
		missao, err := o.executarAgente0b(perfil, entrada, histSessao)
		if err != nil {
			fmt.Printf("   ⚠️ Falha de comunicação com a IA: %v\n", err)
			continue
		}

		// O Agente 1 usa um histórico próprio para seus passos de investigação,
		// mas recebe a missão e o perfil que já foram estabelecidos no histSessao.
		histInvestigador := models.NovoHistoricoChat(fmt.Sprintf(`%s

CONTEXTO DA SESSÃO ATUAL:
- Perfil do ambiente: %s
- Missão a cumprir: %s

Você é o Agente Investigador. Leia o ambiente, não modifique nada.`, systemPromptBase, perfil, missao))

		dossie, err := o.executarAgente1(missao, histInvestigador)
		if err != nil {
			fmt.Printf("   ⚠️ Investigação interrompida por erro: %v\n", err)
			continue
		}

		analise, err := o.executarAgente2(missao, perfil, dossie, histSessao)
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
				histReparador := models.NovoHistoricoChat(fmt.Sprintf(`%s

CONTEXTO DA SESSÃO ATUAL:
- Perfil do ambiente: %s
- Plano aprovado para execução: %s

Você é o Agente Reparador. Execute o plano, verifique o resultado.`, systemPromptBase, perfil, analise.Plano))

				err := o.executarAgente3_Reparador(analise.Plano, histReparador)
				if err != nil {
					fmt.Printf("   ⚠️ Reparo interrompido por erro: %v\n", err)
				}
			} else {
				fmt.Println("\n   🚫 Execução cancelada.")
			}
		} else {
			fmt.Println("\n   ✅ Análise concluída. Sem correções pendentes.")
		}

		fmt.Println("\n--------------------------------------------------")
	}
}

// executarAgente0a analisa o ambiente inicial.
// Usa histSessao para que o resultado fique registrado para os próximos agentes.
func (o *Orquestrador) executarAgente0a(snapshot string, hist *models.HistoricoChat) (string, error) {
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

	perfil, err := llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, prompt)
	if err != nil {
		return "", err
	}
	return perfil, nil
}

// executarAgente0b formula a missão técnica a partir da entrada do usuário.
// Usa histSessao — a IA já tem o perfil do ambiente e o contexto anterior.
func (o *Orquestrador) executarAgente0b(perfil, entrada string, hist *models.HistoricoChat) (string, error) {
	fmt.Println("🧠 [AGENTE 0b] Formulando missão contextualizada...")

	prompt := fmt.Sprintf(`Reescreva a intenção do usuário como um objetivo técnico conciso e acionável. Zero enrolação.

=== ENTRADA DO USUÁRIO ===
"%s"

(Use o histórico desta conversa se a entrada for ambígua ou referenciar algo anterior.)`, entrada)

	missao, err := llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, prompt)
	if err != nil {
		return "", err
	}
	return missao, nil
}

// executarAgente1 investiga o estado real do sistema.
// Usa histInvestigador — histórico próprio com system já contendo missão e perfil.
func (o *Orquestrador) executarAgente1(missao string, hist *models.HistoricoChat) (string, error) {
	fmt.Println("🔎 [AGENTE 1] Iniciando investigação de estado...")

	dossie := ""
	for passo := 1; passo <= MAX_PASSOS_INVESTIGACAO; passo++ {
		instrucao := `Inspecione o estado REAL do sistema para cumprir a missão. Você é um detetive técnico metódico.

ESTRATÉGIAS DE INVESTIGAÇÃO AVANÇADAS (Siga rigorosamente):
- Mapeamento Inicial: Para entender a estrutura do projeto, use 'find . -maxdepth 2 -type d' ou 'ls -laR' de forma contida.
- Busca Localizada: Use 'find . -type f -name "*nome_do_ficheiro*"' em vez de assumir que o ficheiro está na raiz.
- Extração de Código: Para procurar funções ou classes, NUNCA use 'cat' às cegas. Use OBRIGATORIAMENTE 'grep -rnw . -e "nome_da_funcao"' (busca recursiva com número de linha).
- Leitura Segura de Ficheiros: Se precisar ler o conteúdo, proteja a sua memória. Use 'head -n 50', 'tail' ou 'grep -n -C 10 "termo"' (para ver 10 linhas antes e depois do termo). Apenas use 'cat' em ficheiros que sabe serem pequenos.
- Resiliência: Se um 'grep' não encontrar nada, tente com a flag '-i' (case-insensitive) ou reduza os termos de pesquisa.

REGRAS DO PROCESSO:
1. O Passo 1 SEMPRE deve ser um comando de leitura ou mapeamento. Nunca use "analisar" no primeiro passo.
2. Não altere absolutamente nada. Use apenas comandos de leitura ('find', 'grep', 'ls', 'head', 'cat').
3. Responda OBRIGATORIAMENTE no formato JSON abaixo, SEM NENHUM TEXTO FORA DO BLOCO MARKDOWN:
` + "```json" + `
{
  "raciocinio": "A sua linha de pensamento. (ex: 'O ficheiro não estava na raiz, vou usar o comando find recursivamente')",
  "acao": "comando",
  "dados": "comando_de_terminal_exato_aqui"
}
` + "```" + `
4. Quando tiver evidências robustas e suficientes extraídas do terminal, mude a "acao" para "analisar" e coloque o seu diagnóstico final em "dados".`

		resposta, err := llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, instrucao)
		if err != nil {
			return dossie, err
		}

		decisao, err := llm.ExtrairJSON(resposta)
		if err != nil {
			passo++
			continue
		}

		// Trava: proíbe "analisar" no passo 1
		if decisao.Acao == "analisar" && passo == 1 {
			feedback := "ERRO DE PROCESSO: Você não pode usar 'analisar' no passo 1. Execute um 'comando' de leitura primeiro."
			llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, feedback)
			continue
		}

		if decisao.Acao == "analisar" {
			dossie += fmt.Sprintf("\n[CONCLUSÃO DA LEITURA]\n%s\n", decisao.Dados)
			break
		}

		resultado := ""
		if decisao.Acao == "comando" {
			resultado = system.ExecutarComando(decisao.Dados)
			fmt.Printf("   💻 [BASH] Executando: %s\n", decisao.Dados)
		} else if decisao.Acao == "pesquisar" {
			resultado = system.ExecutarPesquisa(decisao.Dados)
		}

		// Resultado do comando vai de volta como mensagem do usuário ("tool result")
		// para que a IA saiba o que aconteceu antes de decidir o próximo passo.
		feedbackComando := fmt.Sprintf("Resultado do comando `%s`:\n%s", decisao.Dados, resultado)
		llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, feedbackComando)

		dossie += fmt.Sprintf("\n[%s: %s]\n%s\n", strings.ToUpper(decisao.Acao), decisao.Dados, resultado)
	}

	return dossie, nil
}

// executarAgente2 analisa o dossiê e decide se há correção a fazer.
// Usa histSessao para registrar o diagnóstico na memória da sessão.
func (o *Orquestrador) executarAgente2(missao, perfil, dossie string, hist *models.HistoricoChat) (models.AnaliseProblema, error) {
	fmt.Println("🧠 [AGENTE 2] Analisando criticamente e traçando plano...")

	dossieSeguro := models.Truncar(dossie, 12000)

	prompt := fmt.Sprintf(`Você é um Engenheiro Sênior Decisor. Determine se o sistema precisa de MUTAÇÃO (edição de código, arquivos, pacotes).

=== DOSSIÊ DO ESTADO REAL ===
%s

REGRAS CRÍTICAS DE FORMATAÇÃO:
1. Responda OBRIGATORIAMENTE com um bloco markdown JSON.
2. DENTRO DO JSON: use \n para quebras de linha, não Enter literal.
3. Use apenas aspas simples (') dentro das strings, nunca aspas duplas.

Formato EXATO:
`+"```json"+`
{
  "explicacao": "Diagnóstico objetivo. Use aspas simples se precisar citar.",
  "tem_correcao": true,
  "plano": "Comandos exatos separados por \n"
}
`+"```", dossieSeguro)

	resposta, err := llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, prompt)
	if err != nil {
		return models.AnaliseProblema{}, err
	}

	analise, err := llm.ExtrairJSONAnalise(resposta)
	if err != nil {
		return models.AnaliseProblema{Explicacao: "Falha ao gerar plano estruturado.", TemCorrecao: false}, err
	}

	fmt.Println("==================================================")
	fmt.Println("RELATÓRIO:")
	fmt.Println(models.Indent(models.Truncar(analise.Explicacao, 1000)))
	fmt.Println("==================================================")

	return analise, nil
}

// executarAgente3_Reparador executa o plano aprovado.
// Usa histReparador — histórico próprio com system já contendo o plano aprovado.
func (o *Orquestrador) executarAgente3_Reparador(plano string, hist *models.HistoricoChat) error {
	fmt.Println("\n🛠️  [AGENTE 3] Iniciando Reparador (Write/Fix/Search)...")

	for passo := 1; passo <= MAX_PASSOS_REPARO; passo++ {
		instrucao := `Execute e VERIFIQUE a correção conforme o plano no seu contexto.

REGRAS:
1. Modifique arquivos usando ferramentas de terminal.
2. VERIFIQUE a modificação após cada mudança.
3. Consulte o histórico desta conversa para não repetir passos.
4. Responda OBRIGATORIAMENTE no formato:
` + "```json" + `
{
  "raciocinio": "o que estou fazendo e por quê",
  "acao": "comando|pesquisar|analisar",
  "dados": "..."
}
` + "```"

		resposta, err := llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, instrucao)
		if err != nil {
			return err
		}

		decisao, err := llm.ExtrairJSON(resposta)
		if err != nil {
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

		// Devolve o resultado do comando para a IA saber o que aconteceu
		feedback := fmt.Sprintf("Resultado do comando `%s`:\n%s", decisao.Dados, resultado)
		llm.EnviarComHistorico(o.UrlOllama, o.Modelo, hist, feedback)
	}

	if false { // sentinela para evitar warning de loop sem break
		fmt.Printf("\n   ⚠️ Reparador esgotou %d tentativas.\n", MAX_PASSOS_REPARO)
	}

	return nil
}
