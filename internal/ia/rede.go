package ia

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
)

// Formato do arquivo do cérebro (contrato 1.4) e forma da rede do jogo.
const (
	VersaoCerebro = 1
	AtivacaoTanh  = "tanh"
	NEntrada      = 14 // Entrada() (D20): 7 sensores, Vel.X/VMax, Vel.Y/VMax, Modulo/VMax, DirPista[0], DirPista[1]
	NSaida        = 2  // s0 (aumentar/diminuir) e s1 (qual vetor)
)

// CamadasPadrao é a rede 14-12-2 do jogo.
var CamadasPadrao = []int{NEntrada, 12, NSaida}

// ErrCerebroInvalido é devolvido por CarregarRede (e Salvar) quando o arquivo
// ou a rede não segue o contrato 1.4. Use errors.Is.
var ErrCerebroInvalido = errors.New("cérebro inválido")

// Rede é um perceptron de várias camadas: y_k = tanh(W_k·x + b_k).
// Pesos[k] tem uma linha por neurônio de saída da camada k.
type Rede struct {
	Camadas []int       // neurônios por camada, entrada primeiro: {14, 12, 2}
	Pesos   []Matriz    // Pesos[k] é L=Camadas[k+1] × C=Camadas[k]
	Vieses  [][]float64 // Vieses[k] tem Camadas[k+1] valores

	// Metadados gravados no arquivo (preenchidos pelo treino).
	Pista     string
	Geracao   int
	Pontuacao float64
}

// NovaRede cria uma rede com as camadas dadas. Pesos sorteados em
// ±sqrt(6/(entradas+saídas)) (Xavier) e vieses zero. rng nil usa seed fixa.
func NovaRede(camadas []int, rng *rand.Rand) (*Rede, error) {
	if err := validarCamadas(camadas); err != nil {
		return nil, err
	}
	if rng == nil {
		rng = rand.New(rand.NewPCG(1, 1))
	}
	r := vazia(camadas)
	for k, w := range r.Pesos {
		lim := math.Sqrt(6 / float64(camadas[k]+camadas[k+1]))
		for i := range w.D {
			w.D[i] = (rng.Float64()*2 - 1) * lim
		}
	}
	return r, nil
}

// vazia aloca a rede zerada; camadas já validadas.
func vazia(camadas []int) *Rede {
	n := len(camadas) - 1
	r := &Rede{
		Camadas: append([]int(nil), camadas...),
		Pesos:   make([]Matriz, n),
		Vieses:  make([][]float64, n),
	}
	for k := 0; k < n; k++ {
		r.Pesos[k] = NovaMatriz(camadas[k+1], camadas[k])
		r.Vieses[k] = make([]float64, camadas[k+1])
	}
	return r
}

// validarCamadas confere o contrato: ao menos entrada e saída, entrada 14,
// saída 2, nenhuma camada vazia.
func validarCamadas(camadas []int) error {
	if len(camadas) < 2 {
		return fmt.Errorf("%w: precisa de ao menos 2 camadas, veio %d", ErrCerebroInvalido, len(camadas))
	}
	for k, n := range camadas {
		if n < 1 {
			return fmt.Errorf("%w: camada %d com %d neurônios", ErrCerebroInvalido, k, n)
		}
	}
	if camadas[0] != NEntrada {
		return fmt.Errorf("%w: camada de entrada com %d neurônios, o contrato pede %d", ErrCerebroInvalido, camadas[0], NEntrada)
	}
	if ult := camadas[len(camadas)-1]; ult != NSaida {
		return fmt.Errorf("%w: camada de saída com %d neurônios, o contrato pede %d", ErrCerebroInvalido, ult, NSaida)
	}
	return nil
}

// Forward calcula a saída da rede (NSaida valores em [-1,1]) para a entrada x.
// Erro se len(x) != Camadas[0] ou se a rede estiver malformada.
func (r *Rede) Forward(x []float64) ([]float64, error) {
	if err := r.conferir(); err != nil {
		return nil, err
	}
	if len(x) != r.Camadas[0] {
		return nil, fmt.Errorf("ia: entrada com %d valores, a rede espera %d", len(x), r.Camadas[0])
	}
	atual := x
	for k, w := range r.Pesos {
		y := make([]float64, w.L)
		if err := w.MulVecEm(y, atual); err != nil {
			return nil, err
		}
		for i := range y {
			y[i] = math.Tanh(y[i] + r.Vieses[k][i])
		}
		atual = y
	}
	return atual, nil
}

// conferir vê se Pesos e Vieses combinam com Camadas (a rede pode ter sido
// montada à mão por quem usa o pacote).
func (r *Rede) conferir() error {
	if err := validarCamadas(r.Camadas); err != nil {
		return err
	}
	n := len(r.Camadas) - 1
	if len(r.Pesos) != n || len(r.Vieses) != n {
		return fmt.Errorf("%w: %d camadas pedem %d matrizes e %d vieses, há %d e %d",
			ErrCerebroInvalido, len(r.Camadas), n, n, len(r.Pesos), len(r.Vieses))
	}
	for k := 0; k < n; k++ {
		w := r.Pesos[k]
		if w.L != r.Camadas[k+1] || w.C != r.Camadas[k] || len(w.D) != w.L*w.C {
			return fmt.Errorf("%w: pesos[%d] não é %dx%d", ErrCerebroInvalido, k, r.Camadas[k+1], r.Camadas[k])
		}
		if len(r.Vieses[k]) != r.Camadas[k+1] {
			return fmt.Errorf("%w: vieses[%d] tem %d valores, esperava %d", ErrCerebroInvalido, k, len(r.Vieses[k]), r.Camadas[k+1])
		}
	}
	return nil
}

