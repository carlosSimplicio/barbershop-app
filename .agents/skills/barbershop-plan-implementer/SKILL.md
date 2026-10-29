---
name: barbershop-plan-implementer
description: Executa um plano de implementação existente do barbershop-app por completo, tarefa por tarefa, em branch dedicada, com TDD orientado a comportamento, revisão independente por IA e um commit por tarefa, sem aprovação manual entre tarefas. Use quando o usuário pedir para implementar ou continuar um plano; não use para criar o plano nem para mudanças avulsas sem plano aprovado.
---

# Implementar planos do barbershop-app

Implemente todo o escopo do plano indicado, em sequência, preservando a rastreabilidade entre cada subtarefa e seu commit. O pedido para implementar ou continuar o plano autoriza os commits das tarefas e o avanço automático entre elas após as verificações e a revisão por IA, sem solicitar revisão ou aprovação manual a cada etapa. Continue até concluir o plano, salvo bloqueio real ou pedido do usuário para interromper.

## Antes de implementar

1. Leia o `AGENTS.md`, o plano completo e os documentos obrigatórios indicados por ambos.
2. Inspecione o código, testes, OpenAPI, migrations e estado do Git relacionados à primeira tarefa ainda pendente.
3. Não inicie enquanto o plano tiver decisões pendentes, conflitos com a implementação atual ou critérios de verificação insuficientes. Apresente o problema e aguarde decisão do usuário.
4. Garanta que o arquivo do plano esteja versionado antes da primeira tarefa. Se estiver novo ou modificado e as alterações pertencerem ao plano autorizado, crie um commit exclusivo do plano sem solicitar nova autorização; nunca o esconda dentro do commit da primeira tarefa. Se a relação das alterações com o escopo autorizado for incerta, esclareça antes de incluí-las.
5. Verifique o working tree e preserve alterações alheias. Não misture arquivos não relacionados, não descarte trabalho existente e não comece com sobreposições não resolvidas.

## Branch dedicada

- Toda implementação ocorre fora de `main` e de qualquer branch protegida ou de base.
- Use a branch indicada pelo usuário quando existir. Caso contrário, crie `feature/<slug-do-plano>` a partir da branch de base atualmente confirmada.
- Se a branch de base, o destino ou a relação das alterações locais com o plano forem ambíguos, pergunte antes de trocar ou criar a branch.
- Não faça push, abra pull request, rebase ou merge sem pedido explícito.

## Executar uma tarefa por vez

Para cada subtarefa numerada do plano:

1. Declare qual tarefa será implementada, o resultado observável esperado e as verificações aplicáveis.
2. Implemente apenas essa tarefa com o ciclo TDD abaixo.
3. Atualize na mesma tarefa OpenAPI, migrations novas, artefatos gerados, documentação e checkboxes que o plano atribuir ao comportamento. Nunca edite código gerado manualmente.
4. Execute as verificações proporcionais ao risco e todas as que o plano exigir para a tarefa.
5. Resolva o gate de revisão por IA antes de criar o commit.
6. Crie exatamente um commit focado para a tarefa, informe o resultado e avance automaticamente para a próxima, sem encerrar o turno para solicitar aprovação manual.

Não agrupe várias tarefas do plano em um commit e não divida uma tarefa em commits intermediários. Preserve no working tree apenas o trabalho da tarefa atual. Se uma tarefa se revelar grande demais ou depender de uma mudança fora do plano, pare e proponha a revisão do plano em vez de alterar silenciosamente a granularidade.

## TDD orientado a comportamento

Para toda mudança de comportamento tecnicamente testável:

1. **Red:** escreva primeiro o menor teste unitário significativo que expresse o comportamento, recusa ou invariante do plano. Execute-o e confirme que falha pelo motivo esperado, não por erro de compilação, fixture ou infraestrutura alheia.
2. **Green:** implemente o mínimo necessário para satisfazer o comportamento e execute novamente o teste.
3. **Refactor:** melhore nomes, responsabilidades, duplicações e arquitetura sem alterar o comportamento; mantenha os testes verdes.
4. Execute os testes relacionados e as verificações completas exigidas pelo `AGENTS.md` para as camadas afetadas.

