---
name: barbershop-code-review
description: Revisa código e mudanças do barbershop-app quanto a arquitetura, lógica, fluxos, concorrência, bugs, padrões do projeto, duplicação, testes, legibilidade e refatorações. Use quando o usuário pedir revisão de código, diff, commit, branch ou feature e como revisão independente por IA das tarefas de um plano. A revisão inspeciona e relata; não implementa correções.
---

# Revisar código do barbershop-app

Avalie o comportamento e a manutenção do escopo solicitado com evidências do código, dos contratos, dos testes e da documentação. Considere todos os eixos abaixo e registre quando algum não se aplicar ou não puder ser verificado.

## Escopo e referências

1. Leia o `AGENTS.md` da raiz e as instruções adicionais aplicáveis aos arquivos revisados. Siga o roteamento documental e os critérios de revisão final do projeto; consulte somente os documentos relacionados ao escopo.
2. Identifique a mudança solicitada e sua base de comparação. Respeite os arquivos, tarefa, commit ou intervalo de commits indicados. Sem alvo explícito, revise as alterações locais staged, unstaged e os arquivos novos pertinentes, preservando a distinção entre elas. Se não houver alterações locais nem outro alvo identificável, peça o escopo em vez de presumir uma comparação com `main`.
3. Leia o plano e seus requisitos de entrega quando existirem. Consulte `docs/implementation-scope.md` ao verificar conclusão ou pendências; não trate funcionalidades futuras fora da tarefa como requisitos já exigíveis.
4. Identifique outras skills disponíveis pertinentes ao escopo, leia seus `SKILL.md` e aplique os critérios relevantes à avaliação. Respeite a finalidade e os limites de cada skill; essa consulta não autoriza executar implementação, edição ou commits nem solicitar outra revisão recursivamente.
5. Inspecione o diff e o contexto necessário: chamadores, consumidores, regras de domínio, middleware, persistência, componentes compartilhados e testes. Inclua arquivos novos e dependências diretamente afetadas; não limite a análise às linhas alteradas nem transforme uma revisão delimitada em auditoria de todo o repositório.

Não edite arquivos, aplique correções, crie commits ou execute operações sobre dados de desenvolvimento ou produção durante a revisão. Verificações podem usar a infraestrutura descartável prevista pelo projeto. Se faltarem referências necessárias ou houver decisões conflitantes, descreva a lacuna e marque a revisão como parcial, sem inventar uma regra para concluir a avaliação.

## Eixos obrigatórios de análise

| Eixo | O que verificar |
| --- | --- |
| Arquitetura e melhorias arquiteturais | Responsabilidades, direção das dependências, limites dos slices e separação entre domínio e bordas HTTP/persistência. Procure acoplamento evitável, abstrações desnecessárias e oportunidades concretas de simplificação; compare alternativas com os padrões existentes antes de sugerir novas camadas ou dependências. |
| Lógica e outros bugs | Invariantes, validações, normalização, limites, cálculos, datas/fusos, estados finais e propagação de erros. Examine ausência, `null`, zero, `false` e vazio conforme o contrato; procure falhas de recursos, cancelamento, timeout e casos-limite com risco identificável. |
| Fluxos | Percorra as jornadas afetadas do início ao fim, incluindo sucesso, recusa, falha parcial e recuperação. Confira transições, efeitos persistidos, mensagens, redirecionamentos, permissões, expiração de acesso e consistência entre o que a interface anuncia e o que o backend confirmou. |
| Concorrência e atomicidade | Simule operações simultâneas e respostas fora de ordem. Verifique transações, revalidação sob bloqueio, ordem de locks, conflitos, perda de atualização, idempotência, repetição de efeitos e rollback. No frontend, confira coordenação de consultas/mutações, sessão, cancelamento e reconciliação de resultado incerto conforme `docs/architecture.md`; procure interleavings concretos, não apenas uso de goroutines. |
| Segurança, autorização e isolamento | Confira tenant, papel e propriedade no backend em cada caminho afetado, inclusive consultas e erros. Verifique dados sensíveis, tokens transacionais, sessão, CSRF e acesso público conforme as decisões do projeto. Preserve histórico e referências de agendamentos e financeiro. |
| Adequação aos padrões e contratos | Compare com `AGENTS.md`, documentação aplicável, código próximo e skills pertinentes. Confira OpenAPI, código gerado, SQL, migrations novas, documentação e checklists; no frontend, verifique design system, estados visuais, acessibilidade e textos em português brasileiro. Não altere a decisão documentada para justificar uma divergência. |
| Duplicação de código | Procure regras, validações, limites, mecanismos transversais e fluxos equivalentes repetidos, considerando também os módulos existentes. Diferencie repetição com o mesmo contrato de semelhança entre regras de domínios distintos; só proponha compartilhamento que preserve suas diferenças e reduza manutenção. |
| Testes relevantes | Relacione requisitos e riscos a cenários e assertions reais. Confira fluxo esperado, recusas, regressões e invariantes afetadas, especialmente autorização, tenant, concorrência e rollback quando pertinentes. Avalie se os testes falhariam diante do defeito, se são determinísticos e se usam o nível adequado; quantidade de testes ou percentual de cobertura não substituem essa análise. |
| Legibilidade | Avalie nomes, responsabilidades, tamanho e coesão de funções/componentes, fluxo de controle, estados implícitos e tratamento de erros. Procure complexidade que dificulte entender ou alterar o comportamento; comentários devem explicar decisões não evidentes, sem compensar abstrações confusas. |
| Possíveis refatorações | Identifique mudanças locais que reduzam complexidade, acoplamento ou duplicação sem alterar contratos. Indique o trecho, o benefício e os testes que preservariam o comportamento. Separe refatoração necessária para resolver um problema de melhoria opcional; evite reescritas e generalizações especulativas. |
| Performance e recursos | Procure consultas repetidas ou sem filtros adequados, paginação ausente, trabalho redundante, renderizações excessivas, recursos não liberados e crescimento não limitado de memória ou dados. Justifique o impacto pelo caminho executado e pelo volume plausível, sem exigir otimização sem evidência. |

