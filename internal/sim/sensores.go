package sim

import "math"

// NumSensores é a quantidade de raios dos sensores (contrato 1.3).
const NumSensores = 7

// margemCaixaRaio (px) folga a caixa de cada raio contra o arredondamento e a
// folgaPonta dos trechos.
const margemCaixaRaio = 1e-6

// angulosSensores são os ângulos dos raios, em graus, em relação ao ângulo do
// carro (c.Angulo()); positivo é para a esquerda (anti-horário).
var angulosSensores = [NumSensores]float64{-90, -45, -20, 0, 20, 45, 90}

// Ler monta a Leitura do carro na pista (contrato 1.3): os 7 sensores
// (distância até a parede / AlcanceSensor, em [0,1]; 1 se o raio não bate),
// a velocidade, o módulo e o ângulo guardado no carro (D11).
func Ler(c *Carro, p *Pista, cfg Config) Leitura {
	l := Leitura{Vel: c.Vel, Modulo: c.Vel.Len(), Angulo: c.Angulo()}
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
		raios[i] = Segmento{A: c.Pos, B: c.Pos.Add(direcaoSensor(c, i).Scale(d))}
	}
	return raios
}

// direcaoSensor é o vetor unitário do raio i do carro c.
func direcaoSensor(c *Carro, i int) Vec2 {
	return Vec2{X: 1}.Rotate(c.Angulo() + angulosSensores[i]*math.Pi/180)
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
		dirs[i] = direcaoSensor(c, i)
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
