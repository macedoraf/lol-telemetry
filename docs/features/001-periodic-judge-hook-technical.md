# 001-periodic-judge-hook — Especificação Técnica

## 1. Visão Geral da Solução

Implementar um **pipeline de Judge acionado por hooks** sobre o módulo `riotclient` existente. A funcionalidade divide-se em três responsabilidades bem delimitadas:

- **Hooks (`internal/hooks`)**: decidem *quando* acionar o Judge. O hook periódico dispara a cada marco absoluto de 5 minutos do relógio do jogo (`gameTime`), uma única vez por marco.
- **Payload Builder (`internal/judge/payload`)**: transforma os dados brutos da Live Client Data API em um objeto mastigado para o LLM.
- **Judge (`internal/judge`)**: orquestra a chamada ao provedor LLM (OpenRouter) e retorna uma frase curta e acionável.

A interface gráfica continua sendo responsável apenas por exibir o resultado; o `cmd/lol-cli/main.go` permanece sem lógica de negócio.

## 2. Novos Pacotes e Responsabilidades

```text
lol-telemetry/
├── cmd/lol-cli/
│   └── main.go                      # entrypoint; wiring apenas
├── internal/
│   ├── types/                       # modelos compartilhados (novos tipos de domínio)
│   ├── processor/                   # métricas existentes (CS/min, GPM)
│   ├── renderer/                    # TUI existente
│   ├── hooks/                       # gatilhos (registry + periodic)
│   │   ├── registry.go              # lista de hooks registrados
│   │   └── periodic.go              # hook de marco de 5 minutos
│   └── judge/
│       ├── judge.go                 # núcleo do Judge
│       ├── payload/
│       │   └── builder.go           # monta JudgePayload
│       └── openrouter/
│           └── client.go            # cliente HTTP para OpenRouter
├── pkg/riotclient/                  # cliente da Live Client Data API (sem alterações de comportamento)
└── testdata/
    └── mocks/                       # JSONs reais da API para testes
```

### Responsabilidades detalhadas

| Pacote | Responsabilidade |
|--------|------------------|
| `internal/hooks` | Detectar condições de disparo e emitir `HookTrigger` para o orchestrator. |
| `internal/hooks/registry.go` | Manter uma lista de `Hook` registrados; permitir adicionar novos gatilhos sem tocar no Judge. |
| `internal/hooks/periodic.go` | Implementar a lógica de marco absoluto de 5 minutos com deduplicação. |
| `internal/judge` | Receber `JudgeRequest`, invocar o LLM e devolver `JudgeResponse`. |
| `internal/judge/payload` | Extrair da `AllGameData` apenas as informações relevantes e formatá-las. |
| `internal/judge/openrouter` | Fazer o POST para `/v1/chat/completions`, tratar respostas e erros. |
| `internal/orchestrator` *(opcional)* | Coordenar polling, hooks e Judge em um único loop controlável. |

## 3. Interfaces Principais

```go
// Hook decide se deve disparar com base no estado atual do jogo.
type Hook interface {
    Name() string
    ShouldFire(ctx HookContext) (bool, error)
    Instruction() string
}

// HookContext carrega o estado bruto da API e metadados do ciclo.
//
// O campo PrevFired é um mapa de hook -> último marco absoluto processado
// (em segundos). Ele é usado para deduplicar disparos dentro da mesma partida.
// Quando o sistema detecta que a partida anterior encerrou (gameTime <= 0
// ou erro na chamada da Live Client Data API), o estado de disparo deve ser
// zerado para que os marcos de uma partida futura sejam reprocessados.
type HookContext struct {
    Data      riotclient.AllGameData
    GameTime  float64
    PrevFired map[string]int64 // mapa de hook -> último marco processado
}

// Judge executa a análise a partir de um payload mastigado.
type Judge interface {
    Evaluate(ctx context.Context, req JudgeRequest) (JudgeResponse, error)
}

// PayloadBuilder transforma AllGameData em JudgePayload.
type PayloadBuilder interface {
    Build(data riotclient.AllGameData, instruction string) (JudgePayload, error)
}

// LLMClient encapsula a chamada HTTP ao provedor.
type LLMClient interface {
    Complete(ctx context.Context, prompt string) (string, error)
}
```

### Estruturas de dados do domínio

```go
type JudgeRequest struct {
    GameMinute   int
    Matchup      LaneMatchup
    KDA          PlayerKDA
    Gold         GoldSnapshot
    Items        ItemSnapshot
    Objectives   ObjectiveState
    GameState    GameSnapshot
    Question     string
    SystemPrompt string
}

type JudgeResponse struct {
    Advice string // máximo 140 caracteres
}
```

## 4. Fluxo de Dados Completo

1. **Detecção**: o `Orchestrator` chama `client.GetGameData()` a cada intervalo de polling (ex.: 1s).
2. **Validação de partida ativa**: se a chamada retornar erro ou `gameTime <= 0`, o ciclo descansa; nenhum hook é avaliado. Nesse momento, o `Orchestrator` também **reseta o mapa `PrevFired`** de cada hook, garantindo que os marcos de uma partida futura não sejam suprimidos pela deduplicação da partida anterior.
3. **Avaliação dos hooks**: cada `Hook` registrado recebe `HookContext`. O hook periódico calcula o marco atual `floor(gameTime/300)*300` e compara com o último marco processado.
4. **Disparo**: quando `ShouldFire` retorna `true`, o orchestrator gera um `Trigger` contendo o nome do hook e a instrução.
5. **Coleta**: o `PayloadBuilder` consome `AllGameData` e produz `JudgeRequest`.
6. **Payload**: o request inclui matchup de rota, KDA, ouro, itens, objetivos e a pergunta do hook.
7. **OpenRouter**: o `LLMClient` envia o system prompt + payload JSON e recebe a resposta bruta.
8. **Resposta**: o `Judge` trunca/valida a resposta em até 140 caracteres e retorna `JudgeResponse`.
9. **TUI**: o `RendererAgent` recebe `JudgeResponse` via canal e exibe o parecer no dashboard.

