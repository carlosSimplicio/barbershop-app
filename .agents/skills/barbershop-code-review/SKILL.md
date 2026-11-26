---
name: barbershop-code-review
description: Investiga bugs, erros lógicos, concorrência, contratos, performance e oportunidades de arquitetura, duplicação e legibilidade no barbershop-app. Use para revisar código, diff, commit, branch, feature ou implementação de plano, inclusive na revisão independente por IA do implementador. Relata achados com evidências; não implementa correções.
---

# Revisar código do barbershop-app

Priorize encontrar falhas e investigar suas condições de ocorrência. Examine bugs, lógica, concorrência, contratos e performance; depois avalie duplicações, arquitetura e legibilidade. A conferência documental integra a revisão, mas preencher critérios ou organizar o relatório não encerra a investigação. Considere todos os eixos aplicáveis e organize a entrega depois da análise.

## Escopo e referências

1. Leia o `AGENTS.md` da raiz e as instruções adicionais aplicáveis aos arquivos revisados. Siga o roteamento documental e os critérios de revisão final do projeto; consulte somente os documentos relacionados ao escopo.
2. Identifique a mudança solicitada e sua base de comparação. Respeite os arquivos, tarefa, commit ou intervalo de commits indicados. Sem alvo explícito, revise as alterações locais staged, unstaged e os arquivos novos pertinentes, preservando a distinção entre elas. Se não houver alterações locais nem outro alvo identificável, peça o escopo em vez de presumir uma comparação com `main`.
3. Leia o plano e seus requisitos de entrega quando existirem. Consulte `docs/implementation-scope.md` ao verificar conclusão ou pendências; não trate funcionalidades futuras fora da tarefa como requisitos já exigíveis.
4. Consulte outras skills somente quando sua finalidade contribuir para avaliar o escopo. Essa consulta não autoriza implementação, correções, commits ou outra revisão recursiva.
5. Inspecione o diff e o contexto necessário: chamadores, consumidores, regras de domínio, middleware, persistência, componentes compartilhados e testes. Inclua arquivos novos e dependências diretamente afetadas; não limite a análise às linhas alteradas nem transforme uma revisão delimitada em auditoria de todo o repositório.

Preserve os fontes revisados e o trabalho do usuário; não implemente correções, crie commits ou opere sobre dados de desenvolvimento ou produção. Sondagens temporárias são permitidas conforme a seção de verificações. Se faltarem referências necessárias ou houver decisões conflitantes, explicite a lacuna e o alcance parcial da revisão, sem inventar regras.

## Investigação antes do relatório

- Percorra os fluxos afetados e identifique invariantes com risco relevante. Para cada uma, formule uma condição concreta que poderia violá-la e procure a proteção efetiva nas camadas envolvidas. Inclua resultados persistidos e metadados relevantes, além do status HTTP e do estado principal.
- Relacione esses riscos às assertions reais dos testes: qual cenário exercitam, o que verificam e se falhariam diante do defeito considerado. Um teste com nome pertinente ou uma suíte verde não demonstra uma propriedade que não foi verificada.
- Investigue hipóteses plausíveis até confirmá-las, refutá-las por uma proteção concreta ou identificar uma limitação real. Escolha inspeção, rastreamento entre camadas ou reprodução conforme a dúvida. A exigência de evidência deve levar à investigação; não descarte uma hipótese somente porque ainda não foi reproduzida. Não apresente hipótese inconclusiva como bug comprovado; registre as limitações relevantes na entrega.
- Em concorrência, descreva a ordem das operações, snapshots, esperas e efeitos após adquirir bloqueios. Quando necessário, construa uma reprodução com coordenação observável, sem depender apenas de disparar operações simultâneas ou de atrasos fixos.
- Confira o comportamento efetivo de bibliotecas e banco contra o contrato da aplicação. A presença de uma opção de validação, transação ou operação idempotente não comprova, por si só, todas as propriedades esperadas. Em performance, examine também trabalho repetido e efeitos físicos quando houver custo relevante no caminho executado.
- Depois da investigação funcional, compare implementações equivalentes, variantes e seus consumidores e responsabilidades acumuladas. Recomende compartilhamento ou extração somente quando reduzir manutenção com preservação dos contratos; não use tamanho de arquivo ou semelhança visual como justificativa isolada.

## Eixos obrigatórios de análise

