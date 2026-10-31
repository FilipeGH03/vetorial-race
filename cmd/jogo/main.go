// Comando jogo abre a janela da Corrida Vetorial.
package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"vetorial-race/internal/jogo"
)

func main() {
	pista := flag.String("pista", "pistas/oval.json", "arquivo da pista")
	editar := flag.String("editar", "", "abre o editor de pista (ex.: pistas/nova.json): carrega o arquivo se existir; S salva nele")
	flag.Parse()

	if *editar != "" {
		e, err := jogo.NovoEditorTela(*editar)
		if err != nil {
			log.Fatal(err)
		}
		ebiten.SetTPS(jogo.TPS)
		ebiten.SetWindowSize(jogo.LarguraTela, jogo.AlturaTela)
		ebiten.SetWindowTitle("Corrida Vetorial - editor de pista")
		if err := ebiten.RunGame(e); err != nil {
			log.Fatal(err)
		}
		return
	}

	j, err := jogo.Novo(*pista)
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
