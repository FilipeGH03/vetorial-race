package ia

import (
	"path/filepath"
	"testing"

	"vetorial-race/internal/sim"
)

// O que o treino mede tem de ser o que o jogo faz: a mesma rede, rodada numa
// sim.Corrida com a Config do treino (BaterMata), dá a mesma distância e o
// mesmo tick de morte que Treino.Rodar.
func TestRodarIgualACorridaDoJogo(t *testing.T) {
	tr := treinoOval(t)
	for seed := uint64(1); seed <= 4; seed++ {
		params := ParamsPadrao()
		params.Seed = seed
		params.Tamanho = 6
		pop, err := NovaPopulacao(params)
		if err != nil {
			t.Fatal(err)
		}
		for i, r := range pop.Redes {
			want := tr.Rodar(r)
			cor, err := sim.NovaCorrida(tr.Pista, tr.Cfg, sim.ModoVoltas, 1, []sim.Controlador{NovoControlador(r)})
			if err != nil {
				t.Fatal(err)
			}
			for n := 0; n < tr.MaxTicks && cor.Carros[0].Vivo && !cor.Terminada; n++ {
				cor.Step()
			}
			c := cor.Carros[0]
			if c.Progresso.Distancia != want.Distancia || c.Progresso.Voltas != want.Voltas || cor.Tick != want.Ticks {
				t.Errorf("seed %d rede %d: corrida (dist %v, voltas %d, ticks %d), treino %+v",
					seed, i, c.Progresso.Distancia, c.Progresso.Voltas, cor.Tick, want)
			}
		}
	}
}

// A nota gravada no cérebro é a que o jogo reproduz ao recarregá-lo.
func TestCerebroSalvoReproduzANota(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 10
	params.Elite = 2
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	for g := 0; g < 3; g++ {
		if _, err := tr.AvaliarPopulacao(pop, 0); err != nil {
			t.Fatal(err)
		}
		if g < 2 {
			if err := pop.Proxima(); err != nil {
				t.Fatal(err)
			}
		}
	}
	melhor, nota := pop.Melhor()
	melhor = melhor.Copia()
	melhor.Pista = "oval"
	arq := filepath.Join(t.TempDir(), "c", "oval.json")
	if err := melhor.Salvar(arq); err != nil {
		t.Fatal(err)
	}
	lido, err := CarregarRede(arq)
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.Rodar(lido).Pontuacao; got != nota {
		t.Errorf("cérebro recarregado dá %v, nota gravada %v", got, nota)
	}
}

// Mesma seed, mesmas notas (reprodutibilidade do treino, em paralelo).
func TestTreinoMesmaSeedMesmoResultado(t *testing.T) {
	tr := treinoOval(t)
	rodar := func() []float64 {
		params := ParamsPadrao()
		params.Tamanho = 12
		params.Elite = 2
		pop, err := NovaPopulacao(params)
		if err != nil {
			t.Fatal(err)
		}
		var notas []float64
		for g := 0; g < 4; g++ {
			if _, err := tr.AvaliarPopulacao(pop, 0); err != nil {
				t.Fatal(err)
			}
			_, n := pop.Melhor()
			notas = append(notas, n)
			if err := pop.Proxima(); err != nil {
				t.Fatal(err)
			}
		}
		return notas
	}
	a, b := rodar(), rodar()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("geração %d: %v x %v", i, a[i], b[i])
		}
	}
}

// População de 2 (mínimo) e Elite 0 também avaliam, mesmo com mais workers
// que indivíduos.
func TestAvaliarPopulacaoMinima(t *testing.T) {
	tr := treinoOval(t)
	params := ParamsPadrao()
	params.Tamanho = 2
	params.Elite = 0
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	res, err := tr.AvaliarPopulacao(pop, 16)
	if err != nil || len(res) != 2 {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}

// Critério de prova da C3: o treino do oval precisa levar a rede a completar
// 1 volta. Hoje isso não acontece (a IA para nas quinas, ~600-900 px de 1366).
// Roda ~15 s: pulado com -short.
func TestTreinoOvalCompletaUmaVolta(t *testing.T) {
	if testing.Short() {
		t.Skip("treino longo")
	}
	tr := treinoOval(t)
	params := ParamsPadrao() // seed 1, 50 indivíduos, 10-12-2: o comando da prova
	pop, err := NovaPopulacao(params)
	if err != nil {
		t.Fatal(err)
	}
	const geracoes = 100
	var melhor float64
	for g := 0; g < geracoes; g++ {
		res, err := tr.AvaliarPopulacao(pop, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range res {
			melhor = max(melhor, r.Distancia)
			if r.Voltas >= 1 {
				t.Logf("1ª volta na geração %d", g)
				return
			}
		}
		if err := pop.Proxima(); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("nenhum indivíduo completou 1 volta em %d gerações (melhor distância %.0f de %.0f px)", geracoes, melhor, tr.Pista.Comprimento)
}
