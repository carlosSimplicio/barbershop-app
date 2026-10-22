# Produto e regras de negócio

## Visão e públicos

O produto atende estabelecimentos de beleza e cuidados pessoais. Seus públicos são:

- Clientes finais, que usam a página pública do estabelecimento sem criar conta.
- Proprietários, que administram o estabelecimento.
- Profissionais, que operam a própria agenda.

O fluxo público permite escolher profissional, serviço, dia e horário. O painel permite administrar agenda, serviços, profissionais, faturamento, despesas e comissões.

## Cadastro, usuários e permissões

- O próprio proprietário realiza um cadastro público e cria sua conta e seu estabelecimento.
- A criação da conta, do estabelecimento e do vínculo como proprietário deve ser atômica.
- Não existe aprovação manual da plataforma, mas o proprietário precisa confirmar seu e-mail.
- O proprietário escolhe o slug público durante o cadastro do estabelecimento.
- O slug não pode ser alterado depois da criação no MVP.
- O cadastro pode ser concluído sem plano ou assinatura ativa, permitindo o início de um período de avaliação.
- O cadastro inicial recebe nome do estabelecimento, slug, fuso IANA, e-mail e senha. Ele cria atomicamente estabelecimento, proprietário, credencial e solicitação de confirmação, mas não inicia o trial.
- E-mail já cadastrado e slug indisponível são apresentados como conflitos específicos dos respectivos campos. Uma indisponibilidade no envio da confirmação depois da persistência não desfaz o cadastro.
- O proprietário pode solicitar uma nova confirmação informando o e-mail. A resposta não revela se a conta existe, já foi confirmada ou recebeu um novo token; quando aplicável, o token anterior é invalidado antes do novo envio.
- A confirmação recebe o token somente no corpo, recusa de forma indistinguível tokens inexistentes, expirados, substituídos ou usados e, quando válida, confirma a conta e autentica automaticamente o proprietário.
- Antes da confirmação, o estabelecimento persistido não possui trial e o proprietário ainda não possui sessão. Depois da confirmação, a sessão autenticada pode consultar somente seu próprio estabelecimento, com identificador, nome, slug, fuso IANA e situação básica do trial; a resposta admite `trial: null` para estabelecimentos sem o benefício e, quando ele existe, informa `active` ou `expired` e seus limites exatos em UTC. A área temporária `/testes` mostra nome, slug, fuso e situação do trial; a ação fica indisponível enquanto o profissional precisa trocar a senha temporária.
- A interface de cadastro detecta o fuso do dispositivo, usa `America/Sao_Paulo` como fallback e permite confirmá-lo ou alterá-lo antes do envio; o backend valida e persiste somente o identificador IANA recebido. Após o cadastro, a interface orienta a conferir o e-mail sem apresentar o trial como iniciado e permite solicitar novo link para o mesmo endereço, com resposta genérica. A página `/confirmar-email` remove o token do fragmento da URL antes da verificação, envia-o somente no corpo, informa links inválidos ou expirados e, quando a API confirma a conta, sincroniza a sessão e segue para `/testes`. `/login` atende proprietários e profissionais, oferece reenvio quando falta confirmação e segue para `/testes` após sincronizar a sessão. Quando a senha temporária exige troca, `/testes` apresenta o formulário de nova senha e só libera a consulta do estabelecimento após a sessão refletir a troca.

## Planos, assinatura e trial

