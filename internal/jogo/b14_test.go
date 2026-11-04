package jogo

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"vetorial-race/internal/ia"
)

// cerebroDeTeste grava <dir>/oval.json com uma rede 14-12-2 na geração dada.
func cerebroDeTeste(t *testing.T, dir string, geracao int) *ia.Rede {
	t.Helper()
	r, err := ia.NovaRede(ia.CamadasPadrao, rand.New(rand.NewPCG(5, 6)))
	if err != nil {
		t.Fatal(err)
	}
	r.Pista, r.Geracao, r.Pontuacao = "oval", geracao, 1234.5
	if err := r.Salvar(filepath.Join(dir, "oval.json")); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestMenuAlternaInicioDoTreino(t *testing.T) {
	m := NovoMenu([]string{pistaOval}, "")
	if m.Inicio != InicioZero || !strings.Contains(m.Linhas()[CampoInicio], "zero") {
		t.Fatalf("padrão: %d %q", m.Inicio, m.Linhas()[CampoInicio])
	}
	m.Selecao = CampoInicio
	m.Direita()
	if m.Inicio != InicioCerebroSalvo || !strings.Contains(m.Linhas()[CampoInicio], "cérebro salvo") {
		t.Errorf("direita: %d %q", m.Inicio, m.Linhas()[CampoInicio])
	}
	m.Esquerda()
	if m.Inicio != InicioZero {
		t.Errorf("esquerda: %d", m.Inicio)
	}
	m.Baixo() // a linha nova é a última: o cursor volta à pista
	if m.Selecao != CampoPista {
		t.Errorf("cursor: %d", m.Selecao)
	}
}

func TestTreinoVisualDeCerebroSalvoComecaDoArquivo(t *testing.T) {
	dir := t.TempDir()
	base := cerebroDeTeste(t, dir, 350)
	v, err := NovoTreinoVisualDe(pistaOval, dir, 3, InicioCerebroSalvo)
	if err != nil {
		t.Fatal(err)
	}
	if v.Geracao() != 350 {
		t.Errorf("Geracao() = %d, quer 350", v.Geracao())
	}
	if got, want := v.PenasEmVigor(), ia.PenasDaGeracao(350); got != want || got != ia.PenasCheias {
		t.Errorf("penas %+v, quer %+v", got, want)
	}
	if len(v.Individuos) == 0 || v.Individuos[0].ctl == nil {
		t.Fatal("sem indivíduos")
	}
	r0 := v.Individuos[0].ctl.Rede
	if !reflect.DeepEqual(r0.Pesos, base.Pesos) || !reflect.DeepEqual(r0.Vieses, base.Vieses) || !reflect.DeepEqual(r0.Camadas, base.Camadas) {
		t.Error("Individuos[0] não dirige com a rede do arquivo")
	}
	if reflect.DeepEqual(v.Individuos[1].ctl.Rede.Pesos, base.Pesos) {
		t.Error("Individuos[1] devia ser uma mutação do cérebro")
	}
	// Mesma seed, mesmo resultado.
	w, err := NovoTreinoVisualDe(pistaOval, dir, 3, InicioCerebroSalvo)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(v.Individuos[1].ctl.Rede.Pesos, w.Individuos[1].ctl.Rede.Pesos) {
		t.Error("mesma seed deu população diferente")
	}
}

func TestTreinoVisualDeCerebroSalvoGeracaoNaRampa(t *testing.T) {
	dir := t.TempDir()
	cerebroDeTeste(t, dir, 150)
	v, err := NovoTreinoVisualDe(pistaOval, dir, 1, InicioCerebroSalvo)
	if err != nil {
		t.Fatal(err)
	}
	if v.Geracao() != 150 || v.PenasEmVigor() != ia.PenasDaGeracao(150) || v.PenasEmVigor() == ia.PenasCheias {
		t.Errorf("geração %d, penas %+v", v.Geracao(), v.PenasEmVigor())
	}
}

func TestTreinoVisualDeCerebroSalvoAvancaDeGeracao(t *testing.T) {
	dir := t.TempDir()
	cerebroDeTeste(t, dir, 40)
	v, err := NovoTreinoVisualDe(pistaOval, dir, 1, InicioCerebroSalvo)
	if err != nil {
		t.Fatal(err)
	}
	rodarGeracao(t, v)
	if v.Geracao() != 41 {
		t.Errorf("depois de 1 geração: %d, quer 41", v.Geracao())
	}
	if len(v.Historico) != 1 || v.Historico[0].Geracao != 41 && v.Historico[0].Geracao != 40 {
		t.Errorf("histórico %+v", v.Historico)
	}
}

func TestTreinoVisualDeCerebroSalvoSemArquivoOuInvalido(t *testing.T) {
	dir := t.TempDir()
	if _, err := NovoTreinoVisualDe(pistaOval, dir, 1, InicioCerebroSalvo); err == nil {
		t.Error("sem arquivo devia dar erro")
	}
	if err := os.WriteFile(filepath.Join(dir, "oval.json"), []byte("{não é json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NovoTreinoVisualDe(pistaOval, dir, 1, InicioCerebroSalvo); err == nil {
		t.Error("cérebro inválido devia dar erro")
	}
}

func TestAppIniciarTreinoCerebroSalvo(t *testing.T) {
	dir := t.TempDir()
	cerebroDeTeste(t, dir, 320)
	m := NovoMenu([]string{pistaOval}, "")
	m.DirCerebros, m.Inicio = dir, InicioCerebroSalvo
	a := NovoAppSemJanela(m)
	if err := a.IniciarTreino(); err != nil {
		t.Fatal(err)
	}
	if a.Tela != TelaTreino || a.Treino.V.Geracao() != 320 {
		t.Errorf("tela %v, geração %d", a.Tela, a.Treino.V.Geracao())
	}
}

func TestAppIniciarTreinoCerebroSalvoSemArquivoFicaNoMenuComAviso(t *testing.T) {
	m := NovoMenu([]string{pistaOval}, "")
	m.DirCerebros, m.Inicio = t.TempDir(), InicioCerebroSalvo
	a := NovoAppSemJanela(m)
	if err := a.IniciarTreino(); err == nil {
		t.Fatal("abriu o treino sem cérebro")
	}
	if a.Tela != TelaMenu || a.Treino != nil || a.Menu.Aviso == "" {
		t.Errorf("tela %v, treino %v, aviso %q", a.Tela, a.Treino, a.Menu.Aviso)
	}
}

func TestAppIniciarTreinoModoZeroIgualAoDeHoje(t *testing.T) {
	dir := t.TempDir()
	cerebroDeTeste(t, dir, 320) // existe, mas o modo zero não olha
	m := NovoMenu([]string{pistaOval}, "")
	m.DirCerebros = dir
	a := NovoAppSemJanela(m)
	if err := a.IniciarTreino(); err != nil {
		t.Fatal(err)
	}
	ref := novoVisual(t, m.Seed)
	if a.Treino.V.Geracao() != 0 || !reflect.DeepEqual(a.Treino.V.Pop.Redes[7].Pesos, ref.Pop.Redes[7].Pesos) {
		t.Errorf("modo zero mudou: geração %d", a.Treino.V.Geracao())
	}
}