```
[Live Client API] --AllGameData--> [Orchestrator] --HookContext--> [Hooks]
                                         |                              |
                                         v                              v
                              [PayloadBuilder] <------------------ [Trigger]
                                         |
                                         v
                              [JudgeRequest] --> [Judge] --> [OpenRouter]
                                         |                              |
                                         v                              v
                                    [Renderer] <----------------- [JudgeResponse]
```

## 5. Integração com Código Existente

### O que reutilizar

- `pkg/riotclient.Client` e seus métodos existentes (`GetGameData`, `GetEventData`, `GetPlayerItems`, etc.).
- Modelos `riotclient.AllGameData`, `AllPlayer`, `GameData`, `Item`, etc.
- `internal/types` para novos tipos de domínio do Judge.
- `internal/renderer` para exibir as respostas; o renderer passa a ouvir um canal de `JudgeResponse`.

### O que modificar

- **`cmd/lol-cli/main.go`**: apenas instanciar e conectar os novos componentes (registry, orchestrator, renderer). Sem lógica de negócio.
- **`internal/renderer/renderer.go`**: adicionar um novo canal/mensagem Bubble Tea para receber e renderizar `JudgeResponse`.
- **`internal/types/types.go`**: adicionar tipos `JudgeRequest`, `JudgeResponse`, `HookContext`, etc.

### O que não alterar

- `pkg/riotclient` não importa `internal/` ou `cmd/` (regra vigente).
- A lógica de cálculo do `processor` permanece isolada; pode ser consumida pelo payload builder se útil, mas sem fusão de pacotes.

## 6. Estratégia de Testes

### Unitários (in-memory, sem rede)

- **Hook periódico**: table-driven tests com `gameTime` variando de 0s a 1200s. Verificar:
  - disparo exatamente uma vez por marco;
  - marcos anteriores ignorados quando o sistema inicia no meio da partida;
  - nenhum disparo fora de partida ativa (`gameTime <= 0`).
- **Payload builder**: validar extração de matchup, KDA, itens e objetivos a partir de mocks.
- **Judge**: mock de `LLMClient` garantindo que o system prompt e o payload sejam enviados corretamente.

### Integração

- `httptest` para simular o servidor OpenRouter e validar headers (`Authorization`, `Content-Type`), body JSON e parsing da resposta.
- `httptest` para simular a Live Client Data API e testar o fluxo ponta-a-ponta do orchestrator.

### Mocks

- Usar JSONs reais da API em `testdata/mocks/`.
- Criar `mockLLMClient` e `mockHook` para testes unitários puros.

## 7. Dependências

Nenhuma dependência nova é obrigatória além das já adotadas:

- `net/http` nativo para chamadas ao OpenRouter.
- `encoding/json` nativo para serialização do payload.
- `github.com/charmbracelet/bubbletea` e `github.com/charmbracelet/lipgloss` já presentes para a TUI.

A API do OpenRouter é compatível com o formato OpenAI, portanto basta montar o JSON manualmente sem bibliotecas extras. Isso mantém o binário enxuto e evita lock-in.

## 8. Critérios Técnicos de Aceitação

1. O hook periódico dispara exatamente uma vez por marco absoluto de 5 minutos (`05:00`, `10:00`, `15:00`, ...).
2. Se o sistema iniciar após um marco, o próximo disparo ocorre no primeiro marco futuro; marcos passados não são processados retroativamente.
3. O Judge só é acionado quando há uma partida ativa (`gameTime > 0` e API respondendo).
4. O payload enviado ao LLM contém: tempo de jogo, matchup de rota, KDA, ouro, itens, objetivos e pergunta do hook.
5. A resposta do LLM é limitada a no máximo 140 caracteres e exibida como uma única frase acionável.
6. Novos hooks podem ser registrados em `internal/hooks/registry.go` sem modificar o pacote `internal/judge`.
7. `pkg/riotclient` continua 100% desacoplado de `internal/` e `cmd/`.
8. `cmd/lol-cli/main.go` contém apenas wiring; nenhuma lógica de negócio é adicionada.
9. Todos os testes unitários rodam sem chamadas de rede reais.
10. A integração com OpenRouter utiliza `net/http` e `httptest` em testes.
11. O estado de deduplicação (`PrevFired`) de cada hook é zerado quando a partida anterior encerra (`gameTime <= 0` ou erro na Live Client Data API), permitindo o reprocessamento dos marcos de 5 minutos em uma nova partida.

## 9. Decisão de Identificação do Oponente

O oponente da rota do jogador será identificado pelo **campo `Position` declarado pela API** (`TOP`, `JUNGLE`, `MIDDLE`, `BOTTOM`, `UTILITY`).

- Algoritmo:
  1. Localizar o jogador ativo em `AllPlayers` pelo `SummonerName` (ou `riotId` futuramente).
  2. Ler o campo `Team` do jogador ativo.
  3. Entre os jogadores do time oposto (`Team != activeTeam`), selecionar aquele cuja `Position` seja igual à `Position` do jogador ativo.
  4. Se não houver correspondência exata (ex.: posição vazia ou jogador ausente), o payload deve indicar "oponente de rota não identificado" e ainda assim enviar os dados globais disponíveis.

Essa abordagem é determinística, usa apenas dados já expostos pela Live Client Data API e não requer heurísticas de coordenadas ou leitura de memória.

---

_Assinatura de consumo: Techlead — 1 ciclo, ~3.500 tokens estimados._
