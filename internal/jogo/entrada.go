package jogo

import (
	"github.com/hajimehoshi/ebiten/v2"

	"vetorial-race/internal/sim"
)

// Teclas é o estado das quatro direções num tick, já juntando WASD e as setas.
type Teclas struct {
	Cima, Baixo, Esquerda, Direita bool
}

// TeclasDe lê as direções de uma função que diz se a tecla está pressionada
// (ebiten.IsKeyPressed no jogo; um mapa nos testes). W/↑ = cima, S/↓ = baixo,
// A/← = esquerda, D/→ = direita.
func TeclasDe(pressionada func(ebiten.Key) bool) Teclas {
	return Teclas{
		Cima:     pressionada(ebiten.KeyW) || pressionada(ebiten.KeyArrowUp),
		Baixo:    pressionada(ebiten.KeyS) || pressionada(ebiten.KeyArrowDown),
		Esquerda: pressionada(ebiten.KeyA) || pressionada(ebiten.KeyArrowLeft),
		Direita:  pressionada(ebiten.KeyD) || pressionada(ebiten.KeyArrowRight),
	}
}

// ControleDeTeclas converte as teclas em um Controle (contrato 1.1): D/→ = +X,
// A/← = -X, W/↑ = +Y, S/↓ = -Y (y para cima, D3). Duas teclas em eixos
// diferentes dão duas ações. Teclas opostas no mesmo eixo se anulam: o eixo fica
// sem ação e o atrito age. Nunca há mais de uma Acao por eixo (D11).
func ControleDeTeclas(t Teclas) sim.Controle {
	var ctl sim.Controle
	if s := sinal(t.Direita, t.Esquerda); s != 0 {
		ctl.Acoes = append(ctl.Acoes, sim.Acao{Ativa: true, Eixo: sim.EixoX, Sinal: s})
	}
	if s := sinal(t.Cima, t.Baixo); s != 0 {
		ctl.Acoes = append(ctl.Acoes, sim.Acao{Ativa: true, Eixo: sim.EixoY, Sinal: s})
	}
	return ctl
}

// sinal devolve +1 se só "mais" está ativo, -1 se só "menos", 0 nos outros casos.
func sinal(mais, menos bool) int8 {
	switch {
	case mais && !menos:
		return 1
	case menos && !mais:
		return -1
	}
	return 0
}
