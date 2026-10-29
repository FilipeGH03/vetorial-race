// Comando jogo abre a janela da Corrida Vetorial.
package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"vetorial-race/internal/jogo"
)

func main() {
	j, err := jogo.Novo()
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetTPS(jogo.TPS)
	ebiten.SetWindowSize(jogo.LarguraTela, jogo.AlturaTela)
	ebiten.SetWindowTitle("Corrida Vetorial")
	if err := ebiten.RunGame(j); err != nil {
		log.Fatal(err)
	}
}
