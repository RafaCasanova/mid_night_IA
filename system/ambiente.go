package system

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"mid_night/models"
)

var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func LerEntradaDoUsuario() string {
	fmt.Print("\n✏️  Digite sua instrução e aperte Enter: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Erro ao ler entrada:", err)
	}
	return ""
}

func probeCmd(comando string) string {
	shell := "bash"
	if _, err := exec.LookPath("bash"); err != nil {
		shell = "sh"
	}
	cmd := exec.Command(shell, "-c", comando)
	out, _ := cmd.CombinedOutput()
	resultado := strings.TrimSpace(string(out))
	if resultado == "" {
		return "(sem saída)"
	}
	return resultado
}

func detectarSistemaOperacional() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("uname: %s\n", probeCmd("uname -a 2>/dev/null")))

	if v := probeCmd("cat /etc/os-release 2>/dev/null | grep -E '^(PRETTY_NAME|ID|VERSION_ID)=' | head -5"); v != "(sem saída)" {
		sb.WriteString(fmt.Sprintf("os-release: %s\n", v))
	}
	if v := probeCmd("cat /etc/alpine-release 2>/dev/null"); v != "(sem saída)" {
		sb.WriteString(fmt.Sprintf("alpine-release: %s\n", v))
	}
	if v := probeCmd("sw_vers 2>/dev/null"); v != "(sem saída)" {
		sb.WriteString(fmt.Sprintf("macOS: %s\n", v))
	}
	if v := probeCmd("cat /proc/version 2>/dev/null | head -1"); v != "(sem saída)" {
		sb.WriteString(fmt.Sprintf("proc/version: %s\n", v))
	}
	sb.WriteString(fmt.Sprintf("WSL: %s\n", probeCmd("grep -qi microsoft /proc/version 2>/dev/null && echo 'WSL detectado' || echo 'não é WSL'")))
	sb.WriteString(fmt.Sprintf("Container: %s\n", probeCmd("[ -f /.dockerenv ] && echo 'container Docker' || echo 'não é container'")))
	sb.WriteString(fmt.Sprintf("Arquitetura: %s\n", probeCmd("uname -m 2>/dev/null || arch 2>/dev/null")))

	return sb.String()
}

func detectarFerramentas() string {
	var sb strings.Builder
	for _, cat := range models.CategoriasDeFerramentas {
		var disponiveis []string
		for _, t := range cat.Ferramentas {
			out := probeCmd(fmt.Sprintf("command -v %s 2>/dev/null && echo __FOUND__ || echo __NOTFOUND__", t))
			if strings.Contains(out, "__FOUND__") {
				disponiveis = append(disponiveis, t)
			}
		}
		if len(disponiveis) > 0 {
			sb.WriteString(fmt.Sprintf("  [%s]: %s\n", cat.Nome, strings.Join(disponiveis, ", ")))
		} else {
			sb.WriteString(fmt.Sprintf("  [%s]: (nenhuma disponível)\n", cat.Nome))
		}
	}
	return sb.String()
}

func ColetarContextoAmbiente() string {
	fmt.Println("\n🔍 [SISTEMA] Detectando ambiente (cross-platform)...")
	var sb strings.Builder
	sb.WriteString("=== SNAPSHOT DO AMBIENTE ===\n\n")
	sb.WriteString("[SISTEMA OPERACIONAL]\n")
	sb.WriteString(detectarSistemaOperacional())

	sb.WriteString("\n[CONTEXTO DE EXECUÇÃO]\n")
	sb.WriteString(fmt.Sprintf("  usuário: %s\n", probeCmd("whoami 2>/dev/null || id -un 2>/dev/null")))
	sb.WriteString(fmt.Sprintf("  home: %s\n", os.Getenv("HOME")))
	sb.WriteString(fmt.Sprintf("  pwd: %s\n", probeCmd("pwd")))
	sb.WriteString(fmt.Sprintf("  shell: %s\n", probeCmd("echo ${SHELL:-desconhecido}")))
	sb.WriteString(fmt.Sprintf("  PATH: %s\n", os.Getenv("PATH")))

	sb.WriteString("\n[FERRAMENTAS DISPONÍVEIS]\n")
	sb.WriteString(detectarFerramentas())

	sb.WriteString("\n[ARQUIVOS NO DIRETÓRIO ATUAL]\n")
	arquivos := probeCmd("ls -lah 2>/dev/null || ls -la 2>/dev/null || find . -maxdepth 1 2>/dev/null | head -30")
	sb.WriteString(reANSI.ReplaceAllString(arquivos, "") + "\n")

	sb.WriteString("\n[VARIÁVEIS DE AMBIENTE RELEVANTES]\n")
	for _, v := range []string{"LANG", "LC_ALL", "TERM", "EDITOR", "GOPATH", "GOROOT", "JAVA_HOME"} {
		if val := os.Getenv(v); val != "" {
			sb.WriteString(fmt.Sprintf("  %s=%s\n", v, val))
		}
	}

	sb.WriteString("\n[RECURSOS DO SISTEMA]\n")
	sb.WriteString(fmt.Sprintf("  CPUs: %s\n", probeCmd("nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo desconhecido")))
	sb.WriteString(fmt.Sprintf("  Memória: %s\n", probeCmd("free -h 2>/dev/null | grep Mem | awk '{print \"total=\"$2\" usado=\"$3\" livre=\"$4}' || vm_stat 2>/dev/null | head -5")))
	sb.WriteString(fmt.Sprintf("  Disco (pwd): %s\n", probeCmd("df -h . 2>/dev/null | tail -1 | awk '{print \"total=\"$2\" usado=\"$3\" livre=\"$4}'")))

	sb.WriteString("\n=== FIM DO SNAPSHOT ===\n")
	fmt.Println("   ✅ Ambiente detectado.")
	return sb.String()
}

func ExecutarPesquisa(termo string) string {
	fmt.Printf("   🌐 [WEB] Pesquisando: '%s'...\n", termo)
	query := url.QueryEscape(termo)
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json", query)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Sprintf("Erro de rede: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	return string(bodyBytes)
}

func ExecutarComando(comando string) string {
	fmt.Printf("   💻 [BASH] Executando: %s\n", comando)
	cmd := exec.Command("bash", "-c", comando)
	out, err := cmd.CombinedOutput()
	resultado := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Sprintf("ERRO: %v\nSaída: %s", err, resultado)
	}
	if resultado == "" {
		return "Comando executado com sucesso (sem saída)."
	}
	if len(resultado) > 20000 {
		return resultado[:20000] + "\n[AVISO: Saída truncada — filtre melhor.]"
	}
	return resultado
}

func LerConfirmacaoUsuario(mensagem string) bool {
	fmt.Printf("\n⚠️  %s (s/N): ", strings.TrimSpace(mensagem))
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		resp := strings.ToLower(strings.TrimSpace(scanner.Text()))
		return resp == "s" || resp == "sim" || resp == "y" || resp == "yes"
	}
	return false
}
