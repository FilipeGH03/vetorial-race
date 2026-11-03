package sim

import "math"

// Avancar faz um tick completo do carro na pista (contrato 1.1, passos 1 a 5):
// a física de Mover, depois a colisão com as paredes (D2) e por fim o progresso
// (AtualizarProgresso com a posição do começo do tick, D15). Corrida, jogo e
// treino só chamam Avancar; ninguém chama AtualizarProgresso por fora.
//
// O carro bate quando o caminho do tick (o segmento da posição anterior à nova)
// toca alguma parede; olhar o caminho, e não só o ponto final, impede que um
// carro rápido atravesse a parede entre dois ticks. O carro é o seu centro.
//
// Ao bater (D28): Batido = true, Pos volta à posição do começo do tick (a
// última válida) e o carro desliza na parede: Vel vira metade da projeção da
// velocidade na direção do trecho atingido primeiro no caminho do tick,
// Vel = 0,5·(Vel·t̂)·t̂ com t̂ = Norm(b − a). Trecho de comprimento zero: Vel = 0.
// Com cfg.BaterMata o carro também morre (Vivo = false).
// Batido vale só para este tick: é zerado no começo de cada chamada.
// Carro morto não se mexe; pista nil não tem paredes.
func (c *Carro) Avancar(ctl Controle, p *Pista, cfg Config) {
	c.Batido = false
	if !c.Vivo {
		return
	}
	anterior := c.Pos
	c.Mover(ctl, cfg)
	trecho, bateu := trechoAtingido(p, anterior, c.Pos)
	if !bateu {
		c.AtualizarProgresso(anterior, p)
		return
	}
	// Bateu: volta ao começo do tick, então o progresso não muda.
	c.Batido = true
	c.Pos = anterior
	c.Vel = deslizar(c.Vel, trecho)
	if cfg.BaterMata {
		c.Vivo = false
	}
}

// deslizar devolve a velocidade depois de bater no trecho de parede: metade da
// projeção de v na direção do trecho (D28). Trecho de comprimento zero: parado.
func deslizar(v Vec2, trecho Segmento) Vec2 {
	dir := trecho.B.Sub(trecho.A).Norm()
	if dir == (Vec2{}) {
		return Vec2{}
	}
	return dir.Scale(0.5 * v.Dot(dir))
}

// cruzaParede diz se o segmento de "de" a "ate" toca algum trecho das paredes
// da pista (pontas incluídas, com a folga folgaPonta). Trechos de comprimento
// zero contam como pontos.
func cruzaParede(p *Pista, de, ate Vec2) bool {
	_, bateu := trechoAtingido(p, de, ate)
	return bateu
}

// trechoAtingido devolve o trecho de parede que o caminho de "de" a "ate" toca
// primeiro (o de menor parâmetro ao longo do caminho; empate fica com o
// primeiro na ordem das paredes: esquerda, depois direita).
func trechoAtingido(p *Pista, de, ate Vec2) (Segmento, bool) {
	if p == nil || de == ate {
		return Segmento{}, false
	}
	lo := Vec2{X: min(de.X, ate.X), Y: min(de.Y, ate.Y)}
	hi := Vec2{X: max(de.X, ate.X), Y: max(de.Y, ate.Y)}
	var melhor Segmento
	melhorT := math.Inf(1)
	achou := false
	// Laço direto nas paredes (e não trechosParedes): é o caminho quente de todo tick.
	for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
		for k, a := range w {
			b := w[0]
			if k+1 < len(w) {
				b = w[k+1]
			}
			if caixasSeparadas(a, b, lo, hi) || !cruzaTrecho(de, ate, a, b) {
				continue
			}
			if t := paramToque(de, ate, a, b); t < melhorT {
				melhor, melhorT, achou = Segmento{A: a, B: b}, t, true
			}
		}
	}
	return melhor, achou
}

// paramToque é o parâmetro (0 a 1) ao longo do caminho de "de" a "ate" em que
// ele toca o trecho ab, que se sabe tocado (cruzaTrecho). Caminho paralelo ao
// trecho ou trecho de comprimento zero: o ponto do trecho mais perto do começo.
func paramToque(de, ate, a, b Vec2) float64 {
	r := ate.Sub(de)
	if t, _, ok := interseccaoRetas(de, r, a, b.Sub(a)); ok {
		return max(0, min(1, t))
	}
	r2 := r.Dot(r)
	ta, tb := a.Sub(de).Dot(r)/r2, b.Sub(de).Dot(r)/r2
	return max(0, min(1, min(ta, tb)))
}
