package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"mid_night/models"
)

var reThink = regexp.MustCompile(`(?s)<think>.*?</think>`)
// Expressão regular para capturar blocos markdown de json
var reBlocoJSON = regexp.MustCompile(`(?s)\x60\x60\x60(?:json)?\s*(.*?)\s*\x60\x60\x60`)

type respostaOllamaRaw struct {
	Message struct {
		Role     string `json:"role"`
		Content  string `json:"content"`
		Thinking string `json:"thinking"`
	} `json:"message"`
}

func LimparConteudo(s string) string {
	s = reThink.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func ExtrairJSON(raw string) (models.Decisao, error) {
	limpo := LimparConteudo(raw)
	var d models.Decisao

	// 1. Tenta capturar usando o bloco markdown (mais seguro)
	matches := reBlocoJSON.FindStringSubmatch(limpo)
	if len(matches) > 1 {
		if err := json.Unmarshal([]byte(matches[1]), &d); err == nil {
			return d, nil
		}
	}

	// 2. Fallback: procura da primeira chave até a última
	primeiro := strings.Index(limpo, "{")
	ultimo := strings.LastIndex(limpo, "}")
	if primeiro != -1 && ultimo != -1 && ultimo >= primeiro {
		if err := json.Unmarshal([]byte(limpo[primeiro:ultimo+1]), &d); err == nil {
			return d, nil
		}
	}

	// 3. Tenta parsear direto como último recurso
	err := json.Unmarshal([]byte(limpo), &d)
	if err != nil {
		return d, fmt.Errorf("parse do JSON falhou: %w", err)
	}
	return d, nil
}

func ExtrairJSONAnalise(raw string) (models.AnaliseProblema, error) {
	limpo := LimparConteudo(raw)

	type TempAnalise struct {
		Explicacao  string      `json:"explicacao"`
		TemCorrecao bool        `json:"tem_correcao"`
		Plano       interface{} `json:"plano"`
	}

	parseTemp := func(data string) (models.AnaliseProblema, error) {
		var temp TempAnalise
		if err := json.Unmarshal([]byte(data), &temp); err != nil {
			return models.AnaliseProblema{}, err
		}

		var planoStr string
		switch v := temp.Plano.(type) {
		case string:
			planoStr = v
		case []interface{}:
			var linhas []string
			for _, linha := range v {
				linhas = append(linhas, fmt.Sprintf("%v", linha))
			}
			planoStr = strings.Join(linhas, "\n")
		case nil:
			planoStr = ""
		default:
			planoStr = fmt.Sprintf("%v", v)
		}

		return models.AnaliseProblema{
			Explicacao:  temp.Explicacao,
			TemCorrecao: temp.TemCorrecao,
			Plano:       planoStr,
		}, nil
	}

	// 1. Tenta capturar do bloco markdown
	matches := reBlocoJSON.FindStringSubmatch(limpo)
	if len(matches) > 1 {
		if result, err := parseTemp(matches[1]); err == nil {
			return result, nil
		}
	}

	// 2. Fallback: delimitação por chaves extremas
	primeiro := strings.Index(limpo, "{")
	ultimo := strings.LastIndex(limpo, "}")
	if primeiro != -1 && ultimo != -1 && ultimo >= primeiro {
		if result, err := parseTemp(limpo[primeiro : ultimo+1]); err == nil {
			return result, nil
		}
	}

	// 3. Parse direto se tudo falhar (AQUI ESTÁ A MÁGICA DE DEBUG)
	result, err := parseTemp(limpo)
	if err != nil {
		erroDetalhado := fmt.Errorf("falha no parse: %w\n\n=== TEXTO CRU GERADO PELA IA ===\n%s\n================================", err, limpo)
		return models.AnaliseProblema{}, erroDetalhado
	}
	
	return result, nil
}

func EnviarParaOllama(urlOllama string, reqBody models.RequestOllama) (models.ResponseOllama, error) {
	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", urlOllama, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.ResponseOllama{}, fmt.Errorf("erro de rede/timeout: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return models.ResponseOllama{}, fmt.Errorf("erro da API (Status %d): %s", resp.StatusCode, string(body))
	}

	if os.Getenv("DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "\n[DEBUG] Ollama raw response (%d bytes):\n%s\n", len(body), models.Truncar(string(body), 1000))
	}

	var raw respostaOllamaRaw
	json.Unmarshal(body, &raw)

	content := raw.Message.Content
	if strings.TrimSpace(content) == "" && raw.Message.Thinking != "" {
		content = raw.Message.Thinking
	}
	if strings.TrimSpace(content) == "" {
		reContent := regexp.MustCompile(`"content"\s*:\s*"((?:[^"\\]|\\.)*)"`)
		if m := reContent.FindSubmatch(body); len(m) > 1 {
			var decoded string
			json.Unmarshal([]byte(`"`+string(m[1])+`"`), &decoded)
			content = decoded
		}
	}

	return models.ResponseOllama{
		Message: models.Mensagem{
			Role:    raw.Message.Role,
			Content: content,
		},
	}, nil
}
