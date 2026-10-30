package sim

import "math"

// Regras da largada (D16).
const (
	// RecuoLargada é quanto (px) a 1ª fila fica antes da linha de largada
	// (checkpoint 0): a 1ª travessia da largada acontece logo no começo.
	RecuoLargada = 20.0
	// FolgaLargada é a menor distância (px) de cada carro da largada às paredes.
	FolgaLargada = 8.0

	// espacoLateralLargada é o espaço mínimo entre carros vizinhos de uma fila,
	// em larguras de pista: a largura do sprite na tela (D14: comprimento 0,4 da
	// largura, proporção de ~2 para 1). Se não cabem todos lado a lado nesse
	// espaço, a largada é em filas.
	espacoLateralLargada = 0.2
	// espacoFilaLargada é a distância entre filas, em larguras de pista: o
	// comprimento do sprite (0,4) com folga.
	espacoFilaLargada = 0.5
	// passoFaixaLargada é a resolução (px) da busca da faixa livre de uma fila.
	passoFaixaLargada = 0.5
	// folgaLinhaLargada é a menor distância (px) de um carro à linha de largada.
	folgaLinhaLargada = 1.0
)

// Largada devolve n carros (NovoCarro, IDs 0 a n-1) parados antes da linha de
// largada (checkpoint 0), lado a lado entre as paredes e apontados na direção
// da pista (D16).
//
// A 1ª fila fica paralela à linha de largada, RecuoLargada px atrás dela, com os
// carros apontados para a frente da linha. Cada carro fica a pelo menos
// FolgaLargada px das paredes, atrás da linha e no trecho da pista que chega
// nela. Se os n carros não cabem lado a lado (vizinhos a menos de 0,2 largura
// um do outro), eles se dividem em filas com o mesmo número de carros (±1).
// Cada fila de trás atravessa a pista num ponto da linha central, apontada na
// direção dela ali, no primeiro ponto em que fica inteira a pelo menos 0,5
// largura da fila da frente (o comprimento do carro com folga). Na fila, os
// carros se espalham por igual, da parede esquerda para a direita, na ordem dos
// IDs.
//
// Como todos nascem atrás da linha com Proximo = 0, a 1ª travessia da largada
// (andando para a frente) abre a volta 1 e a Distancia passa a contar. O
// resultado só depende da pista e de n. Pista nil ou vazia, ou n ≤ 0, devolve nil.
func Largada(p *Pista, n int) []Carro {
	if p == nil || n <= 0 || len(p.Centro) < 2 || len(p.Checkpoints) != len(p.Centro) {
		return nil
	}
	f0 := primeiraFila(p)
	porFila := int(f0.largura() / (espacoLateralLargada * p.Largura))
	porFila = max(1, min(n, porFila))
	filas := (n + porFila - 1) / porFila
	carros := make([]Carro, 0, n)
	f, recuo := f0, RecuoLargada
	for i := 0; i < filas; i++ {
		if i > 0 {
			f, recuo = filaAtras(p, recuo, f)
		}
		k := n / filas
		if i < n%filas {
			k++
		}
		for j := 0; j < k; j++ {
			o := f.esq - f.largura()*(float64(j)+0.5)/float64(k)
			c := NovoCarro(len(carros), f.base.Add(f.lateral.Scale(o)))
			c.DefinirAngulo(f.frente.Angle())
			carros = append(carros, c)
		}
	}
	return carros
}

// faixa é o trecho livre de uma fila da largada: os pontos base + lateral·o com
// o de dir a esq (dir ≤ 0 ≤ esq).
type faixa struct {
	base, frente, lateral Vec2 // ponto do meio da fila, direção dos carros e a esquerda dela
	esq, dir              float64
}

func (f faixa) largura() float64 { return f.esq - f.dir }

// pontas devolve as pontas da faixa (direita e esquerda).
func (f faixa) pontas() (Vec2, Vec2) {
	return f.base.Add(f.lateral.Scale(f.dir)), f.base.Add(f.lateral.Scale(f.esq))
}

// primeiraFila é a fila paralela à linha de largada, RecuoLargada px atrás do
// meio dela. Se esse ponto não serve (pista que chega na largada por uma curva
// fechada), usa a fila que atravessa a pista RecuoLargada px antes, ao longo da
// linha central.
func primeiraFila(p *Pista) faixa {
	cp0 := p.Checkpoints[0]
	meio := cp0.A.Add(cp0.B).Scale(0.5)
	lat := cp0.A.Sub(cp0.B).Norm() // da parede direita para a esquerda
	frente := Vec2{lat.Y, -lat.X}  // a esquerda de frente é lat
	base := meio.Sub(frente.Scale(RecuoLargada))
	if !cruzaParede(p, meio, base) {
		if f, ok := faixaEm(p, base, frente); ok {
			return f
		}
	}
	return filaUnica(p, RecuoLargada)
}

// filaAtras é a fila de trás da fila anterior: a que atravessa a pista no
// ponto da linha central mais perto da largada, a partir de desde px antes de
// Centro[0], que fica inteira a pelo menos 0,5 largura da anterior (recua de 1
// em 1 px até achar). Devolve a fila e o recuo dela ao longo da linha central.
func filaAtras(p *Pista, desde float64, anterior faixa) (faixa, float64) {
	gap := espacoFilaLargada * p.Largura
	a1, a2 := anterior.pontas()
	for recuo := desde + 1; recuo < desde+p.Comprimento/2; recuo++ {
		base, frente := pontoAtras(p, recuo)
		if distanciaPontoSegmento(base, a1, a2) < gap {
			continue // a fila passa por base: perto demais, nem precisa montar
		}
		f, ok := faixaEm(p, base, frente)
		if !ok {
			continue
		}
		if b1, b2 := f.pontas(); distanciaEntreSegmentos(a1, a2, b1, b2) >= gap {
			return f, recuo
		}
	}
	return filaUnica(p, desde+gap), desde + gap
}

