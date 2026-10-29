// Package assets embute os arquivos de imagem do jogo.
package assets

import _ "embed"

// CarroAzulPNG é o sprite do carro: ~128 px de comprimento, fundo transparente,
// frente apontando para +x (D3/D4).
//
//go:embed carro_azul.png
var CarroAzulPNG []byte