## Verificações e evidências

- Confira as verificações exigidas pelo `AGENTS.md` e pelo plano para as camadas afetadas. Diferencie resultados fornecidos pelo implementador de comandos executados pelo revisor; não repita suítes já comprovadas sem uma dúvida concreta que a execução resolva.
- Quando necessário, execute verificações proporcionais ao risco sem modificar os fontes; use modos de checagem, não comandos que reformatem, regenerem ou apliquem migrations no banco local. Informe falhas de infraestrutura, verificações não executadas e resultados anteriores à última alteração relevante.
- No backend, avalie unitários com `testing` para regras puras e integração sob a tag `integration`, com PostgreSQL descartável, migrations reais e fluxos HTTP reais. No frontend, avalie unitários com Vitest; não exija E2E nem integração de frontend sem nova decisão documentada.
- Para cada lacuna de teste, descreva o cenário, a regra ou risco protegido, o resultado esperado e o nível mais baixo capaz de reproduzi-lo com fidelidade. Evite exigir testes de detalhes internos, bibliotecas ou casos sem requisito ou risco concreto.
- Antes de relatar um defeito, trace suas condições de ocorrência e confira proteções existentes em outras camadas. Para concorrência, descreva a ordem das operações que viola a invariante. Não apresente suspeitas sem confirmação como bugs comprovados.

## Relatório

Escreva em português brasileiro, com achados primeiro e em ordem de impacto. Não imponha quantidade mínima de achados.

Para cada achado, apresente severidade, arquivo e linha, cenário ou evidência, consequência, requisito ou risco relacionado e correção recomendada. Use severidade crítica para comprometimento grave ou perda de dados, alta para quebra de fluxo ou invariante central, média para defeito limitado ou violação concreta de padrão e baixa para problema localizado de manutenção. Ajuste a classificação ao impacto e às condições reais de ocorrência.

Separe os problemas que precisam de correção das oportunidades opcionais de arquitetura, duplicação, legibilidade ou refatoração. Uma violação concreta das regras do projeto pode exigir correção mesmo sem bug funcional; uma preferência pessoal de estilo não é um defeito. Identifique problemas preexistentes relevantes e diferencie-os dos introduzidos pela mudança. Sugestões fora do escopo devem ser registradas como tal, sem bloquear automaticamente a tarefa.

Depois dos achados, inclua:

- Uma tabela curta de cobertura dos eixos obrigatórios, indicando `avaliado`, `não se aplica` ou `não verificado`, com evidência ou justificativa. `Avaliado` indica análise realizada, não ausência de problemas.
- Verificações executadas, resultados fornecidos e limitações; instruções, documentos e skills efetivamente consultados.
- Uma conclusão que indique se há correções necessárias, se não foram encontrados problemas no escopo ou se a revisão ficou parcial. Se houver lacunas que impeçam concluir, explicite-as; não declare aprovação nem ausência garantida de bugs.

Quando usada pelo implementador de planos, esta skill produz o relatório para o agente implementador resolver o gate. O revisor não assume a implementação, não aprova commits em nome do usuário e não solicita revisão manual entre tarefas.
