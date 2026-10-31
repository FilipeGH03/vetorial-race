package sim

import "math"

// NumSensores é a quantidade de raios dos sensores (contrato 1.3).
const NumSensores = 7

// margemCaixaRaio (px) folga a caixa de cada raio contra o arredondamento e a
// folgaPonta dos trechos.
const margemCaixaRaio = 1e-6

// angulosSensores são os ângulos dos raios, em graus, a partir de +x do mundo
// (D20): o sensor k aponta para k·360/7 graus, fixo, sem depender do ângulo do
// carro.
var angulosSensores = func() (a [NumSensores]float64) {
	for k := range a {
		a[k] = float64(k) * 360 / NumSensores
	}
	return a
}()

// dirsSensores são os vetores unitários dos raios no referencial do mundo.
var dirsSensores = func() (d [NumSensores]Vec2) {
	for k := range d {
		s, c := math.Sincos(angulosSensores[k] * math.Pi / 180)
		d[k] = Vec2{X: c, Y: s}
	}
	return d
}()

// Ler monta a Leitura do carro na pista (contrato 1.3, D20): os 7 sensores em
// direções fixas do mundo (distância até a parede / AlcanceSensor, em [0,1]; 1
// se o raio não bate), a velocidade, o módulo, o ângulo guardado no carro (D11)
// e a direção da pista à frente (DirPista).
func Ler(c *Carro, p *Pista, cfg Config) Leitura {
	l := Leitura{Vel: c.Vel, Modulo: c.Vel.Len(), Angulo: c.Angulo(), DirPista: dirPista(c, p)}
	dist := distanciasSensores(c, p, cfg.AlcanceSensor)
	for i, d := range dist {
		if cfg.AlcanceSensor > 0 {
			l.Sensores[i] = d / cfg.AlcanceSensor
		} else {
			l.Sensores[i] = 1 // sem alcance o sensor não enxerga parede nenhuma
		}
	}
	return l
}

// RaiosSensores devolve os raios dos sensores, do centro do carro até onde
// cada um bate na parede (ou até o alcance, se não bate), para a tela de
// treino desenhar. Mesma ordem de Leitura.Sensores.
func RaiosSensores(c *Carro, p *Pista, cfg Config) [NumSensores]Segmento {
	var raios [NumSensores]Segmento
	dist := distanciasSensores(c, p, cfg.AlcanceSensor)
	for i, d := range dist {
		raios[i] = Segmento{A: c.Pos, B: c.Pos.Add(direcaoSensor(i).Scale(d))}
	}
	return raios
}

// direcaoSensor é o vetor unitário do raio i, no referencial do mundo (D20).
func direcaoSensor(i int) Vec2 { return dirsSensores[i] }

// dirPista é a direção da pista à frente do carro (D23), com m pontos em Centro
// e j o trecho Centro[j] -> Centro[j+1] da linha central mais perto da posição
// do carro (não depende de Progresso): [0] = Norm(Centro[(j+1)%m] − Centro[j]) e
// [1] = Norm(Centro[(j+6)%m] − Centro[(j+5)%m]). Sem pista, vetores nulos.
func dirPista(c *Carro, p *Pista) [2]Vec2 {
	if p == nil || len(p.Centro) == 0 {
		return [2]Vec2{}
	}
	m := len(p.Centro)
	j, _ := trechoMaisPertoDe(p, c.Pos)
	at := func(k int) Vec2 { return p.Centro[(j+k)%m] }
	return [2]Vec2{at(1).Sub(at(0)).Norm(), at(6).Sub(at(5)).Norm()}
}

// trechoMaisPertoDe devolve o índice j do trecho Centro[j] -> Centro[j+1] da
// linha central mais perto de q e a projeção de q nele, em px a partir de
// Centro[j], limitada a [0, comprimento do trecho]. Empate fica com o menor j.
// Pista sem Centro devolve (0, 0).
func trechoMaisPertoDe(p *Pista, q Vec2) (int, float64) {
	n := len(p.Centro)
	melhor, proj, menor := 0, 0.0, math.Inf(1)
	for j, a := range p.Centro {
		ab := p.Centro[(j+1)%n].Sub(a)
		l2 := ab.Dot(ab)
		t := 0.0
		if l2 > 0 {
			t = max(0, min(1, q.Sub(a).Dot(ab)/l2))
		}
		if r := q.Sub(a.Add(ab.Scale(t))); r.Dot(r) < menor {
			melhor, proj, menor = j, t*math.Sqrt(l2), r.Dot(r)
		}
	}
	return melhor, proj
}