- Planos, cobrança de assinatura do estabelecimento e períodos de avaliação fazem parte do produto.
- Não confunda a assinatura paga pelo estabelecimento com pagamentos de clientes pelos serviços realizados.
- O proprietário pode criar conta e estabelecimento sem contratar imediatamente um plano.
- O trial dura 14 dias e oferece acesso completo às funcionalidades disponíveis.
- O trial começa quando o proprietário confirma o e-mail e ativa a conta; o tempo entre cadastro e confirmação não consome o período.
- Cada proprietário e estabelecimento tem direito a um único trial. Cancelar, expirar, renovar ou recriar a assinatura não reinicia o benefício.
- A confirmação válida cria o trial no intervalo exato de 14 períodos de 24 horas a partir do instante da confirmação; a confirmação, o consumo do token, o trial e a sessão são persistidos atomicamente.
- O MVP oferece um único plano pago com todas as funcionalidades atuais.
- O plano inicial custa R$ 20,00 por mês.
- O catálogo inicial é criado por seed versionado; não existe perfil ou painel de administrador da plataforma no MVP.
- A arquitetura deve permitir futuramente múltiplos planos, diferentes períodos de cobrança, como mensal e semestral, e adicionais cobrados junto à assinatura, como uma ferramenta de IA.
- O gateway real ainda não foi escolhido; a primeira versão usa cobrança mockada, sem movimentar dinheiro.
- No fluxo mockado, o proprietário escolhe “Assinar plano” e a cobrança é aprovada imediatamente, ativando a assinatura mensal sem solicitar dados de pagamento.
- O mock não renova automaticamente. O proprietário precisa escolher “Renovar” a cada período mensal.
- Se a renovação ocorrer durante um período ativo, acrescente um mês ao fim vigente; se ocorrer após expiração, inicie um novo período na reativação.
- O proprietário pode cancelar a assinatura; o acesso permanece ativo até o fim do período já contratado e nenhuma nova renovação ocorre depois.
- O cancelamento não remove dados, não encerra o acesso imediatamente e não gera reembolso automático.
- Quando trial ou assinatura expirar, preserve todos os dados e coloque o painel em modo somente leitura.
- Com acesso expirado, permita autenticação, consulta dos dados e assinatura/renovação, mas bloqueie demais alterações.
- A página pública pode continuar identificando o estabelecimento, mas não oferece nem aceita novos agendamentos até a reativação.
- Preserve agendamentos futuros criados antes da expiração; não os cancele nem exclua automaticamente.
- Enquanto o acesso estiver expirado, esses agendamentos ficam congelados e não podem ser cancelados, reagendados, concluídos ou alterados por cliente, profissional ou proprietário.
- Ao reativar a assinatura, restaure o acesso normal aos agendamentos preservados.
- Agendamentos cuja data passou durante a expiração continuam como `scheduled`; não aplique transição automática.
- Após a reativação, alerte o proprietário sobre esses registros para resolução manual como concluído, `no_show` ou cancelado.
- Proprietários e profissionais entram com e-mail e senha.
- Cada usuário pertence a exatamente um estabelecimento; garanta essa cardinalidade também no banco.
- O proprietário administra apenas seu estabelecimento e pode cadastrar e administrar serviços e profissionais.
- O profissional acessa somente a própria agenda e pode corrigir apenas o nome de clientes relacionados aos seus agendamentos; não pode alterar telefones.
- O proprietário acessa e administra o financeiro completo do estabelecimento.
- O profissional possui uma visão financeira somente leitura restrita aos próprios atendimentos e comissões.
- O profissional não acessa despesas, faturamento geral nem informações financeiras de outros profissionais.
- Toda autorização é aplicada no backend. Ocultar ações na interface não substitui controle de acesso.

## Clientes

- Clientes não criam conta. O agendamento público exige somente nome e telefone.
- Não solicite e-mail, documento, endereço, observações ou outros dados pessoais no MVP.
- O telefone não é verificado por SMS, WhatsApp ou código temporário.
- Dentro de um estabelecimento, o telefone normalizado identifica um cliente e reúne seu histórico.
- Garanta unicidade por `(establishment_id, normalized_phone)`, nunca pelo telefone globalmente.
- O mesmo telefone pode representar clientes independentes em estabelecimentos diferentes.
- Alterar o nome não pode quebrar ou duplicar o histórico.
- Somente o proprietário pode alterar o telefone de um cliente. O profissional pode alterar apenas o nome e somente quando possuir agendamento relacionado ao cliente.
- Ao alterar um telefone, preserve o mesmo cliente e seu histórico. Se o novo telefone normalizado já pertencer a outro cliente do estabelecimento, rejeite a alteração; não mescle históricos automaticamente no MVP.
- Em novo agendamento público, se o telefone já identificar um cliente, vincule o agendamento ao cadastro existente e não sobrescreva seu nome com o valor recebido.
- Não revele no fluxo público se o nome informado diverge do cadastro; correções de nome ocorrem somente no painel autorizado.
- Para reencontrar seus agendamentos, o cliente informa novamente o telefone na página pública do estabelecimento.
- A consulta pública retorna somente agendamentos futuros em estado `scheduled` naquele estabelecimento e apenas os dados necessários para identificar, cancelar ou reagendar o horário.
- Não exponha ao cliente, por telefone, atendimentos concluídos, cancelados, `no_show` ou horários já passados; esse histórico permanece no painel autenticado.
- Proteja o fluxo contra enumeração e abuso sem adicionar confirmação visível; use rate limiting e identificadores não previsíveis.
- Não há fluxo automatizado de exclusão ou anonimização no MVP. Não remova dados pessoais de forma improvisada.

