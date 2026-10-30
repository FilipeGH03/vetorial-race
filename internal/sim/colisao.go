package sim

// Avancar faz um tick completo do carro na pista (contrato 1.1, passos 1 a 5):
// a física de Mover e depois a colisão com as paredes (D2).
//
// O carro bate quando o caminho do tick (o segmento da posição anterior à nova)
// toca alguma parede; olhar o caminho, e não só o ponto final, impede que um
// carro rápido atravesse a parede entre dois ticks. O carro é o seu centro.
//
// Ao bater: Batido = true, Vel = 0 e Pos volta à posição do começo do tick (a
// última válida). Com cfg.BaterMata o carro também morre (Vivo = false).
// Batido vale só para este tick: é zerado no começo de cada chamada.
// Carro morto não se mexe; pista nil não tem paredes.
func (c *Carro) Avancar(ctl Controle, p *Pista, cfg Config) {
	c.Batido = false
	if !c.Vivo {
		return
	}
	anterior := c.Pos
	c.Mover(ctl, cfg)
	if !cruzaParede(p, anterior, c.Pos) {
		return
	}
	c.Batido = true
	c.Pos = anterior
	c.Vel = Vec2{}
	if cfg.BaterMata {
		c.Vivo = false
	}
}

// cruzaParede diz se o segmento de "de" a "ate" toca algum trecho das paredes
// da pista (pontas incluídas, com a folga folgaPonta). Trechos de comprimento
// zero contam como pontos.
func cruzaParede(p *Pista, de, ate Vec2) bool {
	if p == nil || de == ate {
		return false
	}
	lo := Vec2{X: min(de.X, ate.X), Y: min(de.Y, ate.Y)}
	hi := Vec2{X: max(de.X, ate.X), Y: max(de.Y, ate.Y)}
	for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
		n := len(w)
		for i := 0; i < n; i++ {
			a, b := w[i], w[(i+1)%n]
			if caixasSeparadas(a, b, lo, hi) {
				continue
			}
			if cruzaTrecho(de, ate, a, b) {
				return true
			}
		}
	}
	return false
}
