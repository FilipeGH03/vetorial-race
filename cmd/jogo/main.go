// Comando jogo abre a janela da Corrida Vetorial: menu, partida e editor de pista.
package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"vetorial-race/internal/jogo"
)

func main() {
	pista := flag.String("pista", "", "pista já escolhida no menu (ex.: pistas/oval.json)")
	editar := flag.String("editar", "", "abre o editor de pista (ex.: pistas/nova.json): carrega o arquivo se existir; S salva nele")
	treino := flag.String("treino", "", "abre a tela de treino visual da IA na pista (ex.: pistas/oval.json); S salva o melhor cérebro em cerebros/<pista>.json")
	seed := flag.Uint64("seed", 1, "seed do treino visual (mesma seed, mesmo resultado)")
	flag.Parse()

	ebiten.SetTPS(jogo.TPS)
	ebiten.SetWindowSize(jogo.LarguraTela, jogo.AlturaTela)

	if *treino != "" {
		t, err := jogo.NovoTreinoTela(*treino, jogo.DirCerebros, *seed)
		if err != nil {
			log.Fatal(err)
		}
		ebiten.SetWindowTitle("Corrida Vetorial - treino da IA")
		if err := ebiten.RunGame(t); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *editar != "" {
		e, err := jogo.NovoEditorTela(*editar)
		if err != nil {
			log.Fatal(err)
		}
		ebiten.SetWindowTitle("Corrida Vetorial - editor de pista")
		if err := ebiten.RunGame(e); err != nil {
			log.Fatal(err)
		}
		return
	}

	a, err := jogo.NovoApp(jogo.DirPistas, *pista)
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowTitle("Corrida Vetorial")
	if err := ebiten.RunGame(a); err != nil {
		log.Fatal(err)
	}
}