## Localização

- O produto atende inicialmente o Brasil, com interface e mensagens em `pt-BR` e moeda `BRL`.
- Formate moeda apenas na apresentação e represente valores com tipo exato, nunca ponto flutuante binário.
- Cada estabelecimento possui fuso horário IANA configurável. Use `America/Sao_Paulo` como padrão no cadastro e permita que o proprietário selecione outro identificador IANA válido.
- Permita alterar o fuso somente enquanto o estabelecimento não possuir agendamentos, bloqueios ou lançamentos financeiros. Depois do primeiro dado operacional, trate-o como imutável no MVP.
- Interprete jornadas, bloqueios e datas no fuso do estabelecimento.
- Persista instantes concretos de forma inequívoca e converta-os ao exibir ou calcular disponibilidade.
- Normalize telefones brasileiros de maneira consistente, mantendo código do país e DDD quando disponíveis.

## Agenda e disponibilidade

- Cada agendamento possui exatamente um serviço e um profissional.
- O proprietário define explicitamente quais serviços cada profissional oferece; um profissional não oferece automaticamente todos os serviços do estabelecimento.
- Exiba e aceite novos agendamentos somente para pares ativos de profissional e serviço pertencentes ao mesmo estabelecimento.
- A associação determina elegibilidade, mas não redefine preço ou duração do serviço. A eventual comissão específica continua configurada separadamente para o par profissional-serviço.
- Preço e duração pertencem ao serviço e são iguais para todos os profissionais associados; não crie sobrescritas por profissional no MVP.
- Cada profissional possui jornada semanal recorrente própria, editável por ele ou pelo proprietário.
- Permita vários períodos no mesmo dia, como `08:00–12:00` e `13:00–18:00`, sem sobreposição entre períodos.
- Proprietário e profissional podem criar bloqueios pontuais para compromissos, folgas ou outras indisponibilidades.
- Cada serviço possui um preço padrão obrigatório, armazenado com precisão monetária; aceite zero para serviços gratuitos e rejeite valores negativos.
- O proprietário define a duração do serviço; quando vazia, use 30 minutos.
- Ofereça inícios em intervalos de 15 minutos. Não existe buffer antes ou depois do serviço no MVP.
- A duração integral do serviço define o fim do atendimento; o slot define apenas possíveis horários de início.
- Calcule disponibilidade com jornada, duração, agendamentos e bloqueios.
- Um intervalo que se sobreponha total ou parcialmente a agendamento ou bloqueio não está disponível.
- Permita que o mesmo cliente mantenha agendamentos `scheduled` sobrepostos, inclusive com profissionais diferentes. Não trate telefone ou cliente como recurso exclusivo de agenda.
- A sobreposição permitida não elimina proteção contra submissões duplicadas: operações de criação devem ser idempotentes para não gerar cópias acidentais da mesma solicitação.
- Valide novamente no backend ao criar ou reagendar e impeça condições de corrida e dupla ocupação do profissional.
- Copie para o agendamento o preço e a duração do serviço no momento da criação.
- Alterar preço ou duração no catálogo afeta somente novos agendamentos e não modifica valores nem intervalos já registrados.
- Cancelar ou reagendar preserva os snapshots originais de preço e duração do agendamento.

### Limites configuráveis

O proprietário pode configurar por estabelecimento:

- Antecedência mínima para o cliente agendar.
- Quantos dias no futuro o cliente pode agendar.
- Prazo mínimo para o cliente cancelar ou reagendar.