// AvancoNaLinha diz quanto o caminho de "de" até "ate" avançou no sentido da
// pista, em px com sinal (D23): a diferença das projeções dos dois pontos na
// linha central, medidas ao longo dela a partir de Centro[0]. Conta também antes
// de cruzar a largada e não depende de Progresso. Andar no sentido da pista dá
// valor positivo, de ré negativo, ficar parado zero; a soma dos avanços de um
// caminho que dá uma volta é ≈ p.Comprimento. A diferença é levada para
// (−Comprimento/2, Comprimento/2], de modo que passar do fim da linha central
// para o começo não dá um salto. Pista nil ou sem Centro devolve 0.
func AvancoNaLinha(p *Pista, de, ate Vec2) float64 {
	if p == nil || len(p.Centro) == 0 || p.Comprimento <= 0 {
		return 0
	}
	d := arcoDoPonto(p, ate) - arcoDoPonto(p, de)
	d = math.Mod(d, p.Comprimento)
	if d > p.Comprimento/2 {
		d -= p.Comprimento
	} else if d <= -p.Comprimento/2 {
		d += p.Comprimento
	}
	return d
}

// arcoDoPonto é a distância, ao longo da linha central a partir de
// Centro[0], da projeção de q nela (de 0 a p.Comprimento).
func arcoDoPonto(p *Pista, q Vec2) float64 {
	j, s := trechoMaisPertoDe(p, q)
	return float64(j)*p.Passo + s
}

// distanciasSensores devolve, para cada raio, a distância do centro do carro
// até a parede mais próxima nessa direção, no máximo alcance. Trechos de parede
// de comprimento zero são aceitos (valem como pontos).
func distanciasSensores(c *Carro, p *Pista, alcance float64) [NumSensores]float64 {
	alcance = max(alcance, 0)
	var dist [NumSensores]float64
	var dirs [NumSensores]Vec2
	for i := range dist {
		dist[i] = alcance
		dirs[i] = direcaoSensor(i)
	}
	if p == nil || alcance == 0 {
		return dist
	}
	o := c.Pos
	lo := Vec2{X: o.X - alcance, Y: o.Y - alcance}
	hi := Vec2{X: o.X + alcance, Y: o.Y + alcance}
	// Caixa de cada raio (com margem para o arredondamento): só os trechos que
	// a tocam podem ser atingidos, e são poucos. Muda só a velocidade (A5).
	var rlo, rhi [NumSensores]Vec2
	for i, d := range dirs {
		f := o.Add(d.Scale(alcance))
		rlo[i] = Vec2{X: min(o.X, f.X) - margemCaixaRaio, Y: min(o.Y, f.Y) - margemCaixaRaio}
		rhi[i] = Vec2{X: max(o.X, f.X) + margemCaixaRaio, Y: max(o.Y, f.Y) + margemCaixaRaio}
	}
	// Laço direto nas paredes (e não trechosParedes): é o caminho quente do treino.
	for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
		for k, a := range w {
			b := w[0]
			if k+1 < len(w) {
				b = w[k+1]
			}
			sLo := Vec2{X: min(a.X, b.X), Y: min(a.Y, b.Y)}
			sHi := Vec2{X: max(a.X, b.X), Y: max(a.Y, b.Y)}
			if sHi.X < lo.X || sLo.X > hi.X || sHi.Y < lo.Y || sLo.Y > hi.Y {
				continue // longe demais para qualquer raio
			}
			for i, d := range dirs {
				if sHi.X < rlo[i].X || sLo.X > rhi[i].X || sHi.Y < rlo[i].Y || sLo.Y > rhi[i].Y {
					continue
				}
				if dd, ok := distanciaRaio(o, d, a, b); ok && dd < dist[i] {
					dist[i] = dd
				}
			}
		}
	}
	return dist
}
