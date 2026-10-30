package sim

import "math"

// NumSensores é a quantidade de raios dos sensores (contrato 1.3).
const NumSensores = 7

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
	for _, w := range [2][]Vec2{p.ParedeEsq, p.ParedeDir} {
		n := len(w)
		for j := 0; j < n; j++ {
			a, b := w[j], w[(j+1)%n]
			if caixasSeparadas(a, b, lo, hi) {
				continue // longe demais para qualquer raio
			}
			for i, d := range dirs {
				if t, ok := distanciaRaio(o, d, a, b); ok && t < dist[i] {
					dist[i] = t
				}
			}
		}
	}
	return dist
}
