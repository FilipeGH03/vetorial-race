package sim

// Config reúne as constantes da simulação. Jogo e treino usam o mesmo Config.
type Config struct {
	DT         float64 // passo fixo, em segundos (1/60)
	Aceleracao float64 // px/s² aplicada por ação ativa
	Atrito     float64 // 1/s: sem ação num eixo, v *= (1 - Atrito*DT)
	VMax       float64 // px/s: limite de cada componente e do módulo
}

// ConfigPadrao devolve os valores do contrato 1.1.
func ConfigPadrao() Config {
	return Config{
		DT:         1.0 / 60.0,
		Aceleracao: 300,
		Atrito:     0.8,
		VMax:       400,
	}
}
