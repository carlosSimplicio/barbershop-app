---
name: barbershop-plan-implementer
description: Executa um plano de implementação existente do barbershop-app, tarefa por tarefa, em branch dedicada, com TDD orientado a comportamento, revisão independente por IA, aprovação manual e um commit por tarefa. Use quando o usuário pedir para implementar ou continuar um plano; não use para criar o plano nem para mudanças avulsas sem plano aprovado.
---

# Implementar planos do barbershop-app

Implemente somente o escopo do plano indicado, em sequência, preservando a rastreabilidade entre cada subtarefa e seu commit.

## Antes de implementar

1. Leia o `AGENTS.md`, o plano completo e os documentos obrigatórios indicados por ambos.
2. Inspecione o código, testes, OpenAPI, migrations e estado do Git relacionados à primeira tarefa ainda pendente.
3. Não inicie enquanto o plano tiver decisões pendentes, conflitos com a implementação atual ou critérios de verificação insuficientes. Apresente o problema e aguarde decisão do usuário.
4. Garanta que o arquivo do plano esteja versionado antes da primeira tarefa. Se estiver novo ou modificado, solicite autorização para um commit exclusivo do plano; nunca o esconda dentro do commit da primeira tarefa.
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
5. Passe pelos gates de revisão por IA e manual antes de criar o commit.
6. Depois da aprovação manual, crie exatamente um commit focado para a tarefa e somente então avance para a próxima.

Não agrupe várias tarefas do plano em um commit e não divida uma tarefa em commits intermediários. Preserve no working tree apenas o trabalho da tarefa atual. Se uma tarefa se revelar grande demais ou depender de uma mudança fora do plano, pare e proponha a revisão do plano em vez de alterar silenciosamente a granularidade.

## TDD orientado a comportamento

Para toda mudança de comportamento tecnicamente testável:

1. **Red:** escreva primeiro o menor teste unitário significativo que expresse o comportamento, recusa ou invariante do plano. Execute-o e confirme que falha pelo motivo esperado, não por erro de compilação, fixture ou infraestrutura alheia.
2. **Green:** implemente o mínimo necessário para satisfazer o comportamento e execute novamente o teste.
3. **Refactor:** melhore nomes, responsabilidades, duplicações e arquitetura sem alterar o comportamento; mantenha os testes verdes.
4. Execute os testes relacionados e as verificações completas exigidas pelo `AGENTS.md` para as camadas afetadas.

Use `testing` no backend e Vitest no frontend. Teste resultados observáveis e regras de negócio, priorizando fluxo esperado, recusas, autorização, isolamento de tenant, transações, concorrência e invariantes relevantes. Não teste detalhes internos, chamadas de mock sem significado, getters, marcação estática ou comportamento de bibliotecas. Não introduza integração, testcontainers ou E2E sem decisão documentada.

Migrations, artefatos gerados, configuração e documentação podem não possuir um teste unitário significativo isolado. Nesses casos, declare a exceção, use as verificações determinísticas apropriadas e não crie um teste artificial apenas para simular TDD.

## Gate de revisão por IA

Depois que a tarefa estiver implementada e as verificações passarem, mas antes do commit:

- Solicite revisão a um subagente independente. O pedido deve incluir a subtarefa do plano, o diff e os documentos aplicáveis, sem induzir uma conclusão.
- O revisor apenas inspeciona e relata; não edita arquivos nem cria commits.
- Peça que priorize bugs, erros lógicos, regressões, requisitos ausentes, segurança, autorização e isolamento, concorrência, testes insuficientes, legibilidade, arquitetura e performance.
- Analise cada achado com evidência. Corrija os achados pertinentes ao escopo, execute novamente o ciclo e as verificações e solicite nova revisão focada quando a correção for material.
- Não substitua essa etapa por auto-revisão. Se um revisor independente não estiver disponível, pare antes do commit e informe o bloqueio.

## Gate de revisão manual

Quando a revisão por IA estiver resolvida:

1. Apresente ao usuário o resultado da tarefa, arquivos alterados, testes e comandos executados, achados da revisão por IA e respectivas resoluções, riscos remanescentes e o diff preparado para commit.
2. Solicite explicitamente a revisão e aprovação manual do usuário e encerre o turno sem criar o commit.
3. Se houver pedidos de mudança, implemente-os com TDD quando aplicável, repita as verificações e refaça a revisão por IA quando a alteração for material. Solicite nova aprovação manual.

Somente uma aprovação explícita libera o commit daquela tarefa. Silêncio, ausência de objeção ou aprovação de tarefa anterior não contam.

## Commit da tarefa

- Antes do commit, confira o diff staged e confirme que ele contém somente a tarefa aprovada, incluindo testes, contrato, geração e documentação inseparáveis.
- Use uma mensagem curta e descritiva; não imponha Conventional Commits.
- Informe o hash e o resumo depois do commit. Não altere o commit aprovado ou commits de tarefas anteriores sem autorização explícita.
- Se hooks ou o próprio commit modificarem arquivos, pare, mostre a diferença e repita os gates antes de prosseguir.

## Conclusão do plano

Na última tarefa, execute as verificações conjuntas do plano antes dos gates de revisão por IA e manual e antes do commit. Depois do último commit, faça apenas a conferência final de branch e histórico e confirme que implementação, testes, OpenAPI, artefatos gerados, migrations, documentação e `docs/implementation-scope.md` permanecem consistentes. Entregue a lista de commits por tarefa e informe claramente qualquer verificação não executada ou risco remanescente. Não faça merge ou push sem solicitação explícita.