As três configurações são opcionais; valor ausente/`NULL` significa sem restrição. Não use zero quando ele for ambíguo. Elas se aplicam apenas ao cliente. Proprietários e profissionais podem ajustar a agenda a qualquer momento, respeitando jornada, bloqueios e conflitos.

## Estado e alteração de agendamentos

- Estados válidos: `scheduled`, `completed`, `canceled` e `no_show`, apresentados em português na interface.
- Não crie outros estados nem transições automáticas sem decisão de produto.
- Proprietário, profissional e cliente podem cancelar ou reagendar dentro do próprio escopo.
- O cancelamento feito pelo cliente não solicita motivo. Cancelamentos feitos por proprietário ou profissional exigem justificativa interna e registram autor e data.
- O cliente só pode cancelar ou reagendar um agendamento `scheduled` antes de seu horário de início, considerando o fuso do estabelecimento.
- Os limites configuráveis podem restringir ainda mais essa janela, mas nunca permitir ao cliente alterar um agendamento já iniciado ou passado.
- O profissional pode marcar seus próprios agendamentos como `completed` ou `no_show`.
- O proprietário pode marcar como `completed` ou `no_show` qualquer agendamento do estabelecimento.
- Depois que um agendamento entrar em `completed`, `canceled` ou `no_show`, clientes e profissionais não podem reverter nem trocar seu estado.
- Somente o proprietário pode corrigir um estado final. Exija justificativa e preserve autor, data, estado anterior e novo no histórico de auditoria.
- Uma correção para `completed` aplica as mesmas exigências de pagamento e gera receita e comissão na mesma transação. Uma correção que retire `completed` reverte esses efeitos como correção interna, sem representar reembolso.
- Só permita corrigir um estado final de volta para `scheduled` se o horário ainda for futuro e a disponibilidade for revalidada sem conflito.
- Ao concluir um atendimento com valor maior que zero, exija uma única forma de pagamento e gere receita e comissão de forma atômica usando o valor registrado no agendamento.
- Quando o valor final for zero, não exija forma de pagamento e registre receita e comissão como zero.
- O profissional pode concluir o próprio atendimento e informar a forma de pagamento, mas não pode alterar o valor cobrado.
- Somente o proprietário pode ajustar o valor do atendimento, antes ou depois da conclusão.
- Marcar `no_show` não gera receita nem comissão.
- Depois da conclusão, somente o proprietário pode corrigir valor cobrado ou forma de pagamento.
- Preserve um histórico auditável com autor, data, valores anteriores e valores novos; não sobrescreva silenciosamente o registro original.
- Exija uma justificativa textual para toda correção pós-conclusão e armazene-a no histórico de auditoria.
- Ao corrigir o valor, recalcule a comissão usando o tipo e a regra de comissão registrados na conclusão, não a configuração vigente no momento da correção.
- Reagendamento repete todas as validações da criação.
- Reagendamento altera o mesmo agendamento de forma atômica e preserva no histórico autor, data, horário anterior e novo.
- Se a validação ou reserva do novo horário falhar, mantenha o horário original sem qualquer alteração.
- Cancelamento preserva o registro; não faça exclusão física.
- O cliente não passa por login, OTP, SMS ou WhatsApp para cancelar ou reagendar no MVP.

## Serviços e profissionais inativos

- Quando houver histórico, desative profissionais e serviços em vez de excluí-los.
- Ao desativar um profissional, encerre imediatamente suas sessões e impeça novos logins.
- Itens inativos não aparecem em novos agendamentos públicos, mas permanecem em consultas históricas.
- Ao remover uma associação entre profissional e serviço, impeça somente novos agendamentos para esse par e preserve agendamentos existentes.
- Preserve referências e snapshots necessários para interpretar agenda e financeiro antigos.
- Antes de desativar, informe quantos agendamentos futuros serão afetados.
- Agendamentos futuros continuam válidos; não os cancele automaticamente.
- Exiba um alerta interno ao proprietário para tratamento manual, sem notificação externa.
- Não use exclusão em cascata que remova agendamentos, atendimentos, comissões ou lançamentos.

## Financeiro

