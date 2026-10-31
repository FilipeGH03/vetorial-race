package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"vetorial-race/internal/ia"
)

func TestRodarGravaOCerebro(t *testing.T) {
	saida := filepath.Join(t.TempDir(), "sub", "oval.json")
	if err := rodar("../../pistas/oval.json", 2, 1, saida, "14,6,2", 8); err != nil {
		t.Fatal(err)
	}
	r, err := ia.CarregarRede(saida)
	if err != nil {
		t.Fatal(err)
	}
	if r.Pista != "oval" || !reflect.DeepEqual(r.Camadas, []int{14, 6, 2}) {
		t.Fatalf("cérebro gravado: pista %q, camadas %v", r.Pista, r.Camadas)
	}
}

func TestRodarArgumentosInvalidos(t *testing.T) {
	dir := t.TempDir()
	casos := []struct {
		nome, pista, camadas string
		geracoes, pop        int
	}{
		{"zero gerações", "../../pistas/oval.json", "14,12,2", 0, 8},
		{"camadas não numéricas", "../../pistas/oval.json", "14,x,2", 1, 8},
		{"entrada errada", "../../pistas/oval.json", "9,12,2", 1, 8},
		{"entrada antiga (antes da D20)", "../../pistas/oval.json", "10,12,2", 1, 8},
		{"pista inexistente", "../../pistas/nao-existe.json", "14,12,2", 1, 8},
		{"população pequena", "../../pistas/oval.json", "14,12,2", 1, 1},
	}
	for _, c := range casos {
		if err := rodar(c.pista, c.geracoes, 1, filepath.Join(dir, "x.json"), c.camadas, c.pop); err == nil {
			t.Errorf("%s: deveria dar erro", c.nome)
		}
	}
}

func TestSepararEJuntar(t *testing.T) {
	c, err := separar(" 10, 16 ,8,2")
	if err != nil || !reflect.DeepEqual(c, []int{10, 16, 8, 2}) {
		t.Fatalf("separar: %v, %v", c, err)
	}
	if s := juntar(ia.CamadasPadrao); s != "14,12,2" {
		t.Fatalf("juntar: %q", s)
	}
}
