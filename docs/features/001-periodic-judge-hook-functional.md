# 001-periodic-judge-hook — Especificação Funcional

## ID
`001-periodic-judge-hook`

## Objetivo do Produto
Permitir que o assistente (Judge) acompanhe a partida de League of Legends e emita pareceres táticos rápidos sempre que o relógio interno do jogo atingir marcos absolutos de 5 minutos, sem violar as diretrizes da Vanguard ou os Termos de Serviço do jogo.

## Contexto
O jogador quer receber insights oportunos durante a partida para ajustar decisões macro (rotas, objetivos, recalls, power spikes). Para isso, o sistema deve observar o estado atual do jogo por meio da Live Client Data API local (porta 2999) e acionar o Judge apenas nos momentos relevantes, começando pelos marcos de 5 minutos.

## User Stories

- **EU COMO** jogador de League of Legends
  **GOSTARIA** que o Judge se ativasse automaticamente aos 5, 10, 15 minutos etc. da partida
  **PARA** receber um parecer tático curto sem precisar pedir manualmente.

- **EU COMO** mantenedor do projeto
  **GOSTARIA** de registrar novos gatilhos (hooks) sem alterar o núcleo do Judge
  **PARA** adicionar eventos futuros (ex.: morte, dragão, baron) de forma isolada.

- **EU COMO** jogador
  **GOSTARIA** que o sistema usasse apenas a Live Client Data API oficial
  **PARA** manter a conta em conformidade com a Vanguard e os Termos de Serviço.

## Descrição Funcional

### Hook periódico de 5 minutos
- O sistema consulta periodicamente o estado do jogo pela Live Client Data API (porta 2999).
- Quando detecta que uma partida está ativa, passa a monitorar o tempo de jogo (`gameTime`).
- A cada vez que `gameTime` cruza um marco absoluto de 5 minutos (05:00, 10:00, 15:00, 20:00...), o hook dispara exatamente uma vez por marco.
- O gatilho é absoluto ao relógio do jogo, não relativo ao início da observação: se o sistema for iniciado aos 07:30, o próximo disparo será aos 10:00, e os demais marcos anteriores são ignorados.
- Para evitar disparos duplicados, o hook registra o último marco processado e só dispara quando um novo marco é atingido.

### Detecção de partida ativa
- Uma partida é considerada ativa quando a Live Client Data API responde com dados válidos de jogo (estado de campeões, tempo de jogo, etc.).
- O sistema não deve tentar acionar o Judge fora de uma partida ativa.

## Arquitetura de Hooks Extensível

- **Registro de hooks:** novos gatilhos podem ser cadastrados em uma lista de hooks independentes, cada um com sua própria regra de ativação.
- **Pipeline reutilizável do Judge:** o núcleo do Judge recebe uma requisição padronizada composta por (contexto do jogo + instrução do hook) e devolve uma resposta curta. Ele não conhece detalhes do gatilho.
- **Desacoplamento:** o módulo de gatilho decide *quando* perguntar; o Judge decide *como* responder. Nenhum hook específico deve estar embutido no código do Judge.
- **Contrato mínimo:** cada hook informa qual contexto deve ser coletado e qual pergunta deve ser enviada ao Judge.

## Exemplo de Payload Mastigado para o LLM

O payload entregue ao Judge deve conter apenas informações já expostas pela Live Client Data API, organizadas para facilitar a resposta:

- **Tempo de jogo:** minuto atual do marco (ex.: 05:00).
- **Matchup da rota do jogador:** campeão aliado, campeão inimigo, níveis, CS (minions abatidos).
- **KDA:** abates, mortes, assistências do jogador e, quando disponível, do oponente da rota.
- **Ouro:** ouro atual e, se disponível, ouro do oponente da rota.
- **Itens:** itens comprados pelo jogador e pelos principais oponentes visíveis.
- **Objetivos:** torres, dragões, barons, arautos — indicando quais equipes os controlam.
- **Estado geral da partida:** placar global, campeões vivos/mortos, posse de buffs relevantes.
- **Pergunta do hook:** qual pergunta específica o Judge deve responder para aquele marco.

## System Prompt do Judge

> Você é um assistente tático de League of Legends. Sua função é analisar o estado atual da partida e responder com **uma única frase curta e acionável** (máximo 140 caracteres). Foque em macro: objetivos, recalls, power spikes, rotação ou avisos de risco. Não dê explicações longas, não use listas numeradas e não repita dados óbvios. Seja direto, como um coach no ouvido do jogador.

## Critérios de Aceitação

1. **Dado** que o sistema está em execução **Quando** uma partida ativa é detectada pela Live Client Data API **Então** o monitoramento do relógio do jogo é iniciado.

2. **Dado** que uma partida está ativa **Quando** o relógio do jogo atinge 05:00 pela primeira vez **Então** o Judge é acionado exatamente uma vez.

3. **Dado** que o marco de 05:00 já foi processado **Quando** o relógio do jogo atinge 10:00 **Então** o Judge é acionado novamente, uma única vez, e assim sucessivamente a cada 5 minutos.

4. **Dado** que o sistema foi iniciado no meio da partida, por exemplo aos 07:30 **Quando** o relógio atinge 10:00 **Então** o Judge dispara no primeiro marco futuro, sem processar 05:00 retroativamente.

5. **Dado** que o sistema perde a conexão com a Live Client Data API **Quando** a partida termina ou a API fica indisponível **Então** nenhum novo disparo ocorre até que uma nova partida ativa seja detectada.

6. **Dado** que o Judge foi acionado **Quando** o payload mastigado é enviado **Então** a resposta do LLM deve ter no máximo 140 caracteres e conter apenas uma frase acionável.

7. **Dado** que um novo hook precisa ser adicionado **Quando** ele é registrado na lista de hooks **Então** ele reutiliza o pipeline do Judge sem exigir mudanças no núcleo.

8. **Dado** que o sistema está coletando dados **Quando** qualquer informação é lida **Então** ela deve vir exclusivamente da Live Client Data API (porta 2999), sem leitura de memória, processos ou arquivos do cliente do jogo.

---

_Assinatura de consumo: Produteiro — 1 ciclo, ~4.000 tokens estimados._
