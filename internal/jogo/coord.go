// Package jogo reúne a parte de tela/entrada do jogo. Regras (física, colisão,
// voltas, laser) moram em internal/sim.
package jogo

// Dimensões da janela e passo fixo.
const (
	LarguraTela = 1280
	AlturaTela  = 720
	TPS         = 60
)

// MundoParaTela converte coordenadas matemáticas (y para cima) em coordenadas de
// tela (y para baixo): telaY = altura - mundoY (D3).
func MundoParaTela(x, y float64) (float64, float64) {
	return x, AlturaTela - y
}

// TelaParaMundo é a inversa de MundoParaTela.
func TelaParaMundo(x, y float64) (float64, float64) {
	return x, AlturaTela - y
}

// AnguloTela devolve o ângulo de rotação do sprite na tela para um ângulo de
// mundo (anti-horário a partir de +x). Como a tela inverte y, gira-se por -angulo.
func AnguloTela(angulo float64) float64 {
	return -angulo
}