- Use regime de caixa no MVP.
- Somente o proprietário pode administrar lançamentos e consultar a visão financeira geral.
- A visão do profissional exibe apenas seus atendimentos, valores e comissões e não permite alterar lançamentos financeiros.
- O fechamento do próprio atendimento pelo profissional não é edição financeira direta: ele registra a forma de pagamento e gera receita e comissão como parte do caso de uso de conclusão, sem permissão para alterar o valor.
- Faturamento considera apenas atendimentos concluídos. Agendamentos marcados, cancelados e `no_show` não entram.
- Reconheça a receita quando o atendimento for concluído e seu pagamento registrado.
- Permita registrar atendimento sem agendamento para encaixe ou chegada espontânea.
- Esse registro contém exatamente um serviço e, no mínimo, profissional, valor e data; entra como concluído. Exija forma de pagamento somente quando o valor for maior que zero.
- Não exija nome nem telefone do cliente em atendimentos sem agendamento; permita registrá-los sem cliente identificado.
- O proprietário pode registrar esse atendimento para qualquer profissional do estabelecimento; o profissional pode registrá-lo somente para si próprio.
- O atendimento sem agendamento exige uma associação ativa entre o profissional e o serviço; o proprietário deve criá-la antes do lançamento quando ela não existir.
- Quando o profissional registrar o próprio atendimento, use o preço atual do serviço e não permita que ele altere o valor. Ajustes de valor continuam exclusivos do proprietário.
- O proprietário pode informar a data e hora atual ou passada do atendimento; o profissional só pode registrar atendimentos ocorridos no dia atual do estabelecimento.
- Rejeite datas futuras. Persista separadamente quando o atendimento ocorreu e quando e por quem o lançamento foi criado.
- Somente o proprietário pode anular um atendimento concluído ou sem agendamento registrado por engano.
- A anulação exige justificativa e preserva autor, data e dados originais para auditoria; não exclua nem sobrescreva o registro.
- Receita e comissão anuladas deixam de compor os indicadores financeiros.
- A anulação financeira não adiciona outro estado ao agendamento: mantenha os quatro estados definidos e represente a reversão com metadados próprios do atendimento ou lançamento.
- Trate a anulação somente como correção interna de lançamento. Ela não processa nem registra reembolso, estorno ou devolução de dinheiro ao cliente no MVP.
- Registre despesas separadamente das receitas.
- Estados de despesa: `planned`, `paid` e `canceled`, traduzidos na interface.
- Permita cadastrar uma despesa prevista com data atual ou futura. Despesas `planned` não compõem os totais financeiros.
- Para uma despesa já quitada, permita a criação direta como `paid`, desde que a data real do pagamento seja atual ou passada.
- Uma despesa com data futura deve ser criada como `planned` e nunca como `paid`.
- A chegada da data prevista não altera o estado automaticamente. Somente o proprietário pode marcar a despesa como `paid` e informar a data real do pagamento.
- Destaque no painel como atrasada toda despesa ainda `planned` cuja data prevista já passou no fuso do estabelecimento. Esse alerta é calculado e não cria outro estado.
- Não envie notificação externa sobre o atraso no MVP.
- A data real de pagamento não pode estar no futuro. Reconheça a despesa somente nessa data e apenas enquanto estiver `paid`.
- Somente o proprietário pode criar ou editar despesas.
- Ao editar uma despesa, exija justificativa e preserve autor, data, valores anteriores e novos no histórico de auditoria.
- Não sobrescreva silenciosamente lançamentos de despesas já contabilizados.
- O proprietário pode cancelar uma despesa lançada por engano, mas não excluí-la fisicamente.
- Cancelamento exige justificativa e registra autor e data; a despesa cancelada permanece no histórico e deixa de compor os totais financeiros.
- Cada despesa é lançada individualmente no MVP; não gere recorrências automaticamente.
- Ofereça categorias padrão de despesas definidas pelo sistema.
- Categorias padrão iniciais: `rent` (aluguel), `water` (água), `electricity` (energia), `internet_phone` (internet/telefone), `supplies` (materiais e insumos), `maintenance` (manutenção), `marketing`, `taxes_fees` (impostos e taxas) e `other` (outros).
- Permita que o proprietário crie categorias personalizadas restritas ao próprio estabelecimento.
- Categorias padrão não podem ser renomeadas ou excluídas pelo estabelecimento.
- Categorias personalizadas já usadas em lançamentos devem ser desativadas, não excluídas, preservando o histórico.
- Categorias inativas não aparecem em novos lançamentos, mas continuam visíveis nos registros anteriores.
- Cada despesa exige descrição, categoria, valor e data prevista. A data de pagamento é obrigatória somente ao marcá-la como `paid`; não registre forma de pagamento para despesas no MVP.
- Formas de recebimento dos atendimentos: `pix`, `card` e `cash`, traduzidas na interface.
- Cada atendimento com valor maior que zero possui exatamente uma forma de pagamento; não divida um recebimento entre múltiplas formas no MVP. Atendimentos gratuitos não possuem forma de pagamento.
- A forma de pagamento é informada somente após o atendimento.
- O sistema apenas registra a forma usada: não processa nem confirma pagamentos e não integra gateways.
- Nunca colete número, código de segurança ou outros dados de cartão.
- O proprietário define comissão padrão por profissional, percentual sobre o valor cobrado ou valor fixo.
- Pode existir uma comissão específica por profissional e serviço; ela prevalece sobre a padrão.
- No MVP, calcule e apresente as comissões geradas, mas não controle pagamento, liquidação, fechamento de período ou saldo devedor ao profissional.
- Comissão percentual deve ficar entre 0% e 100%; comissão fixa não pode ser negativa nem superar o valor cobrado pelo atendimento.
- Calcule a comissão percentual sobre o valor efetivamente cobrado e arredonde para o centavo mais próximo, usando metade para cima em caso de empate.
- Preserve valor, preço e regra de comissão aplicados na conclusão. Mudanças futuras não alteram o histórico.

