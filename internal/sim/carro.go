package sim

// Eixo escolhe qual componente do vetor velocidade uma Acao mexe.
type Eixo int8

const (
	EixoX Eixo = 0
	EixoY Eixo = 1
)

// Acao é um pedido de mudança num componente da velocidade.
type Acao struct {
	Ativa bool // false = não mexe em nada neste tick
	Eixo  Eixo
	Sinal int8 // +1 aumenta, -1 diminui
}

// Controle é um tick de controle. O humano pode segurar duas teclas
// (ex.: W e D), por isso é uma lista.
type Controle struct{ Acoes []Acao }

// Controlador decide o Controle de um carro a cada tick.
type Controlador interface {
	// Chamado uma vez por tick, antes do passo da física.
	Decidir(l Leitura) Controle
}

// Progresso é o avanço do carro na pista (contrato 1.2).
type Progresso struct {
	Proximo   int     // índice do próximo checkpoint a cruzar
	Voltas    int     // voltas completas
	Distancia float64 // ao longo da linha central desde a largada, somando voltas (px)
}

// Leitura é o que o carro "enxerga" (contrato 1.3). Aqui só o tipo; quem
// preenche é sim.Ler.
type Leitura struct {
	Sensores [NumSensores]float64 // distância até a parede / AlcanceSensor, em [0,1]
	Vel      Vec2                 // componentes x e y
	Modulo   float64              // |Vel|
	Angulo   float64              // c.Angulo() (D11)
	DirPista [2]Vec2              // D20: direção da pista à frente, vetores unitários (mundo)
}

// limiarAngulo é o módulo de Vel abaixo do qual o carro mantém o ângulo anterior.
const limiarAngulo = 1.0

// Carro é o estado de um carro na simulação.
type Carro struct {
	ID        int
	Pos       Vec2
	Vel       Vec2 // vetor (x, y); a resultante é Vel
	Vivo      bool
	Batido    bool // bateu na parede neste tick
	Progresso Progresso

	angulo float64 // último ângulo válido (rad); em 0 a frente do sprite aponta para +x
}

// NovoCarro cria um carro vivo e parado em pos, com a frente para +x.
func NovoCarro(id int, pos Vec2) Carro {
	return Carro{ID: id, Pos: pos, Vivo: true}
}

// Angulo é o ângulo em que o carro é desenhado: atan2(Vel.Y, Vel.X), mantendo o
// último enquanto |Vel| < 1.
func (c *Carro) Angulo() float64 { return c.angulo }

// DefinirAngulo fixa o ângulo (ex.: ao colocar o carro na largada, na direção da pista).
func (c *Carro) DefinirAngulo(a float64) { c.angulo = a }

// Mover avança a física do carro um tick (contrato 1.1, passos 1 a 4). Carro
// que não está Vivo não se mexe. A colisão (passo 5) não é feita aqui.
func (c *Carro) Mover(ctl Controle, cfg Config) {
	if !c.Vivo {
		return
	}
	var temAcao [2]bool
	for _, a := range ctl.Acoes {
		if !a.Ativa {
			continue
		}
		d := float64(a.Sinal) * cfg.Aceleracao * cfg.DT
		switch a.Eixo {
		case EixoX:
			c.Vel.X += d
			temAcao[EixoX] = true
		case EixoY:
			c.Vel.Y += d
			temAcao[EixoY] = true
		}
	}
	atrito := 1 - cfg.Atrito*cfg.DT
	if !temAcao[EixoX] {
		c.Vel.X *= atrito
	}
	if !temAcao[EixoY] {
		c.Vel.Y *= atrito
	}
	c.Vel.X = max(-cfg.VMax, min(cfg.VMax, c.Vel.X))
	c.Vel.Y = max(-cfg.VMax, min(cfg.VMax, c.Vel.Y))
	if n := c.Vel.Len(); n > cfg.VMax {
		c.Vel = c.Vel.Scale(cfg.VMax / n)
	}
	c.Pos = c.Pos.Add(c.Vel.Scale(cfg.DT))
	if c.Vel.Len() >= limiarAngulo {
		c.angulo = c.Vel.Angle()
	}
}
