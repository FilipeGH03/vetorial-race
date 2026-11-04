package jogo

import (
	"math/rand/v2"
	"testing"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

// Rede com duas camadas escondidas (contrato 1.4 aceita): o painel não fixa 14-12-2.
func TestModeloRedeComDuasEscondidas(t *testing.T) {
	r, err := ia.NovaRede([]int{14, 8, 6, 2}, rand.New(rand.NewPCG(3, 4)))
	if err != nil {
		t.Fatal(err)
	}
	m, err := MontarModeloRede(r, sim.Leitura{}, AreaPainelRede)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Nos) != 4 || len(m.Nos[1]) != 8 || len(m.Nos[2]) != 6 || len(m.Nos[3]) != 2 {
		t.Fatalf("camadas erradas: %d", len(m.Nos))
	}
	if want := 14*8 + 8*6 + 6*2; len(m.Ligacoes) != want {
		t.Errorf("%d ligações, quero %d", len(m.Ligacoes), want)
	}
	saida, _ := r.Forward(ia.Entrada(sim.Leitura{}))
	for i, s := range saida {
		if m.Nos[3][i].Valor != s {
			t.Errorf("saída %d: %v, Forward dá %v", i, m.Nos[3][i].Valor, s)
		}
	}
	for k, nos := range m.Nos[1:3] {
		for _, n := range nos {
			if n.Rotulo != "" {
				t.Errorf("oculto da camada %d com rótulo %q", k+1, n.Rotulo)
			}
		}
	}
}

// O painel não pode cobrir o botão Salvar, o gráfico nem sair da tela.
func TestAreaPainelRedeNaoSobrepoe(t *testing.T) {
	a := AreaPainelRede
	if a.Min.X < 0 || a.Min.Y < 0 || a.Max.X > LarguraTela || a.Max.Y > AlturaTela {
		t.Errorf("painel %v fora da tela", a)
	}
	if a.Overlaps(BotaoSalvar) {
		t.Errorf("painel %v cobre o botão Salvar %v", a, BotaoSalvar)
	}
	if a.Overlaps(areaGrafico) {
		t.Errorf("painel %v cobre o gráfico %v", a, areaGrafico)
	}
}

// Modelo guarda cópias: remontar não altera o modelo anterior.
func TestModeloRedeAnteriorNaoMuda(t *testing.T) {
	v := novoVisual(t, 1)
	v.AtualizarModeloRede()
	antes := v.ModeloRede()
	val := antes.Nos[0][7].Valor
	for range 40 {
		v.Passo()
	}
	v.AtualizarModeloRede()
	if antes.Nos[0][7].Valor != val {
		t.Error("o modelo antigo mudou ao remontar")
	}
}
