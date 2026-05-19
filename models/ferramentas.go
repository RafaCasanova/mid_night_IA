package models

type CategoriaFerramenta struct {
	Nome        string
	Ferramentas []string
}

// CategoriasDeFerramentas guarda o pré-carregamento dos comandos 
// que o sistema tentará detectar no ambiente hospedeiro.
var CategoriasDeFerramentas = []CategoriaFerramenta{
	{"Shell/core", []string{"bash", "sh", "zsh", "fish", "dash"}},
	{"Texto/busca", []string{"grep", "awk", "sed", "find", "cut", "sort", "uniq", "tr", "head", "tail", "wc", "xargs"}},
	{"Rede", []string{"curl", "wget", "nc", "netstat", "ss", "nslookup", "dig", "ping"}},
	{"Dados", []string{"jq", "yq", "xmllint", "python3", "python", "perl", "ruby"}},
	{"Dev/build", []string{"git", "make", "gcc", "clang", "go", "rustc", "javac", "node", "npm", "pip", "pip3"}},
	{"Sistema", []string{"ps", "top", "df", "du", "lsof", "strace", "ltrace", "systemctl", "service", "cron"}},
	{"Compressão", []string{"tar", "gzip", "zip", "unzip", "bzip2", "xz", "7z"}},
}
