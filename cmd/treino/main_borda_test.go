package main

import (
	"os"
	"path/filepath"
	"testing"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

// Mesma seed, mesmo arquivo de cérebro, byte a byte.
func TestRodarMesmaSeedMesmoArquivo(t *testing.T) {
	dir := t.TempDir()
	var arqs [2][]byte
	for i := range arqs {
		s := filepath.Join(dir, "x"+string(rune('a'+i))+".json")
		if err := rodar("../../pistas/oval.json", 3, 7, s, "14,8,2", 10); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		arqs[i] = b
	}
	if string(arqs[0]) != string(arqs[1]) {
		t.Fatal("mesma seed gerou cérebros diferentes")
	}
}

// Sem -saida o cérebro vai para cerebros/<arquivo da pista sem extensão>.json
// (contrato 1.4), a partir da pasta onde o comando roda, mesmo quando o
// "nome" dentro do JSON da pista é outro.
func TestRodarSaidaPadrao(t *testing.T) {
	dados, err := os.ReadFile("../../pistas/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile("minha-pista.json", dados, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := rodar("minha-pista.json", 1, 1, "", "14,6,2", 6); err != nil {
		t.Fatal(err)
	}
	r, err := ia.CarregarRede(filepath.Join("cerebros", "minha-pista.json"))
	if err != nil {
		t.Fatal(err)
	}
	if r.Pista != "oval" {
		t.Errorf("pista = %q", r.Pista)
	}
	if _, err := os.Stat(filepath.Join("cerebros", "oval.json")); err == nil {
		t.Error("gravou pelo nome de dentro da pista, não pelo nome do arquivo")
	}
}

// A saída padrão usa só o nome do arquivo: '/' ou '..' no caminho da pista
// nunca levam o cérebro para fora de cerebros/.
func TestSaidaPadraoFicaEmCerebros(t *testing.T) {
	casos := map[string]string{
		"pistas/oval.json":        filepath.Join("cerebros", "oval.json"),
		"../../pistas/oval.json":  filepath.Join("cerebros", "oval.json"),
		"a/../../b/curva.json":    filepath.Join("cerebros", "curva.json"),
		"pistas/sem-extensao":     filepath.Join("cerebros", "sem-extensao.json"),
		"pistas/dois.pontos.json": filepath.Join("cerebros", "dois.pontos.json"),
	}
	for pista, want := range casos {
		got, err := saidaPadrao(pista)
		if err != nil || got != want {
			t.Errorf("saidaPadrao(%q) = %q, %v; esperava %q", pista, got, err, want)
		}
	}
	for _, pista := range []string{"", ".json", "pistas/.json", "..", "pistas/..", "pistas/...json"} {
		if got, err := saidaPadrao(pista); err == nil {
			t.Errorf("saidaPadrao(%q) = %q, deveria dar erro", pista, got)
		}
	}
}

// O cérebro gravado guarda a melhor nota e ela se reproduz no jogo.
func TestRodarNotaGravadaReproduz(t *testing.T) {
	saida := filepath.Join(t.TempDir(), "o.json")
	if err := rodar("../../pistas/oval.json", 3, 1, saida, "14,6,2", 8); err != nil {
		t.Fatal(err)
	}
	r, err := ia.CarregarRede(saida)
	if err != nil {
		t.Fatal(err)
	}
	p, err := sim.CarregarPista("../../pistas/oval.json")
	if err != nil {
		t.Fatal(err)
	}
	tr, err := ia.NovoTreino(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.Rodar(r).Pontuacao; got != r.Pontuacao {
		t.Errorf("rodando o cérebro dá %v, gravado %v", got, r.Pontuacao)
	}
	if r.Pontuacao <= 0 {
		t.Errorf("pontuação gravada %v", r.Pontuacao)
	}
}

func TestSepararInvalido(t *testing.T) {
	for _, s := range []string{"", "10,,2", "10;12;2", "10,1.5,2"} {
		if _, err := separar(s); err == nil {
			t.Errorf("separar(%q) deveria dar erro", s)
		}
	}
}

// Camadas mal formadas (saída 3, só 1 camada, neurônio 0) param antes de treinar.
func TestRodarCamadasInvalidas(t *testing.T) {
	saida := filepath.Join(t.TempDir(), "x.json")
	for _, c := range []string{"14,12,3", "14", "14,0,2", "14,-4,2", "10,12,2"} {
		if err := rodar("../../pistas/oval.json", 1, 1, saida, c, 8); err == nil {
			t.Errorf("camadas %q deveriam dar erro", c)
		}
	}
	if _, err := os.Stat(saida); err == nil {
		t.Error("gravou cérebro mesmo com erro")
	}
}