Use `testing` no backend e Vitest no frontend. Teste resultados observáveis e regras de negócio, priorizando fluxo esperado, recusas, autorização, isolamento de tenant, transações, concorrência e invariantes relevantes. Não teste detalhes internos, chamadas de mock sem significado, getters, marcação estática ou comportamento de bibliotecas. Use testes de integração do backend com Testcontainers conforme o `AGENTS.md`; não introduza integração do frontend ou E2E sem nova decisão documentada.

Migrations, artefatos gerados, configuração e documentação podem não possuir um teste unitário significativo isolado. Nesses casos, declare a exceção, use as verificações determinísticas apropriadas e não crie um teste artificial apenas para simular TDD.

## Gate de revisão por IA

Depois que a tarefa estiver implementada e as verificações passarem, mas antes do commit:

- Solicite revisão a um subagente independente usando `$barbershop-code-review`, definida em [../barbershop-code-review/SKILL.md](../barbershop-code-review/SKILL.md). O pedido deve incluir a subtarefa do plano, seus requisitos de entrega, o diff, os documentos aplicáveis e os resultados das verificações, sem induzir uma conclusão.
- Instrua explicitamente o revisor a ler o `AGENTS.md` da raiz e quaisquer instruções adicionais aplicáveis aos arquivos revisados. Ele deve seguir o roteamento documental e os critérios de revisão final definidos nessas instruções, consultando os documentos obrigatórios para o escopo.
- Instrua o revisor a identificar as skills disponíveis pertinentes ao escopo revisado, ler seus respectivos `SKILL.md` e aplicar as diretrizes relevantes à avaliação. A seleção deve considerar a finalidade e os limites de cada skill; consultar uma skill não autoriza executar suas etapas de implementação, edição ou commit durante a revisão.
- Peça que o relatório identifique as instruções, documentos e skills consultados, relacione cada achado ao requisito ou risco concreto e informe lacunas de contexto ou referências obrigatórias indisponíveis. Não considere o gate resolvido enquanto faltar uma referência necessária à revisão.
- O revisor apenas inspeciona e relata; não edita arquivos nem cria commits.
- Peça que cubra todos os eixos obrigatórios da skill de revisão, priorize problemas pelo impacto e diferencie correções necessárias de melhorias opcionais. O relatório deve registrar e justificar os eixos que não se aplicarem ou não puderem ser verificados.
- Analise cada achado com evidência. Corrija os achados pertinentes ao escopo, execute novamente o ciclo e as verificações e solicite nova revisão focada quando a correção for material.
- Não substitua essa etapa por auto-revisão. Se um revisor independente não estiver disponível, pare antes do commit e informe o bloqueio.

## Acompanhamento e bloqueios

- Depois de cada commit, comunique o resultado da tarefa, as verificações, os achados relevantes da revisão por IA e suas resoluções e os riscos remanescentes. Essa atualização é informativa e não exige resposta para continuar.
- Se o usuário pedir mudanças durante a execução, incorpore-as ao escopo autorizado, repita as verificações aplicáveis e refaça a revisão por IA quando a alteração for material.
- Pare e solicite uma decisão quando houver decisão de negócio pendente, conflito documental, necessidade de ampliar ou redividir o plano, sobreposição com trabalho alheio ou bloqueio que impeça concluir as verificações ou a revisão independente. Informe a tarefa afetada e o que falta para continuar; não avance deixando o gate pendente.

## Commit da tarefa

- Antes do commit, confira o diff staged e confirme que ele contém somente a tarefa atual, incluindo testes, contrato, geração e documentação inseparáveis, com verificações e revisão por IA resolvidas.
- Use uma mensagem curta e descritiva; não imponha Conventional Commits.
- Informe o hash e o resumo depois do commit. Não altere commits já criados sem autorização explícita.
- Se hooks ou o próprio commit modificarem arquivos, confira a diferença e repita as verificações e a revisão por IA aplicáveis antes de prosseguir. Se for necessário alterar um commit já criado ou se a mudança sair do escopo da tarefa, solicite uma decisão ao usuário.

## Conclusão do plano

Na última tarefa, execute as verificações conjuntas do plano antes do gate de revisão por IA e antes do commit. Depois do último commit, faça apenas a conferência final de branch e histórico e confirme que implementação, testes, OpenAPI, artefatos gerados, migrations, documentação e `docs/implementation-scope.md` permanecem consistentes. Entregue a lista de commits por tarefa e informe claramente qualquer verificação não executada ou risco remanescente. Não faça merge ou push sem solicitação explícita.
