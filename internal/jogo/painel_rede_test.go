package jogo

import (
	"math"
	"testing"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

// leituraDe é a Leitura que o controlador vê agora para o carro do indivíduo i.
func leituraDe(v *TreinoVisual, i int) sim.Leitura {
	return sim.Ler(&v.Individuos[i].Carro, v.Treino.Pista, v.Treino.Cfg)
}

func TestModeloRedeDoLider(t *testing.T) {
	v := novoVisual(t, 1)
	for range 90 { // alguns tick para os carros se afastarem da largada
		v.Passo()
	}
	v.AtualizarModeloRede()
	m := v.ModeloRede()
	if m == nil {
		t.Fatal("sem modelo depois de AtualizarModeloRede")
	}
	lider := v.Lider()
	if m.Individuo != lider {
		t.Errorf("modelo do indivíduo %d, o líder é o %d", m.Individuo, lider)
	}
	if len(m.Nos) != 3 || len(m.Nos[0]) != 14 || len(m.Nos[1]) != 12 || len(m.Nos[2]) != 2 {
		t.Fatalf("camadas %d/%d/%d/%d, quero 14/12/2", len(m.Nos), len(m.Nos[0]), len(m.Nos[1]), len(m.Nos[2]))
	}
	rede := v.Individuos[lider].ctl.Rede
	ativ, err := rede.Ativacoes(ia.Entrada(leituraDe(v, lider)))
	if err != nil {
		t.Fatal(err)
	}
	for k := range ativ {
		for i, a := range ativ[k] {
			if m.Nos[k][i].Valor != a {
				t.Errorf("nó %d da camada %d: valor %v, Ativacoes dá %v", i, k, m.Nos[k][i].Valor, a)
			}
		}
	}
	saida, _ := rede.Forward(ia.Entrada(leituraDe(v, lider)))
	for i, s := range saida {
		if m.Nos[2][i].Valor != s {
			t.Errorf("saída %d: %v, Forward dá %v", i, m.Nos[2][i].Valor, s)
		}
	}
}

func TestModeloRedeRotulosPosicoesELigacoes(t *testing.T) {
	v := novoVisual(t, 1)
	v.AtualizarModeloRede()
	m := v.ModeloRede()
	if m == nil {
		t.Fatal("sem modelo")
	}
	for i, n := range m.Nos[0] {
		if n.Rotulo == "" {
			t.Errorf("entrada %d sem rótulo", i)
		}
	}
	if m.Nos[2][0].Rotulo == "" || m.Nos[2][1].Rotulo == "" {
		t.Error("saída sem rótulo")
	}
	a := AreaPainelRede
	for k, nos := range m.Nos {
		for i, n := range nos {
			if int(n.X) < a.Min.X || int(n.X) > a.Max.X || int(n.Y) < a.Min.Y || int(n.Y) > a.Max.Y {
				t.Errorf("nó %d da camada %d fora do painel: (%v, %v)", i, k, n.X, n.Y)
			}
			if k > 0 && n.X <= m.Nos[k-1][0].X {
				t.Errorf("camada %d não fica à direita da %d", k, k-1)
			}
			if i > 0 && n.Y <= nos[i-1].Y {
				t.Errorf("nó %d da camada %d não fica abaixo do anterior", i, k)
			}
		}
	}
	if want := 14*12 + 12*2; len(m.Ligacoes) != want {
		t.Fatalf("%d ligações, quero %d", len(m.Ligacoes), want)
	}
	rede := v.Individuos[m.Individuo].ctl.Rede
	maior := 0.0
	for _, lg := range m.Ligacoes {
		if w := rede.Pesos[lg.Camada].At(lg.Para, lg.De); lg.Peso != w {
			t.Fatalf("ligação %+v: peso da rede %v", lg, w)
		}
		maior = math.Max(maior, math.Abs(lg.Peso))
	}
	if m.PesoMax != maior {
		t.Errorf("PesoMax %v, o maior |peso| é %v", m.PesoMax, maior)
	}
}

func TestModeloRedeSegueOTickEATrocaComOLider(t *testing.T) {
	v := novoVisual(t, 1)
	v.AtualizarModeloRede()
	antes := v.ModeloRede()
	for range 30 {
		v.Passo()
	}
	v.AtualizarModeloRede()
	depois := v.ModeloRede()
	if antes == depois {
		t.Fatal("o modelo não foi remontado")
	}
	mudou := false
	for i := range depois.Nos[0] {
		if depois.Nos[0][i].Valor != antes.Nos[0][i].Valor {
			mudou = true
		}
	}
	if !mudou {
		t.Error("as entradas não acompanham o carro: iguais depois de 30 ticks")
	}
	// Muda o líder à mão: o painel passa para a rede do novo líder.
	l := v.Lider()
	outro := (l + 1) % len(v.Individuos)
	v.Individuos[outro].Carro.Progresso.Distancia = v.Individuos[l].Carro.Progresso.Distancia + 100
	if v.Lider() != outro {
		t.Fatalf("o líder devia ser %d, é %d", outro, v.Lider())
	}
	v.AtualizarModeloRede()
	if got := v.ModeloRede().Individuo; got != outro {
		t.Errorf("painel no indivíduo %d depois da troca de líder, quero %d", got, outro)
	}
}

func TestModeloRedeSemLiderMantemAUltima(t *testing.T) {
	v := novoVisual(t, 1)
	for range 30 {
		v.Passo()
	}
	v.AtualizarModeloRede()
	ultima := v.ModeloRede()
	if ultima == nil {
		t.Fatal("sem modelo")
	}
	for i := range v.Individuos { // todos parados
		v.Individuos[i].Fim = true
	}
	if v.Rodando() != 0 {
		t.Fatal("ainda há quem dirige")
	}
	v.AtualizarModeloRede()
	if v.ModeloRede() != ultima {
		t.Error("com todos parados o painel devia manter a última rede")
	}
}

func TestPainelRedeComecaLigadoENAlterna(t *testing.T) {
	v := novoVisual(t, 1)
	if !v.PainelRedeVisivel {
		t.Fatal("o painel da rede devia começar ligado (D32)")
	}
	v.AlternarPainelRede()
	if v.PainelRedeVisivel {
		t.Error("N não desligou o painel")
	}
	v.AlternarPainelRede()
	if !v.PainelRedeVisivel {
		t.Error("N não religou o painel")
	}
}

func TestMontarModeloRedeErros(t *testing.T) {
	if _, err := MontarModeloRede(nil, sim.Leitura{}, AreaPainelRede); err == nil {
		t.Error("rede nil devia dar erro")
	}
	r, err := ia.NovaRede([]int{3, 2}, nil)
	if err == nil {
		// rng nil pode ser recusado ou aceito; só interessa que uma rede de 3
		// entradas não casa com a Entrada de 14 valores.
		if _, err := MontarModeloRede(r, sim.Leitura{}, AreaPainelRede); err == nil {
			t.Error("rede com 3 entradas devia dar erro")
		}
	}
}

func TestCoresDoPainel(t *testing.T) {
	_, fina := corLigacao(0.1, 2)
	cp, grossa := corLigacao(2, 2)
	cn, _ := corLigacao(-2, 2)
	if !(grossa > fina) {
		t.Errorf("espessura %v para |peso| máximo, %v para fraco: devia crescer", grossa, fina)
	}
	if !(cp.G > cp.R) || !(cn.R > cn.G) {
		t.Errorf("peso positivo %v devia ser verde e negativo %v vermelho", cp, cn)
	}
	if _, e := corLigacao(1, 0); e <= 0 {
		t.Error("espessura com PesoMax 0 devia ser positiva")
	}
	if p, n := corAtivacao(1), corAtivacao(-1); !(p.R > p.B) || !(n.B > n.R) {
		t.Errorf("ativação positiva %v devia ser quente e negativa %v fria", p, n)
	}
	if corAtivacao(math.NaN()) != corAtivacao(0) {
		t.Error("NaN devia colorir como 0")
	}
}