// filaUnica é a fila que atravessa a pista no ponto da linha central recuo px
// antes de Centro[0], apontada na direção da pista ali. Se nesse ponto não cabe
// carro nenhum (quina viva logo antes da largada), recua de 1 em 1 px até caber.
func filaUnica(p *Pista, recuo float64) faixa {
	for extra := 0.0; extra < p.Comprimento/2; extra++ {
		base, frente := pontoAtras(p, recuo+extra)
		if f, ok := faixaEm(p, base, frente); ok {
			return f
		}
	}
	// Não acontece numa pista válida; sem faixa livre, todos no ponto da linha central.
	base, frente := pontoAtras(p, recuo)
	return faixa{base: base, frente: frente, lateral: Vec2{-frente.Y, frente.X}}
}

// faixaEm acha a faixa livre da fila que passa por base, atravessando a pista
// na perpendicular de frente. Um ponto da faixa está livre se fica a
// FolgaLargada das paredes, não encosta na linha de largada, está no trecho da
// pista de base (o trecho da linha central mais perto dele é vizinho do de
// base) e, perto da largada, atrás dela. ok = false se o próprio base não está
// livre.
func faixaEm(p *Pista, base, frente Vec2) (faixa, bool) {
	f := faixa{base: base, frente: frente, lateral: Vec2{-frente.Y, frente.X}}
	cp0 := p.Checkpoints[0]
	alcance := p.Largura // até onde a faixa vai de cada lado de base
	paredes := trechosPerto(p, base, alcance+FolgaLargada)
	centro := trechosCentroPerto(p, base, 2*alcance+RecuoLargada) // cobre o trecho mais perto de qualquer ponto da faixa
	n := len(p.Centro)
	const vizinho = 2 // no máximo 2 trechos da linha central de distância do de base
	segBase := maisPerto(p, centro, base)
	livre := func(o float64) bool {
		q := base.Add(f.lateral.Scale(o))
		if distanciaPontoSegmento(q, cp0.A, cp0.B) < folgaLinhaLargada {
			return false
		}
		if distanciaPontoSegmento(q, cp0.A, cp0.B) < p.Largura && orientacao(cp0.A, cp0.B, q) >= 0 {
			return false // à frente da largada
		}
		for _, s := range paredes {
			if distanciaPontoSegmento(q, s.A, s.B) < FolgaLargada {
				return false
			}
		}
		d := maisPerto(p, centro, q) - segBase
		d = ((d % n) + n) % n
		return min(d, n-d) <= vizinho
	}
	if len(centro) == 0 || !livre(0) {
		return f, false
	}
	for f.esq+passoFaixaLargada <= alcance && livre(f.esq+passoFaixaLargada) {
		f.esq += passoFaixaLargada
	}
	for f.dir-passoFaixaLargada >= -alcance && livre(f.dir-passoFaixaLargada) {
		f.dir -= passoFaixaLargada
	}
	return f, true
}

// pontoAtras devolve o ponto da linha central d px antes de Centro[0] (andando
// contra o sentido da pista) e a direção da pista (unitária) no trecho em que
// ele cai.
func pontoAtras(p *Pista, d float64) (ponto, frente Vec2) {
	n := len(p.Centro)
	i := 0
	for k := 0; k < 2*n; k++ {
		a, b := p.Centro[i], p.Centro[(i-1+n)%n] // b vem antes de a na pista
		l := a.Sub(b).Len()
		if l > 0 && d <= l {
			return a.Add(b.Sub(a).Scale(d / l)), a.Sub(b).Scale(1 / l)
		}
		d -= l
		i = (i - 1 + n) % n
	}
	// d maior que duas voltas: fica no começo.
	return p.Centro[0], p.Centro[1].Sub(p.Centro[0]).Norm()
}

// trechosPerto devolve os trechos das paredes a até r px de q.
func trechosPerto(p *Pista, q Vec2, r float64) []Segmento {
	var res []Segmento
	for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
		for i := range w {
			a, b := w[i], w[(i+1)%len(w)]
			if distanciaPontoSegmento(q, a, b) <= r {
				res = append(res, Segmento{a, b})
			}
		}
	}
	return res
}

// trechosCentroPerto devolve os índices i dos trechos Centro[i] -> Centro[i+1]
// da linha central a até r px de q.
func trechosCentroPerto(p *Pista, q Vec2, r float64) []int {
	var res []int
	n := len(p.Centro)
	for i := range p.Centro {
		if distanciaPontoSegmento(q, p.Centro[i], p.Centro[(i+1)%n]) <= r {
			res = append(res, i)
		}
	}
	return res
}

// maisPerto devolve, entre os trechos da linha central em idx, o mais perto de q.
func maisPerto(p *Pista, idx []int, q Vec2) int {
	n := len(p.Centro)
	melhor, dist := -1, math.Inf(1)
	for _, i := range idx {
		if d := distanciaPontoSegmento(q, p.Centro[i], p.Centro[(i+1)%n]); d < dist {
			melhor, dist = i, d
		}
	}
	return melhor
}
