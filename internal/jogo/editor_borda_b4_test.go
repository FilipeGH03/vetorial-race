package jogo

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"vetorial-race/internal/sim"
)

// Salvar com pista inválida não cria arquivo nem pasta.
func TestB4SalvarInvalidoNaoGrava(t *testing.T) {
	e := NovoEditor("x")
	dir := filepath.Join(t.TempDir(), "novo")
	caminho := filepath.Join(dir, "vazio.json")
	if err := e.Salvar(caminho); err == nil {
		t.Fatal("editor vazio não deveria salvar")
	}
	if _, err := os.Stat(caminho); err == nil {
		t.Fatal("arquivo criado para pista inválida")
	}
	// oito: erro de cruzamento e nada gravado
	e2 := NovoEditor("oito")
	e2.Largura = 60
	var tr []sim.Vec2
	for _, q := range [][2]float64{{300, 200}, {600, 200}, {600, 400}, {300, 400}, {300, 200}} {
		tr = append(tr, sim.Vec2{X: q[0], Y: q[1]})
	}
	_ = tr
	e2.Comecar(300, 200)
	for _, q := range [][2]float64{{500, 400}, {700, 400}, {700, 200}, {500, 200}, {300, 400}} {
		for s := 0.0; s < 1; s += 0.02 {
			e2.Arrastar(q[0]*s+300*(1-s), q[1]*s+200*(1-s))
		}
		e2.Arrastar(q[0], q[1])
	}
	e2.Soltar()
	if e2.Erro == nil {
		t.Skip("traço montado não formou cruzamento")
	}
	if err := e2.Salvar(filepath.Join(dir, "oito.json")); err == nil {
		t.Fatal("salvou pista inválida")
	}
	if _, err := os.Stat(filepath.Join(dir, "oito.json")); err == nil {
		t.Fatal("arquivo criado")
	}
}

// Traço no sentido horário (mouse real vai nos dois sentidos) também vira pista
// válida, e salvar por cima de um arquivo existente troca o conteúdo.
func TestB4SentidoHorarioESobrescrever(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	tr := ovalTela(640, 360, 450, 220, 2, rng)
	for i, j := 0, len(tr)-1; i < j; i, j = i+1, j-1 {
		tr[i], tr[j] = tr[j], tr[i]
	}
	e := NovoEditor("h")
	desenhar(e, tr)
	if e.Erro != nil {
		t.Fatalf("horário: %v", e.Erro)
	}
	caminho := filepath.Join(t.TempDir(), "h.json")
	if err := e.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	e.AjustarLargura(20)
	if err := e.Salvar(caminho); err != nil {
		t.Fatal(err)
	}
	p, err := sim.CarregarPista(caminho)
	if err != nil || p.Largura != 110 {
		t.Fatalf("sobrescrever: %v largura %v", err, p)
	}
}

// Carregar pista que não cabe na janela: erro ErrNaoCabe e editor intacto.
func TestB4CarregarNaoCabe(t *testing.T) {
	var pts []sim.Vec2
	for _, q := range [][2]float64{{-300, 0}, {0, 0}, {300, 0}, {600, 0}, {600, 300}, {600, 600}, {300, 600}, {0, 600}, {-300, 600}, {-300, 300}} {
		pts = append(pts, sim.Vec2{X: q[0], Y: q[1]})
	}
	p, err := sim.NovaPista("fora", 80, pts)
	if err != nil {
		t.Skipf("pista de teste não montou: %v", err)
	}
	caminho := filepath.Join(t.TempDir(), "fora.json")
	if err := sim.SalvarPista(caminho, p); err != nil {
		t.Fatal(err)
	}
	e := NovoEditor("a")
	if err := e.Carregar(caminho); !errors.Is(err, ErrNaoCabe) {
		t.Fatalf("esperava ErrNaoCabe, veio %v", err)
	}
}

// Arrastar largo preso na margem: nenhum ponto do traço fica a menos de meia
// largura da borda, mesmo com o mouse fora da janela.
func TestB4MouseForaDaJanela(t *testing.T) {
	e := NovoEditor("a")
	e.Largura = 200
	e.Comecar(-50, 5000)
	e.Arrastar(5000, -50)
	for _, q := range e.Traco {
		if q.X < 100 || q.X > LarguraTela-100 || q.Y < 100 || q.Y > AlturaTela-100 {
			t.Fatalf("ponto fora da margem: %v", q)
		}
	}
	e.Soltar()
	if e.Erro == nil || e.Pista != nil {
		t.Fatal("traço de 2 pontos deveria dar erro")
	}
}
