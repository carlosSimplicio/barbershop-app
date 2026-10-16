---
name: barbershop-task-planner
description: Produz planos concisos e objetivos para implementar funcionalidades, correções ou mudanças técnicas no barbershop-app. Use para identificar alterações, motivação, localização, base documental e subtarefas verificáveis; não use para executar a implementação.
---

# Planejar tarefas do barbershop-app

Produza um documento Markdown curto que permita implementar e avaliar a tarefa em partes, sem repetir extensamente a documentação do projeto.

## Linguagem

- Escreva de forma clara, direta e em português brasileiro.
- Use termos técnicos em inglês quando forem mais precisos, reconhecíveis ou naturais no contexto de desenvolvimento. Não force traduções literais que prejudiquem o sentido.
- Preserve em inglês nomes de código, símbolos, endpoints, tecnologias e termos consolidados. Por exemplo, ao conectar um componente do frontend a uma chamada do backend, use `wiring` quando esse for o termo adequado; nunca use “fiação”.
- Evite jargão desnecessário. Quando um termo em inglês não for evidente para o público esperado, explique-o brevemente em português na primeira ocorrência.

## Antes de planejar

1. Leia o `AGENTS.md` aplicável e `docs/implementation-scope.md`.
2. Siga o roteamento do `AGENTS.md` e leia somente os documentos relacionados à tarefa.
3. Inspecione código, testes, OpenAPI e migrations relacionados para confirmar o estado atual e os locais reais da mudança.
4. Diferencie requisitos documentados de decisões ainda não tomadas. Não invente regras de negócio ou arquitetura.

Sempre questione o usuário sobre qualquer dúvida, lacuna, ambiguidade, conflito ou decisão encontrada durante o planejamento. Apresente o contexto e, quando útil, as opções e seus impactos, mas nunca escolha ou assuma uma resposta por conta própria. Aguarde uma confirmação explícita antes de incorporar a decisão e concluir o documento; reúna perguntas relacionadas em uma rodada curta quando isso facilitar a resposta.

Quando uma decisão do usuário diferir da documentação vigente, aponte claramente o conflito, cite os arquivos e requisitos afetados e confirme que a decisão substitui o comportamento documentado. Depois da confirmação, inclua no plano a atualização dos documentos correspondentes na mesma entrega da implementação. Atualize `docs/product.md`, `docs/architecture.md`, `docs/operations.md` ou `docs/implementation-scope.md` conforme o assunto, e o `AGENTS.md` quando a nova regra for universal. Nunca planeje código divergente mantendo documentação sabidamente desatualizada.

Planeje somente. Não altere implementação, contrato, migrations, documentação de produto ou checklist. A única escrita autorizada pelo uso desta skill é o próprio documento de plano, conforme a seção "Arquivo do plano"; qualquer outra alteração exige pedido explícito do usuário.

## Arquivo do plano

- Quando o plano estiver concluído e sem decisões pendentes, grave-o em `docs/plans/<slug-da-tarefa>-plan.md` no repositório.
- Use um slug curto, descritivo e em inglês. Por exemplo, um plano de autenticação deve ficar em `docs/plans/authentication-plan.md`.
- Se o usuário indicar outro nome de arquivo dentro de `docs/plans/`, respeite-o.
- Se o arquivo já existir, leia-o antes e atualize o plano completo, sem criar versões paralelas ou sobrescrever conteúdo não relacionado.
- Não grave um plano apresentado como concluído enquanto houver decisões pendentes. Durante a fase de perguntas, não crie rascunhos no repositório, salvo solicitação explícita.
- Ao finalizar, informe o caminho do arquivo criado ou atualizado.

## Como decompor

- Divida o trabalho em subtarefas pequenas, coesas e avaliáveis separadamente.
- Cada subtarefa deve produzir um resultado observável e possuir sua própria forma de verificação.
- Declare dependências entre subtarefas; não descreva como independente algo que só possa ser avaliado após outra entrega.
- Prefira uma fatia vertical verificável quando ela puder ser concluída sem criar estados intermediários inconsistentes. Use etapas por camada somente quando a dependência técnica exigir.
- Agrupe alterações mecânicas inseparáveis, como atualizar o OpenAPI e regenerar seus artefatos. Nunca proponha editar código gerado diretamente.
- Inclua documentação e os checkboxes aplicáveis de `docs/implementation-scope.md` na subtarefa que conclui o comportamento correspondente.
- Não use suposições para completar o plano. Se uma informação não estiver explícita na solicitação, na documentação ou na implementação vigente, pergunte ao usuário antes de prosseguir.

## Formato da saída

Comece com o título da tarefa e uma frase descrevendo o resultado esperado. Em seguida, apresente as subtarefas numeradas neste formato:

### N. Nome orientado ao resultado

- **O que:** criação ou modificação concreta.
- **Motivação:** por que a alteração é necessária e qual comportamento ou problema ela atende.
- **Onde:** arquivos, pacotes, endpoints, tabelas, componentes ou documentos afetados. Use caminhos e símbolos confirmados pela inspeção; quando ainda não existirem, indique o diretório ou artefato previsto.
- **Base documental:** caminho e seção ou requisito que justificam a alteração. Quando uma decisão confirmada pelo usuário ainda não estiver nos `docs` ou substituir uma regra vigente, identifique a decisão e os documentos que serão criados ou atualizados na mesma entrega; não apresente uma preferência como requisito existente.
- **Verificação:** critérios observáveis e testes unitários que permitem avaliar esta subtarefa isoladamente.
- **Dependências:** somente quando existirem.

Ao final, inclua **Verificação conjunta** somente quando houver comandos ou checagens que façam sentido apenas para o conjunto da entrega. Não entregue o plano como concluído enquanto houver decisões pendentes; pergunte e aguarde as respostas primeiro.

Não acrescente introduções longas, estimativas não solicitadas, pseudocódigo, alternativas irrelevantes ou listas genéricas de boas práticas.

## Testes no plano

- Especifique testes pelo comportamento observável ou regra de negócio protegida, não pela estrutura interna da implementação.
- Cubra o fluxo esperado, recusas ou erros relevantes e invariantes que mudam o resultado para o usuário ou para o domínio.
- Para bugs, descreva o comportamento que falhava e o resultado esperado do teste de regressão.
- Evite casos esdrúxulos sem origem em requisito, risco real ou regressão conhecida, assim como testes de getters, detalhes de mock, marcação estática ou comportamento de bibliotecas.
- Não use meta de cobertura como substituto para cenários significativos.
- Mantenha o escopo atual de testes unitários com `testing` no backend e Vitest no frontend; não proponha integração, testcontainers ou E2E sem nova decisão documentada.

## Restrições recorrentes

Quando forem pertinentes à tarefa, torne explícitos no plano o isolamento por estabelecimento, papel e propriedade do recurso; a preservação de histórico; transações e concorrência; idempotência; valores monetários exatos; fusos; dados sensíveis; migrations novas; regeneração de código; e consistência entre implementação, OpenAPI, testes e documentação.
