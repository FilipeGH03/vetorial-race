// Package ia é a rede neural (MLP escrita à mão, sem biblioteca de ML) e o
// treino por neuroevolução. Não importa Ebitengine: roda sem janela.
package ia

import "fmt"

// Matriz é uma matriz L×C de float64 guardada por linhas em D
// (elemento (i, j) = D[i*C+j]).
type Matriz struct {
	L, C int
	D    []float64
}

// NovaMatriz cria uma matriz l×c zerada.
func NovaMatriz(l, c int) Matriz {
	return Matriz{L: l, C: c, D: make([]float64, l*c)}
}

// At devolve o elemento da linha i, coluna j.
func (m Matriz) At(i, j int) float64 { return m.D[i*m.C+j] }

// Set troca o elemento da linha i, coluna j.
func (m Matriz) Set(i, j int, v float64) { m.D[i*m.C+j] = v }

// Linha devolve a linha i (compartilha memória com a matriz).
func (m Matriz) Linha(i int) []float64 { return m.D[i*m.C : (i+1)*m.C] }

// MulVec devolve o produto M·x (vetor de tamanho L). Erro se len(x) != C.
func (m Matriz) MulVec(x []float64) ([]float64, error) {
	y := make([]float64, m.L)
	if err := m.MulVecEm(y, x); err != nil {
		return nil, err
	}
	return y, nil
}

// MulVecEm grava M·x em dst, sem alocar. Erro se len(x) != C ou len(dst) != L.
func (m Matriz) MulVecEm(dst, x []float64) error {
	if len(x) != m.C {
		return fmt.Errorf("ia: vetor com %d valores, a matriz espera %d", len(x), m.C)
	}
	if len(dst) != m.L {
		return fmt.Errorf("ia: destino com %d valores, a matriz produz %d", len(dst), m.L)
	}
	for i := 0; i < m.L; i++ {
		var soma float64
		for j, v := range m.Linha(i) {
			soma += v * x[j]
		}
		dst[i] = soma
	}
	return nil
}