| Eixo | O que verificar |
| --- | --- |
| Arquitetura e melhorias arquiteturais | Responsabilidades, direção das dependências, limites dos slices e separação entre domínio e bordas HTTP/persistência. Procure acoplamento evitável, abstrações desnecessárias e oportunidades concretas de simplificação; compare alternativas com os padrões existentes antes de sugerir novas camadas ou dependências. |
| Lógica e outros bugs | Invariantes, validações, normalização, limites, cálculos, datas/fusos, estados finais e propagação de erros. Examine ausência, `null`, zero, `false` e vazio conforme o contrato; procure falhas de recursos, cancelamento, timeout e casos-limite com risco identificável. |
| Fluxos | Percorra as jornadas afetadas do início ao fim, incluindo sucesso, recusa, falha parcial e recuperação. Confira transições, efeitos persistidos, mensagens, redirecionamentos, permissões, expiração de acesso e consistência entre o que a interface anuncia e o que o backend confirmou. |
| Concorrência e atomicidade | Verifique transações, snapshots, ordem de locks, revalidação após espera, perda de atualização, idempotência e rollback. No frontend, confira respostas fora de ordem, sessão, cancelamento e recuperação de falhas conforme `docs/architecture.md`; descreva interleavings concretos. |
| Segurança, autorização e isolamento | Confira tenant, papel e propriedade no backend em cada caminho afetado, inclusive consultas e erros. Verifique dados sensíveis, tokens transacionais, sessão, CSRF e acesso público conforme as decisões do projeto. Preserve histórico e referências de agendamentos e financeiro. |
| Adequação aos padrões e contratos | Compare com `AGENTS.md`, documentação aplicável, código próximo e skills pertinentes. Confira OpenAPI, código gerado, SQL, migrations novas, documentação e checklists; no frontend, verifique design system, estados visuais, acessibilidade e textos em português brasileiro. Não altere a decisão documentada para justificar uma divergência. |
| Duplicação de código | Procure regras, validações, limites, mecanismos transversais e fluxos equivalentes repetidos, considerando também os módulos existentes. Diferencie repetição com o mesmo contrato de semelhança entre regras de domínios distintos; só proponha compartilhamento que preserve suas diferenças e reduza manutenção. |
| Testes relevantes | Relacione requisitos e riscos a cenários e assertions reais. Confira fluxo esperado, recusas, regressões e invariantes afetadas, especialmente autorização, tenant, concorrência e rollback quando pertinentes. Avalie se os testes falhariam diante do defeito, se são determinísticos e se usam o nível adequado; quantidade de testes ou percentual de cobertura não substituem essa análise. |
| Legibilidade | Avalie nomes, responsabilidades, tamanho e coesão de funções/componentes, fluxo de controle, estados implícitos e tratamento de erros. Procure complexidade que dificulte entender ou alterar o comportamento; comentários devem explicar decisões não evidentes, sem compensar abstrações confusas. |
| Possíveis refatorações | Identifique mudanças locais que reduzam complexidade, acoplamento ou duplicação sem alterar contratos. Indique o trecho, o benefício e os testes que preservariam o comportamento. Separe refatoração necessária para resolver um problema de melhoria opcional; evite reescritas e generalizações especulativas. |
| Performance e recursos | Procure consultas repetidas ou sem filtros adequados, paginação ausente, trabalho redundante, renderizações excessivas, recursos não liberados e crescimento não limitado de memória ou dados. Justifique o impacto pelo caminho executado e pelo volume plausível, sem exigir otimização sem evidência. |

## Verificações e evidências

- Confira as verificações exigidas pelo `AGENTS.md` e pelo plano. Diferencie resultados fornecidos pelo implementador de comandos executados pelo revisor. Reexecute suítes quando resolverem uma dúvida concreta; resultados anteriores não dispensam sondagens para propriedades sem cobertura.
- Use modos de checagem no checkout revisado, sem reformatar, regenerar ou aplicar migrations no banco local. Para investigar uma hipótese, pode criar scripts ou testes temporários em `/tmp` ou numa cópia isolada, executar contra a infraestrutura descartável prevista pelo projeto e remover apenas os artefatos que criou. Não altere fontes, testes ou gerados do checkout revisado, nem sobrescreva alterações do usuário.
- Prefira o nível mais baixo que reproduza o risco com fidelidade. No backend, use regras puras quando suficientes e integração sob a tag `integration` com PostgreSQL descartável, migrations reais e HTTP real quando o problema depender dessas fronteiras. No frontend, use Vitest; não introduza E2E ou integração de frontend sem nova decisão.
- Para lacunas relevantes de teste, informe cenário, invariante, resultado esperado e nível adequado da regressão. Não exija testar detalhes internos ou bibliotecas sem relação com um contrato da aplicação. Informe falhas de infraestrutura, sondagens inconclusivas e verificações não executadas.

## Relatório

Escreva em português brasileiro, com achados primeiro e em ordem de impacto. Ajuste a extensão ao escopo; não imponha quantidade mínima de achados nem tabela de cobertura obrigatória. Preencher seções ou listar testes aprovados não substitui evidências sobre o comportamento.

Para cada achado, apresente severidade, arquivo e linha, cenário ou evidência, consequência, requisito ou risco relacionado e correção recomendada. Use severidade crítica para comprometimento grave ou perda de dados, alta para quebra de fluxo ou invariante central, média para defeito limitado ou violação concreta de padrão e baixa para problema localizado de manutenção. Ajuste a classificação ao impacto e às condições reais de ocorrência.

Separe os problemas que precisam de correção das oportunidades opcionais de arquitetura, duplicação, legibilidade ou refatoração. Uma violação concreta das regras do projeto pode exigir correção mesmo sem bug funcional; uma preferência pessoal de estilo não é um defeito. Identifique problemas preexistentes relevantes e diferencie-os dos introduzidos pela mudança. Sugestões fora do escopo devem ser registradas como tal, sem bloquear automaticamente a tarefa.

Depois dos achados, registre de forma concisa:

- **Oportunidades de refatoração:** separe melhorias opcionais, com trecho, evidência, benefício, contratos e testes a preservar. Se nenhuma se justificar, resuma as comparações relevantes e por que manter a estrutura; não crie recomendações para preencher a seção.
- **Evidências e limites:** destaque os riscos relevantes investigados e as propriedades que as verificações realmente cobrem. Informe comandos e sondagens executados, resultados fornecidos, referências consultadas e lacunas. Justifique eixos que não se apliquem ou não tenham sido verificados, sem exigir uma linha para cada eixo avaliado.
- **Conclusão:** indique correções necessárias, ausência de problemas encontrados no escopo ou revisão parcial. Explicite hipóteses relevantes não resolvidas; não declare aprovação nem ausência garantida de bugs.

Quando usada pelo implementador de planos, esta skill produz o relatório para o agente implementador resolver o gate. O revisor não assume a implementação, não aprova commits em nome do usuário e não solicita revisão manual entre tarefas.
