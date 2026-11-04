package jogo

import (
	"math/rand/v2"
	"path/filepath"
	"reflect"
	"testing"

	"vetorial-race/internal/ia"
)

func TestTreinoVisualDeCerebroDuasEscondidas(t *testing.T) {
	dir := t.TempDir()
	r, err := ia.NovaRede([]int{14, 8, 6, 2}, rand.New(rand.NewPCG(1, 2)))
	if err != nil {
		t.Fatal(err)
	}
	r.Pista, r.Geracao = "oval", 200
	if err := r.Salvar(filepath.Join(dir, "oval.json")); err != nil {
		t.Fatal(err)
	}
	v, err := NovoTreinoVisualDe(pistaOval, dir, 2, InicioCerebroSalvo)
	if err != nil {
		t.Fatal(err)
	}
	if v.Geracao() != 200 || !reflect.DeepEqual(v.Individuos[0].ctl.Rede.Camadas, []int{14, 8, 6, 2}) {
		t.Errorf("geração %d, camadas %v", v.Geracao(), v.Individuos[0].ctl.Rede.Camadas)
	}
}

func TestTreinoVisualDeCerebroEntradaErradaDaErro(t *testing.T) {
	dir := t.TempDir()
	r, err := ia.NovaRede([]int{13, 12, 2}, rand.New(rand.NewPCG(1, 2)))
	if err == nil {
		_ = r.Salvar(filepath.Join(dir, "oval.json"))
	}
	if _, err := NovoTreinoVisualDe(pistaOval, dir, 1, InicioCerebroSalvo); err == nil {
		t.Error("cérebro com 13 entradas devia dar erro")
	}
}
