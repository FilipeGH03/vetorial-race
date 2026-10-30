package sim

import (
	"fmt"
	"math"
	"testing"
)

// andar leva o carro por pontos q (cada um em linha reta do anterior, em passos
// de uns 4 px) e chama AtualizarProgresso a cada passo, como o tick faria.
func andar(c *Carro, p *Pista, q ...Vec2) {
	for _, alvo := range q {
		ini := c.Pos
		passos := int(math.Ceil(alvo.Sub(ini).Len() / 4))
		for i := 1; i <= passos; i++ {
			de := c.Pos
			c.Pos = ini.Add(alvo.Sub(ini).Scale(float64(i) / float64(passos)))
			c.AtualizarProgresso(de, p)
		}
	}
}

// voltaPor devolve k pontos consecutivos do circuito pts a partir do índice i0,
// no sentido de i0 crescente (ou decrescente se passo = -1), dando a volta
// quantas vezes precisar.
func voltaPor(pts []Vec2, i0, k, passo int) []Vec2 {
	n := len(pts)
	res := make([]Vec2, k)
	for j := range res {
		res[j] = pts[((i0+passo*j)%n+n)%n]
	}
	return res
}

func TestUmaVoltaNoOval(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	// Começa 3 pontos atrás da largada e anda 1 volta completa, até o ponto 1.
	c := carroEm(p.Centro[n-3], 0)
	andar(c, p, voltaPor(p.Centro, n-3, n+5, 1)...)
	if c.Progresso.Voltas != 1 {
		t.Errorf("Voltas = %d, quer 1", c.Progresso.Voltas)
	}
	// Está no ponto 1 da volta 2: Distancia ≈ L + Passo.
	want := p.Comprimento + p.Passo
	if got := c.Progresso.Distancia; math.Abs(got-want) > p.Passo {
		t.Errorf("Distancia = %.1f, quer ≈ %.1f (comprimento %.1f)", got, want, p.Comprimento)
	}
	t.Logf("comprimento %.1f, passo %.2f, Distancia %.1f, Proximo %d", p.Comprimento, p.Passo, c.Progresso.Distancia, c.Progresso.Proximo)
}

func TestDistanciaDaVoltaCompleta(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-3], 0)
	andar(c, p, voltaPor(p.Centro, n-3, n+3, 1)...) // termina em Centro[0], em cima da linha
	andar(c, p, p.Centro[1])                        // um passo à frente cruza a linha
	if c.Progresso.Voltas != 1 {
		t.Fatalf("Voltas = %d, quer 1", c.Progresso.Voltas)
	}
	if got, want := c.Progresso.Distancia, p.Comprimento; math.Abs(got-want) > 2*p.Passo {
		t.Errorf("Distancia = %.1f, quer ≈ %.1f", got, want)
	}
}

func TestLargadaNaoContaVolta(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-3], 0)
	andar(c, p, p.Centro[n-2], p.Centro[n-1])
	if c.Progresso != (Progresso{}) {
		t.Errorf("antes da linha: progresso %+v, quer zero", c.Progresso)
	}
	andar(c, p, p.Centro[1])
	if c.Progresso.Voltas != 0 || c.Progresso.Proximo != 1 {
		t.Errorf("cruzar a largada: progresso %+v, quer Voltas 0 e Proximo 1", c.Progresso)
	}
	if d := c.Progresso.Distancia; d < 0 || d > 2*p.Passo {
		t.Errorf("Distancia logo depois da largada = %.1f", d)
	}
}

func TestVariasVoltas(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-3], 0)
	andar(c, p, voltaPor(p.Centro, n-3, 3*n+n/2, 1)...) // 3 voltas e meia
	if c.Progresso.Voltas != 3 {
		t.Errorf("Voltas = %d, quer 3", c.Progresso.Voltas)
	}
	want := 3.5*p.Comprimento - 3*p.Passo
	if got := c.Progresso.Distancia; math.Abs(got-want) > 4*p.Passo {
		t.Errorf("Distancia = %.1f, quer ≈ %.1f", got, want)
	}
}

// caminhosDaPista são trajetórias que dão a volta na pista: a linha central e
// várias distâncias das paredes (f perto de 0 é rente à parede esquerda).
func caminhosDaPista(p *Pista) map[string][]Vec2 {
	n := len(p.Checkpoints)
	caminhos := map[string][]Vec2{"linha central": p.Centro}
	for _, f := range []float64{0.02, 0.2, 0.5, 0.8, 0.98} {
		q := make([]Vec2, n)
		for i, cp := range p.Checkpoints {
			q[i] = cp.A.Add(cp.B.Sub(cp.A).Scale(f))
		}
		caminhos[fmt.Sprintf("f=%v", f)] = q
	}
	return caminhos
}

