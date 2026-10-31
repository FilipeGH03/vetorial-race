// Comando treino evolui as redes da IA numa pista, sem janela (D5), e grava o
// melhor cérebro (contrato 1.4).
//
//	go run ./cmd/treino -pista pistas/oval.json -geracoes 30 -seed 1
package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"vetorial-race/internal/ia"
	"vetorial-race/internal/sim"
)

func main() {
	log.SetFlags(0)
	pista := flag.String("pista", "pistas/oval.json", "arquivo da pista")
	geracoes := flag.Int("geracoes", 50, "quantas gerações avaliar")
	seed := flag.Uint64("seed", 1, "seed do sorteio (mesma seed, mesmo resultado)")
	saida := flag.String("saida", "", "arquivo do cérebro (padrão: cerebros/<nome da pista>.json)")
	camadas := flag.String("camadas", juntar(ia.CamadasPadrao), "neurônios por camada, entrada 10 e saída 2 (ex.: 10,16,8,2)")
	populacao := flag.Int("populacao", ia.ParamsPadrao().Tamanho, "indivíduos por geração")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "uso: treino [-pista arq] [-geracoes n] [-seed n] [-saida arq] [-camadas 10,12,2] [-populacao n]")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := rodar(*pista, *geracoes, *seed, *saida, *camadas, *populacao); err != nil {
		log.Fatal("treino: ", err)
	}
}

func rodar(arqPista string, geracoes int, seed uint64, saida, camadas string, populacao int) error {
	if geracoes < 1 {
		return fmt.Errorf("-geracoes %d: precisa de ao menos 1", geracoes)
	}
	forma, err := separar(camadas)
	if err != nil {
		return err
	}
	p, err := sim.CarregarPista(arqPista)
	if err != nil {
		return err
	}
	t, err := ia.NovoTreino(p)
	if err != nil {
		return err
	}
	params := ia.ParamsPadrao()
	params.Camadas = forma
	params.Seed = seed
	params.Tamanho = populacao
	pop, err := ia.NovaPopulacao(params)
	if err != nil {
		return err
	}
	if saida == "" {
		saida = filepath.Join("cerebros", p.Nome+".json")
	}

	fmt.Printf("pista %s (%.0f px por volta), rede %v, %d indivíduos, seed %d, %.0f s cada\n",
		p.Nome, p.Comprimento, forma, populacao, seed, ia.TempoMaxTreino)
	var melhor *ia.Rede
	primeiraVolta := -1
	inicio := time.Now()
	for g := 0; ; g++ {
		ini := time.Now()
		res, err := t.AvaliarPopulacao(pop, 0)
		if err != nil {
			return err
		}
		r, nota := pop.Melhor()
		b := res[pop.Ranking()[0]]
		fmt.Printf("geração %3d  melhor %9.1f  distância %7.0f  voltas %2d  1ª volta %5s  %s  (%.1fs)\n",
			pop.Geracao, nota, b.Distancia, b.Voltas, tempoVolta(b), fim(b), time.Since(ini).Seconds())
		if b.Voltas >= 1 && primeiraVolta < 0 {
			primeiraVolta = pop.Geracao
		}
		if melhor == nil || nota > melhor.Pontuacao {
			melhor = r.Copia()
		}
		if g == geracoes-1 {
			break
		}
		// Proxima deixa os filhos com nota 0: a avaliação do começo do laço
		// vem antes do próximo Proxima.
		if err := pop.Proxima(); err != nil {
			return err
		}
	}

	melhor.Pista = p.Nome
	if err := melhor.Salvar(saida); err != nil {
		return err
	}
	if primeiraVolta >= 0 {
		fmt.Printf("1ª volta completa na geração %d\n", primeiraVolta)
	} else {
		fmt.Println("nenhum indivíduo completou uma volta")
	}
	fmt.Printf("melhor: geração %d, pontuação %.1f -> %s (%.1fs no total)\n",
		melhor.Geracao, melhor.Pontuacao, saida, time.Since(inicio).Seconds())
	return nil
}

func tempoVolta(r ia.Resultado) string {
	if r.Voltas < 1 {
		return "-"
	}
	return fmt.Sprintf("%.1fs", r.TempoVolta)
}

func fim(r ia.Resultado) string {
	if r.Bateu {
		return "bateu"
	}
	return "tempo"
}

func juntar(c []int) string {
	s := make([]string, len(c))
	for i, n := range c {
		s[i] = strconv.Itoa(n)
	}
	return strings.Join(s, ",")
}

// separar lê "10,12,2"; a forma é validada pela população (entrada 10, saída 2).
func separar(s string) ([]int, error) {
	partes := strings.Split(s, ",")
	c := make([]int, len(partes))
	for i, p := range partes {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("-camadas %q: %v", s, err)
		}
		c[i] = n
	}
	return c, nil
}
