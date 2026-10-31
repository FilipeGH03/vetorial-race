package ia

import (
	"math"

	"vetorial-race/internal/sim"
)

// ZonaMorta é o limiar da D9: com |s0| abaixo dele a rede não mexe em nenhum
// vetor no tick.
const ZonaMorta = 0.2

// vmaxEntrada é o VMax que normaliza a velocidade na entrada da rede. Jogo e
// treino usam as mesmas constantes (contrato 1.1), então é o de ConfigPadrao.
var vmaxEntrada = sim.ConfigPadrao().VMax

// Entrada monta o vetor de entrada da rede (contrato 1.3, D20), NEntrada
// valores nesta ordem: Sensores[0..6], Vel.X/VMax, Vel.Y/VMax, Modulo/VMax,
// DirPista[0].X, DirPista[0].Y, DirPista[1].X, DirPista[1].Y.
func Entrada(l sim.Leitura) []float64 {
	x := make([]float64, NEntrada)
	n := copy(x, l.Sensores[:])
	x[n] = l.Vel.X / vmaxEntrada
	x[n+1] = l.Vel.Y / vmaxEntrada
	x[n+2] = l.Modulo / vmaxEntrada
	for k, d := range l.DirPista {
		x[n+3+2*k] = d.X
		x[n+4+2*k] = d.Y
	}
	return x
}

// AcaoDaSaida converte a saída da rede (s0, s1) na Acao do tick (contrato 1.3):
// s0 > 0 aumenta (+1), senão diminui (-1); s1 > 0 mexe no EixoX, senão no
// EixoY. Na zona morta (|s0| < ZonaMorta, D9) a Acao fica inativa. Saída com
// menos de 2 valores ou s0 NaN também dá Acao inativa.
func AcaoDaSaida(s []float64) sim.Acao {
	if len(s) < NSaida || !(math.Abs(s[0]) >= ZonaMorta) {
		return sim.Acao{}
	}
	a := sim.Acao{Ativa: true, Eixo: sim.EixoY, Sinal: -1}
	if s[0] > 0 {
		a.Sinal = 1
	}
	if s[1] > 0 {
		a.Eixo = sim.EixoX
	}
	return a
}

// Controlador dirige um carro com uma rede (implementa sim.Controlador): a
// cada tick monta a Entrada da Leitura, roda a rede e devolve no máximo uma
// Acao (contrato 1.3). Não guarda estado nem altera a rede.
type Controlador struct {
	Rede *Rede
}

// NovoControlador cria o controlador da rede r.
func NovoControlador(r *Rede) *Controlador {
	return &Controlador{Rede: r}
}

// Decidir roda a rede sobre a Leitura e devolve um Controle com uma Acao
// ativa, ou Controle{} (sem nenhuma Acao) na zona morta (contrato 1.3, D9).
// Sem rede, ou com rede malformada, também não mexe em nada.
//
// Rede.Forward aloca a cada chamada; medido no oval: ~0,6 µs por tick contra
// ~5 µs do sim.Ler, então não vale reaproveitar buffers aqui.
func (c *Controlador) Decidir(l sim.Leitura) sim.Controle {
	if c == nil || c.Rede == nil {
		return sim.Controle{}
	}
	s, err := c.Rede.Forward(Entrada(l))
	if err != nil {
		return sim.Controle{}
	}
	a := AcaoDaSaida(s)
	if !a.Ativa {
		return sim.Controle{}
	}
	return sim.Controle{Acoes: []sim.Acao{a}}
}