// Em todas as formas de pista e a qualquer distância das paredes, uma volta
// soma 1 volta e uma Distancia perto do comprimento.
func TestVoltaEmTodasAsPistas(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		n := len(p.Checkpoints)
		for nome, q := range caminhosDaPista(p) {
			c := carroEm(q[n-3], 0)
			andar(c, p, voltaPor(q, n-3, 2*n, 1)...) // termina 3 pontos antes de fechar a 2ª volta
			if c.Progresso.Voltas != 1 {
				t.Errorf("%s, %s: Voltas = %d, quer 1", p.Nome, nome, c.Progresso.Voltas)
			}
			// Fora da linha central o caminho anda um pouco mais ou menos que ela.
			if got, want := c.Progresso.Distancia, 2*p.Comprimento-3*p.Passo; math.Abs(got-want) > 0.1*p.Comprimento {
				t.Errorf("%s, %s: Distancia = %.1f, quer ≈ %.1f", p.Nome, nome, got, want)
			}
		}
	}
}

func TestDistanciaNuncaDiminui(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-3], 0)
	ult := 0.0
	for _, q := range voltaPor(p.Centro, n-3, 2*n, 1) {
		andar(c, p, q)
		if c.Progresso.Distancia < ult {
			t.Fatalf("Distancia diminuiu de %.2f para %.2f em %v", ult, c.Progresso.Distancia, c.Pos)
		}
		ult = c.Progresso.Distancia
	}
}

// Andar de ré (no sentido contrário ao da pista) não soma nada, em nenhuma pista.
func TestReNaoConta(t *testing.T) {
	for _, p := range pistasDeForma(t) {
		n := len(p.Checkpoints)
		for nome, q := range caminhosDaPista(p) {
			c := carroEm(q[3], 0)
			andar(c, p, voltaPor(q, 3, 3*n, -1)...) // 3 voltas ao contrário
			if c.Progresso != (Progresso{}) {
				t.Errorf("%s, %s: 3 voltas de ré: progresso %+v, quer zero", p.Nome, nome, c.Progresso)
			}
		}
	}
}

func TestReDepoisDeLargarNaoSomaNemRepete(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	c := carroEm(p.Centro[n-2], 0)
	andar(c, p, p.Centro[0], p.Centro[5]) // larga e vai ao ponto 5
	apos := c.Progresso
	if apos.Proximo < 5 || apos.Voltas != 0 {
		t.Fatalf("progresso depois de andar 5 pontos: %+v", apos)
	}
	// Volta de ré até antes da largada e anda para a frente de novo até o ponto 5.
	andar(c, p, p.Centro[n-2])
	if c.Progresso != apos {
		t.Errorf("de ré: progresso %+v, quer o mesmo %+v", c.Progresso, apos)
	}
	andar(c, p, p.Centro[5])
	if c.Progresso != apos {
		t.Errorf("ir de novo até o ponto 5: progresso %+v, quer o mesmo %+v (checkpoint repetido não conta)", c.Progresso, apos)
	}
	andar(c, p, p.Centro[7])
	if c.Progresso.Proximo <= apos.Proximo || c.Progresso.Distancia <= apos.Distancia {
		t.Errorf("seguir em frente: progresso %+v, depois de %+v", c.Progresso, apos)
	}
}

func TestPularCheckpointNaoConta(t *testing.T) {
	p := carregarOval(t)
	meio := func(i int) Vec2 { return p.Centro[i].Add(p.Centro[i+1]).Scale(0.5) }
	// O carro está depois do checkpoint 2 (Proximo = 3) e "teleporta" de antes
	// do checkpoint 5 para depois dele, sem cruzar o 3 nem o 4.
	c := carroEm(meio(4), 0)
	c.Progresso = Progresso{Proximo: 3, Distancia: 2 * p.Passo}
	c.Pos = meio(5)
	c.AtualizarProgresso(meio(4), p)
	if c.Progresso.Proximo != 3 || c.Progresso.Voltas != 0 {
		t.Errorf("pulou o checkpoint 5: progresso %+v, quer Proximo 3", c.Progresso)
	}
	// Num único caminho que cruza o 3, o 4 e o 5 em ordem, os três contam.
	c = carroEm(meio(2), 0)
	c.Progresso = Progresso{Proximo: 3, Distancia: 2 * p.Passo}
	c.Pos = meio(5)
	c.AtualizarProgresso(meio(2), p)
	if c.Progresso.Proximo != 6 {
		t.Errorf("cruzou 3, 4 e 5 de uma vez: Proximo = %d, quer 6", c.Progresso.Proximo)
	}
}