// Copia devolve uma cópia independente da rede (o treino usa para elitismo).
func (r *Rede) Copia() *Rede {
	c := &Rede{
		Camadas:   append([]int(nil), r.Camadas...),
		Pesos:     make([]Matriz, len(r.Pesos)),
		Vieses:    make([][]float64, len(r.Vieses)),
		Pista:     r.Pista,
		Geracao:   r.Geracao,
		Pontuacao: r.Pontuacao,
	}
	for k, w := range r.Pesos {
		c.Pesos[k] = Matriz{L: w.L, C: w.C, D: append([]float64(nil), w.D...)}
	}
	for k, b := range r.Vieses {
		c.Vieses[k] = append([]float64(nil), b...)
	}
	return c
}

// arquivoCerebro é o JSON do contrato 1.4.
type arquivoCerebro struct {
	Versao    int           `json:"versao"`
	Camadas   []int         `json:"camadas"`
	Ativacao  string        `json:"ativacao"`
	Pesos     [][][]float64 `json:"pesos"`
	Vieses    [][]float64   `json:"vieses"`
	Pista     string        `json:"pista"`
	Geracao   int           `json:"geracao"`
	Pontuacao float64       `json:"pontuacao"`
}

// Salvar grava a rede em caminho (contrato 1.4), criando a pasta se preciso.
// Escreve num arquivo temporário e renomeia, para nunca deixar JSON pela metade.
func (r *Rede) Salvar(caminho string) error {
	if err := r.conferir(); err != nil {
		return err
	}
	a := arquivoCerebro{
		Versao:    VersaoCerebro,
		Camadas:   r.Camadas,
		Ativacao:  AtivacaoTanh,
		Pesos:     make([][][]float64, len(r.Pesos)),
		Vieses:    r.Vieses,
		Pista:     r.Pista,
		Geracao:   r.Geracao,
		Pontuacao: r.Pontuacao,
	}
	for k, w := range r.Pesos {
		linhas := make([][]float64, w.L)
		for i := range linhas {
			linhas[i] = w.Linha(i)
		}
		a.Pesos[k] = linhas
	}
	dados, err := json.MarshalIndent(a, "", "  ") // falha com NaN/Inf
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCerebroInvalido, err)
	}
	dados = append(dados, '\n')

	dir := filepath.Dir(caminho)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".cerebro-*.tmp")
	if err != nil {
		return err
	}
	_, err = tmp.Write(dados)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), caminho)
	}
	if err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// CarregarRede lê um cérebro (contrato 1.4). Erros de formato devolvem
// ErrCerebroInvalido (errors.Is): versão ou ativação desconhecida, camadas[0]
// != 14 (D20), camadas[last] != 2, ou pesos/vieses que não combinam com as camadas.
func CarregarRede(caminho string) (*Rede, error) {
	dados, err := os.ReadFile(caminho)
	if err != nil {
		return nil, err
	}
	var a arquivoCerebro
	if err := json.NewDecoder(bytes.NewReader(dados)).Decode(&a); err != nil {
		return nil, fmt.Errorf("%w: json: %v", ErrCerebroInvalido, err)
	}
	if a.Versao != VersaoCerebro {
		return nil, fmt.Errorf("%w: versão %d, esperava %d", ErrCerebroInvalido, a.Versao, VersaoCerebro)
	}
	if a.Ativacao != AtivacaoTanh {
		return nil, fmt.Errorf("%w: ativação %q, só há %q", ErrCerebroInvalido, a.Ativacao, AtivacaoTanh)
	}
	if err := validarCamadas(a.Camadas); err != nil {
		return nil, err
	}
	n := len(a.Camadas) - 1
	if len(a.Pesos) != n || len(a.Vieses) != n {
		return nil, fmt.Errorf("%w: %d camadas pedem %d matrizes e %d vieses, o arquivo tem %d e %d",
			ErrCerebroInvalido, len(a.Camadas), n, n, len(a.Pesos), len(a.Vieses))
	}
	// Só aloca depois de conferir os tamanhos contra o que o arquivo traz:
	// camadas enormes num arquivo pequeno não viram memória.
	r := &Rede{
		Camadas:   a.Camadas,
		Pesos:     make([]Matriz, n),
		Vieses:    a.Vieses,
		Pista:     a.Pista,
		Geracao:   a.Geracao,
		Pontuacao: a.Pontuacao,
	}
	for k := 0; k < n; k++ {
		linhas, saida, entrada := a.Pesos[k], a.Camadas[k+1], a.Camadas[k]
		if len(linhas) != saida {
			return nil, fmt.Errorf("%w: pesos[%d] tem %d linhas, esperava %d", ErrCerebroInvalido, k, len(linhas), saida)
		}
		for i, linha := range linhas {
			if len(linha) != entrada {
				return nil, fmt.Errorf("%w: pesos[%d][%d] tem %d valores, esperava %d", ErrCerebroInvalido, k, i, len(linha), entrada)
			}
		}
		if len(a.Vieses[k]) != saida {
			return nil, fmt.Errorf("%w: vieses[%d] tem %d valores, esperava %d", ErrCerebroInvalido, k, len(a.Vieses[k]), saida)
		}
		w := NovaMatriz(saida, entrada)
		for i, linha := range linhas {
			copy(w.Linha(i), linha)
		}
		r.Pesos[k] = w
	}
	return r, nil
}