### Indicadores financeiros

- Exiba faturamento bruto: soma das receitas válidas de atendimentos concluídos no período.
- Exiba total de comissões: soma das comissões válidas geradas no período.
- Exiba despesas: soma das despesas pagas e não canceladas no período.
- Exiba resultado líquido: faturamento bruto menos comissões e despesas.
- Exiba quantidade de atendimentos concluídos.
- Exiba ticket médio: faturamento bruto dividido pela quantidade de atendimentos concluídos; quando não houver atendimentos, apresente zero sem dividir por zero.
- Permita filtrar por período, profissional, serviço e forma de pagamento, inclusive combinando filtros.
- Aplique limites de período conforme o fuso horário do estabelecimento e use as mesmas regras de cálculo em cards, tabelas e exportações futuras.
- Na visão do profissional, restrinja os indicadores aos próprios atendimentos e comissões e não exponha despesas nem o resultado geral.

## Experiência de uso

- A home deslogada da plataforma apresenta sua proposta de valor em português e oferece caminhos para cadastro e login. O formulário de cadastro, o estado de confirmação pendente e o login estão disponíveis. Após autenticar, o usuário chega à área temporária `/testes`, que mostra apenas dados não sensíveis da sessão e permite consultar novamente a sessão e o estabelecimento, trocar a senha temporária quando exigido e encerrar a sessão.
- A página pública sempre exibe nome e logo do estabelecimento; ambos são obrigatórios.
- Descrição curta, endereço, telefone de contato e links de redes sociais são opcionais e aparecem somente quando preenchidos.
- Não renderize seções vazias ou placeholders para informações públicas opcionais ausentes.
- Use o nome do estabelecimento como base para o texto alternativo do logo quando não houver descrição mais adequada.
- Página pública e agenda operacional são mobile-first.
- Demais áreas administrativas e financeiras priorizam desktop, mas permanecem responsivas.
- Evite versões duplicadas quando um componente responsivo atender aos dois contextos.
- Componentes próprios devem preservar alvos de toque adequados, navegação por teclado, foco visível, contraste e rótulos acessíveis.

## Fora do MVP

- Notificações ou confirmações de agenda por WhatsApp e SMS.
- Processamento de pagamentos dos clientes finais pelos serviços e integrações do agendamento com adquirentes ou gateways. Essa restrição não se aplica à futura cobrança da assinatura da plataforma.
- Emissão de nota fiscal e integrações fiscais.
- Exclusão ou anonimização automatizada de dados de clientes.
- Despesas recorrentes e geração automática de lançamentos.
- Serviços externos de comunicação além do e-mail transacional necessário à autenticação do proprietário.