func TestProgressoSemEfeito(t *testing.T) {
	p := carregarOval(t)
	n := len(p.Centro)
	// Carro morto, pista nil e pista sem checkpoints não mudam nada nem dão pânico.
	c := carroEm(p.Centro[n-1], 0)
	c.Vivo = false
	c.Pos = p.Centro[2]
	c.AtualizarProgresso(p.Centro[n-1], p)
	if c.Progresso != (Progresso{}) {
		t.Errorf("carro morto: progresso %+v", c.Progresso)
	}
	c.Vivo = true
	c.AtualizarProgresso(p.Centro[n-1], nil)
	c.AtualizarProgresso(p.Centro[n-1], &Pista{})
	if c.Progresso != (Progresso{}) {
		t.Errorf("pista nil/vazia: progresso %+v", c.Progresso)
	}
	c.AtualizarProgresso(c.Pos, p) // parado
	c.Progresso.Proximo = 99       // fora do intervalo: recomeça sem pânico
	c.AtualizarProgresso(c.Pos, p)
	c.Progresso.Proximo = -5
	c.AtualizarProgresso(c.Pos, p)
}

// Dirigindo de verdade (física e colisão do Avancar), um motorista simples que
// persegue um ponto da linha central dá voltas.
func TestVoltaDirigindoComAvancar(t *testing.T) {
	p := carregarOval(t)
	cfg := ConfigPadrao()
	n := len(p.Centro)
	c := carroEm(p.Centro[n-4], 0)
	c.DefinirAngulo(p.Centro[n-3].Sub(p.Centro[n-4]).Angle())
	// Mira o ponto da linha central 4 pontos à frente do mais perto do carro.
	alvo := func() Vec2 {
		melhor, d := 0, math.Inf(1)
		for i, q := range p.Centro {
			if dd := q.Sub(c.Pos).Len(); dd < d {
				melhor, d = i, dd
			}
		}
		return p.Centro[(melhor+4)%n]
	}
	batidas := 0
	for tick := 0; tick < 60*60 && c.Progresso.Voltas < 2; tick++ {
		erro := alvo().Sub(c.Pos).Norm().Scale(200).Sub(c.Vel)
		var ctl Controle
		for _, e := range []struct {
			eixo Eixo
			v    float64
		}{{EixoX, erro.X}, {EixoY, erro.Y}} {
			if math.Abs(e.v) > 10 {
				s := int8(1)
				if e.v < 0 {
					s = -1
				}
				ctl.Acoes = append(ctl.Acoes, Acao{Ativa: true, Eixo: e.eixo, Sinal: s})
			}
		}
		c.Avancar(ctl, p, cfg) // já atualiza o progresso (D15)
		if c.Batido {
			batidas++
		}
	}
	t.Logf("progresso %+v, %d batidas", c.Progresso, batidas)
	if c.Progresso.Voltas < 1 {
		t.Fatalf("o motorista de teste não deu uma volta em 60 s: %+v (batidas %d)", c.Progresso, batidas)
	}
	if c.Progresso.Voltas == 1 && c.Progresso.Distancia < p.Comprimento-p.Passo {
		t.Errorf("1 volta com Distancia %.1f < comprimento %.1f", c.Progresso.Distancia, p.Comprimento)
	}
}

func TestClassificar(t *testing.T) {
	mk := func(id, voltas int, dist float64) Carro {
		c := NovoCarro(id, Vec2{})
		c.Progresso = Progresso{Voltas: voltas, Distancia: dist}
		return c
	}
	carros := []Carro{
		mk(0, 1, 900),  // 3º
		mk(1, 2, 100),  // 1º: mais voltas, mesmo com Distancia menor
		mk(2, 1, 1200), // 2º: mesmas voltas que o 0, mais Distancia
		mk(3, 0, 5000), // último: menos voltas
		mk(4, 1, 900),  // empate total com o 0: fica depois dele
	}
	got := Classificar(carros)
	want := []int{1, 2, 0, 4, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Classificar = %v, quer %v", got, want)
		}
	}
	for i, c := range carros {
		if c.ID != i {
			t.Errorf("carros[%d].ID = %d: a classificação alterou o slice", i, c.ID)
		}
	}
	if r := Classificar(nil); len(r) != 0 {
		t.Errorf("Classificar(nil) = %v", r)
	}
	if !(Progresso{Voltas: 1}).MaisAdiante(Progresso{Voltas: 0, Distancia: 99}) || (Progresso{}).MaisAdiante(Progresso{}) {
		t.Error("MaisAdiante errou")
	}
}
