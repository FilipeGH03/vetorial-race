package sim

// Config reúne as constantes da simulação. Jogo e treino usam o mesmo Config.
type Config struct {
	DT         float64 // passo fixo, em segundos (1/60)
	Aceleracao float64 // px/s² aplicada por ação ativa
	Atrito     float64 // 1/s: sem ação num eixo, v *= (1 - Atrito*DT)
	VMax       float64 // px/s: limite de cada componente e do módulo

	// AlcanceSensor é o comprimento dos raios dos sensores (contrato 1.3), em px.
	AlcanceSensor float64
	// BaterMata escolhe o que acontece ao bater na parede (D2): false (corrida)
	// zera a velocidade e devolve o carro à última posição válida; true (treino)
	// também faz isso, mas o carro morre (Vivo = false).
	BaterMata bool
}

// ConfigPadrao devolve os valores dos contratos 1.1 e 1.3, com BaterMata = false
// (corrida); o treino liga BaterMata.
func ConfigPadrao() Config {
	return Config{
		DT:            1.0 / 60.0,
		Aceleracao:    300,
		Atrito:        0.8,
		VMax:          400,
		AlcanceSensor: 300,
	}
}
