# Corrida Vetorial

Jogo de corrida de carros controlados por vetores (Álgebra Linear), com adversários
guiados por rede neural treinada por neuroevolução. Escrito em Go com Ebitengine v2.

## Instalar o Go

Precisa do Go 1.27 ou mais novo. Não precisa de compilador C no Windows.

- Windows: `winget install GoLang.Go` (ou baixe em https://go.dev/dl/). Depois **abra um
  terminal novo** para o `go` entrar no PATH (instalação padrão em `C:\Program Files\Go\bin`).
- Confira com `go version`.

## Rodar

```
go run ./cmd/jogo
```

Abre uma janela 1280x720 a 60 TPS com o menu. Cima/Baixo (ou W/S) escolhem a linha, Esquerda/Direita
(ou A/D) mudam o valor, Enter corre, Esc sai:

- **Pista**: qualquer `pistas/*.json` (pista que não cabe em 1280x720 é recusada com aviso).
- **Modo**: Voltas (vence quem completa as voltas primeiro) ou Eliminação (um laser vermelho sai da
  largada 3 s depois do início e mata quem ficar para trás; o último vivo vence).
- **Voltas**: 1 a 10 (padrão 3). **Carros**: 2 a 8 contando você (padrão 6: você + 5 da IA).

Na corrida, WASD ou setas dirigem (duas teclas juntas dão a diagonal; bater na parede para o carro) e as
setas vermelha (x), verde (y) e amarela (resultante) mostram a velocidade. Os outros carros são
guiados pela rede de `cerebros/<pista>.json`; sem cérebro (ou com cérebro inválido) o jogo avisa na tela
e a IA fica parada. Se você morre na Eliminação, a partida acaba. Ao fim aparecem o vencedor e a
classificação: R joga de novo, Enter volta ao menu. `-pista pistas/outra.json` já escolhe a pista no menu.
Na primeira vez o `go run` baixa as dependências sozinho.

## Tela de treino

```
go run ./cmd/jogo -treino pistas/oval.json      (ou T no menu, com a pista escolhida)
```

Mostra a neuroevolução ao vivo: os 50 carros da geração andam ao mesmo tempo na pista, cada um com os 7
raios dos sensores (direções fixas do mundo) até a parede; o líder aparece em amarelo, quem parou
(bateu e ficou 1 s sem avançar, ou acabou o tempo) fica esmaecido. Cada geração tem 6 rodadas, uma por
posição de `sim.Largada(p, 6)`, e a nota é a média das 6, como em `cmd/treino`. O painel mostra
geração, rodada, melhor e média da pontuação da última geração e o melhor de todas, e o gráfico traça
as duas curvas. Espaço pausa, `+`/`-` (ou cima/baixo) mudam a velocidade (1x a 32x), `S` ou o botão
verde gravam o melhor cérebro em `cerebros/<pista>.json` (atenção: sobrescreve o cérebro que a corrida
usa), Esc sai. `-seed n` muda a seed.

## Editor de pista

```
go run ./cmd/jogo -editar pistas/minha.json
```

Arraste o mouse com o botão esquerdo para desenhar a linha central (o circuito fecha sozinho e o
sentido é o do traço). Roda do mouse ou setas cima/baixo mudam a largura (40 a 200 px). `S` salva no
arquivo, `L` recarrega, `C` ou `Esc` limpam. Se o arquivo já existe, ele é aberto para editar.
Arquivo inválido na abertura (`-editar`) abre o editor vazio com o aviso. Pista inválida (por exemplo um oito) mostra o erro e marca o cruzamento; a pista só cabe em 1280x720.
Depois: `go run ./cmd/jogo -pista pistas/minha.json` (ou escolha no menu).

## Treino da IA

```
go run ./cmd/treino -pista pistas/oval.json -seed 1
```

Roda a neuroevolução sem janela: 50 redes por geração, 300 gerações (`-geracoes n` muda). Cada rede
dirige sozinha 6 vezes, uma de cada posição da largada de 6 carros, por até 60 s cada, e a nota é a
média das 6. Bater na parede zera a velocidade e devolve o carro à última posição válida (como na
corrida), custa 5 pontos, e a volta é encerrada se passar 1 s (60 ticks) sem avançar. A nota de cada
largada é a distância percorrida + 1000 por volta + 20 por segundo poupado na 1ª volta − 5 por
batida. A rede vê 14 entradas: 7 sensores em direções fixas do mundo, a velocidade (x, y e módulo) e
a direção da pista à frente. Imprime a melhor nota de cada geração, a geração da 1ª volta completa
(e a da 1ª vez que o melhor completou a volta das 6 largadas) e grava a melhor rede em
`cerebros/<arquivo da pista sem extensão>.json` (ou em `-saida arq`). A mesma seed dá sempre o mesmo
resultado. Outras opções: `-camadas 14,16,2` (entrada 14 e saída 2) e `-populacao n`.

## Verificar

```
go build ./...
go vet ./...
go test ./...
```

## Estrutura

| pasta | conteúdo |
|---|---|
| `cmd/jogo` | executável do jogo |
| `cmd/treino` | executável do treino da rede (sem janela) |
| `internal/sim` | simulação: física, pista, colisão (sem Ebitengine) |
| `internal/jogo` | tela, entrada e desenho |
| `internal/ia` | rede neural e treino |
| `assets` | imagens (PNG embutido com `go:embed`) |
| `pistas` | pistas em JSON |
| `cerebros` | redes treinadas em JSON |

Convenções: coordenadas da simulação são matemáticas (y para cima); o desenho converte
com `telaY = altura - mundoY`. Passo fixo de 60 ticks por segundo.

## Sprite do carro

`assets/CarroAzul.svg` é o original. O Ebitengine não carrega SVG, então ele foi convertido
uma vez para `assets/carro_azul.png` (128 px de comprimento, fundo transparente, frente para +x)
com o `sharp` do Node (`resize(128)` sobre o SVG com largura/altura explícitas 960x476). Só
é preciso refazer se o SVG mudar.
