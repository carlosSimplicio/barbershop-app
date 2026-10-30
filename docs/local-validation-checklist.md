# Checklist de validação local

Roteiro manual das funcionalidades já entregues. Execute os fluxos na ordem indicada: os cadastros e dados de uma etapa são reaproveitados nas próximas. Marque um item somente após observar o resultado esperado; os checkboxes começam vazios e não representam verificações já executadas.

As referências são [escopo implementado](implementation-scope.md), [regras do produto](product.md), [arquitetura](architecture.md), [operação](operations.md) e [OpenAPI](../backend/openapi/openapi.yaml). Atualize este roteiro quando uma nova funcionalidade for entregue.

Agenda, disponibilidade, clientes, associação profissional–serviço, comissões, atendimentos, financeiro e contratação/renovação de assinatura ainda não estão entregues. Os limites de agenda já podem ser configurados, mas sua aplicação depende desses fluxos futuros. A página pública atual é informativa.

## Como executar e preparar os dados

Use somente dados fictícios. Escolha um sufixo exclusivo por rodada, como `rodada-01`, para evitar colisões com execuções anteriores. Nunca registre senhas, cookies, tokens ou links completos de e-mail nas evidências.

| Identificação no roteiro           | Dados a preparar                                                      | Uso                                                |
| ---------------------------------- | --------------------------------------------------------------------- | -------------------------------------------------- |
| Proprietário A / estabelecimento A | E-mail `owner-a-rodada-01@example.test`, slug `validacao-a-rodada-01` | Fluxo principal e expiração ao final               |
| Profissional A                     | E-mail `professional-a-rodada-01@example.test`                        | Senha temporária, edição, desativação e reativação |
| Proprietário B / estabelecimento B | E-mail `owner-b-rodada-01@example.test`, slug `validacao-b-rodada-01` | Isolamento entre estabelecimentos                  |
| Proprietário pendente              | Outro e-mail e slug exclusivos; deixar sem confirmar                  | Respostas genéricas e confirmação obrigatória      |

Mantenha o proprietário A no navegador principal, o profissional em outro navegador/perfil e o proprietário B em um terceiro perfil quando necessário. Abas do mesmo perfil compartilham cookies e não representam sessões independentes. Uma janela anônima sem login serve para conferir a página pública.

Deixe as ferramentas de desenvolvimento abertas nas abas Rede e Aplicação. Para testes técnicos, use um cliente HTTP com cookie jar local e as mesmas regras de origem e CSRF do navegador. Todos os endpoints citados abaixo têm prefixo `/api/v1`.

As proteções de abuso continuam ativas durante o roteiro. Cadastro, reenvio e recuperação também contabilizam solicitações aceitas; tentativas de contas diferentes compartilham a dimensão de IP. Se receber `429`, aguarde `Retry-After` e retome o passo, sem limpar tabelas. No login, sucessos não apagam as falhas anteriores. Separe rodadas quando necessário para não confundir limitação com defeito funcional.

## Fluxo 1 — Subir o ambiente e abrir a plataforma

- [ ] Conferir Docker com Compose e `make` disponíveis. Para comandos executados no host, conferir as versões de Go e Node fixadas no [README](../README.md#stack-fixada).
- [ ] Se `.env` ainda não existir, copiar `.env.example` para `.env`, executar `make auth-keys` e preencher as cinco variáveis geradas. Preservar a configuração de um ambiente já existente e manter os segredos fora de arquivos versionados.
- [ ] Executar `make compose-config`: a composição deve ser válida.
- [ ] Executar `make dev` e aguardar PostgreSQL, backend, frontend, Caddy e Mailpit. A tarefa `logo-storage-init` deve terminar com sucesso; os serviços permanentes devem ficar saudáveis.
- [ ] Em outro terminal, executar `make migrate-up`: todas as migrations devem ser aplicadas sem apagar dados. Se o ambiente já estava ligado com código antigo, reconstruir o backend conforme os comandos abaixo.
- [ ] Confiar na CA local do Caddy seguindo o [README](../README.md#início-rápido), reiniciar o navegador e abrir `https://localhost:8443` sem erro de certificado.
- [ ] Abrir `http://localhost:8088`: deve redirecionar para HTTPS. Abrir `http://127.0.0.1:8025`: o Mailpit deve estar disponível.
- [ ] Consultar `GET /health`: esperar `200` com `{"status":"ok"}`. Conferir que `/api/v1` chega ao Go e as páginas chegam ao Next.js.
- [ ] Abrir `/` sem sessão: conferir textos em português, layout em tela estreita e links para cadastro e login. A home não deve disparar consulta de sessão ou outra chamada à API.
- [ ] Abrir diretamente `/painel/profissionais`, `/painel/servicos`, `/painel/configuracoes` e `/testes` sem sessão: o conteúdo protegido não deve aparecer e o destino deve ser `/login`.

Atualização de ambiente existente, com `make dev` ativo:

```sh
make migrate-up
docker compose -f compose.yaml -f compose.dev.yaml up -d --build --no-deps backend
```

## Fluxo 2 — Cadastrar o proprietário A e confirmar o e-mail

- [ ] Da home, abrir `/cadastro`. Conferir nome, slug com prefixo visual `/`, e-mail, senha e quatro fusos: Brasília, Amazônia, Acre e Fernando de Noronha.
- [ ] Com a emulação de fuso do navegador, conferir seleção de um fuso brasileiro suportado e fallback para `America/Sao_Paulo` quando o dispositivo estiver em outro fuso. Escolher o fuso desejado antes do envio.
- [ ] Tentar nome vazio, e-mail inválido e slug inválido: menos de 3 ou mais de 63 caracteres, maiúsculas, espaços ou hífen nas extremidades. Conferir erros junto aos campos e ausência de cadastro parcial.
- [ ] Tentar um slug reservado, como `painel`: deve ser recusado sem impedir que a rota da plataforma continue acessível.
- [ ] Validar a política de senha: aceitar de 8 a 32 pontos de código e recusar valores menores/maiores, espaços e caracteres invisíveis como `U+200B` e `U+FEFF`, sem aparar ou normalizar silenciosamente a entrada. Não exigir combinação obrigatória de maiúsculas, números e símbolos.
- [ ] Cadastrar A com dados válidos e exclusivos: esperar `202`, orientação para conferir o e-mail e possibilidade de pedir novo link. O cadastro não deve iniciar sessão nem apresentar o trial como iniciado.
- [ ] Antes de confirmar, tentar login com a senha correta: deve exigir confirmação e oferecer reenvio. Usar essa opção e conferir a nova mensagem no Mailpit.
- [ ] Abrir o link anterior: deve ser recusado como inválido/expirado. Abrir o mais recente: o token deve desaparecer do endereço antes das chamadas de API e seguir somente no corpo de `POST /auth/email-confirmation/verify`.
- [ ] Conferir confirmação `200`, sessão iniciada e entrada por `/painel`, seguida de `/painel/configuracoes`, pois ainda não existe logo.
- [ ] Abrir `/painel/profissionais` diretamente mesmo sem logo: o proprietário deve conseguir acessar a lista vazia e o cadastro.
- [ ] Pela navegação, abrir `/testes` e consultar sessão e estabelecimento: conferir papel `owner`, nome, slug, fuso e trial ativo. A janela UTC deve durar exatamente 14 × 24 horas a partir da confirmação.
- [ ] Tentar reutilizar o link consumido em perfil sem sessão: deve ser recusado, sem nova sessão e sem reiniciar o trial.
- [ ] Pela API, solicitar confirmação para e-mail ausente e para A já confirmado: retornar o mesmo `202` sem corpo, sem nova confirmação ou novo trial. Respeitar o limite de tentativas do escopo.
- [ ] Tentar outro cadastro com o e-mail de A, inclusive com variação de caixa, e depois com o slug de A: conferir conflito específico no campo correspondente e ausência de um segundo estabelecimento parcial.
- [ ] Criar a conta pendente da tabela e mantê-la sem confirmar para os testes posteriores de recuperação e respostas genéricas.

## Fluxo 3 — Configurar A e publicar sua página

- [ ] Voltar a `/painel/configuracoes`: conferir os dados persistidos, o slug somente leitura e a indicação de que falta logo para publicar.
- [ ] Alterar nome e fuso enquanto `time_zone_editable` for verdadeiro; salvar e recarregar. Conferir persistência. Criar serviços ou profissionais não deve, por si só, bloquear o fuso: o bloqueio está ligado a dados operacionais futuros.
- [ ] Preencher descrição e endereço fictícios, telefone fixo com DDD e salvar. Recarregar: conferir textos e telefone com máscara nacional, embora a API normalize com código do país.
- [ ] Trocar para celular com DDD: conferir a máscara de 11 dígitos. Tentar telefone incompleto e entrada além do tamanho permitido; não deve persistir valor inválido.
- [ ] Conferir limites de nome de 120 pontos de código e descrição/endereço de 500. Testar excesso também pela API para garantir validação do backend.
- [ ] Adicionar redes usando as seis opções disponíveis: Instagram, Facebook, X, YouTube, TikTok e LinkedIn. Conferir prefixo HTTPS fixo, exigência de sufixo, botões de adicionar/remover e máximo de cinco links simultâneos.
- [ ] Salvar links em uma ordem conhecida, recarregar e conferir destinos e ordem. Pela API, tentar URL HTTP, URL com credenciais e mais de cinco links: recusar sem salvar parcialmente outros campos enviados no mesmo PATCH.
- [ ] Pela API, preparar um link HTTPS válido incompatível com o prefixo do editor, como uma página empresarial do LinkedIn. Abrir as configurações e alterar somente outro campo: o link legado deve ser preservado integralmente até substituição ou remoção explícita.
- [ ] Preencher antecedência mínima e prazo de alteração com `0` minutos e horizonte com `30` dias: zero deve ser preservado nos campos de minutos. Horizonte zero, negativos e frações devem ser recusados.
- [ ] Limpar os três limites e salvar: a API deve retornar ausência de restrição (`null`), sem substituir por zero. Recarregar e conferir os campos vazios.
- [ ] Salvar apenas um campo e inspecionar o PATCH: somente campos alterados devem ser enviados; os demais valores devem permanecer intactos.
- [ ] Antes do primeiro upload, abrir `/{slug-de-A}` sem sessão: deve mostrar página não encontrada. `GET /public/establishments/{slug-de-A}` também deve retornar `404`.
- [ ] Selecionar PNG, JPEG ou WebP válido: a pré-visualização deve ocupar o lugar do logo. Limpar a seleção antes de enviar: voltar a “Nenhum arquivo selecionado”, sem publicar ou remover imagem.
- [ ] Selecionar novamente e enviar: esperar `204`, limpeza do seletor e da pré-visualização temporária, consulta atualizada e opção “Abrir página pública”. O upload deve usar multipart com CSRF e boundary definido pelo navegador.
- [ ] Abrir a página pública sem sessão: conferir nome, logo, descrição, endereço, telefone e links preenchidos, sem seções vazias ou controles de agendamento.
- [ ] Inspecionar o HTML recebido: nome e conteúdo público devem vir renderizados pelo servidor, com título e URL canônica corretos. O logo deve ter texto alternativo identificando o estabelecimento.
- [ ] Conferir que a API pública expõe somente os campos informativos preenchidos, sem usuários, e-mail do proprietário, trial, limites de agenda, chave de storage ou caminho interno do logo.
- [ ] Limpar descrição, endereço, telefone e redes no painel; salvar e reabrir a página pública. As seções correspondentes devem desaparecer. Preencher novamente os dados que serão usados nas próximas etapas.
- [ ] Usar texto simples contendo marcação, como `<strong>Teste</strong>`, na descrição: a página deve mostrar texto, sem interpretar HTML.
- [ ] Acessar `/painel` com logo já publicado: deve encaminhar a `/painel/profissionais`. Repetir após novo login.

## Fluxo 4 — Substituir o logo e validar recusas e cache

Pré-condição: A publicável e duas imagens válidas de conteúdos diferentes. Separar também arquivo SVG, arquivo truncado, imagem acima de 2 MiB e imagem com dimensão acima de 4.096 pixels.

- [ ] Consultar `GET /public/establishments/{slug-de-A}/logo` sem sessão: conferir bytes da imagem, MIME derivado do conteúdo, `Content-Length`, `X-Content-Type-Options: nosniff`, `Cache-Control: public, no-cache` e `ETag`.
- [ ] Repetir com `If-None-Match` igual ao ETag recebido: esperar `304` sem corpo.
- [ ] Selecionar uma segunda imagem e cancelar a seleção: manter o logo publicado. Selecionar novamente e enviar: conferir substituição no painel e na página pública, usando o mesmo endereço de imagem.
- [ ] Repetir a consulta com o ETag antigo: esperar `200`, novos bytes e novo ETag.
- [ ] Tentar SVG, arquivo truncado ou conteúdo incompatível com um formato aceito: recusar, preservar o logo publicado e limpar seleção/pré-visualização após recusa definitiva.
- [ ] Tentar imagem acima de 2.097.152 bytes e outra acima de 4.096 × 4.096 pixels: recusar. Se o frontend impedir o envio, repetir pela API; excesso de tamanho deve retornar `413`.
- [ ] Conferir pela API que extensão e `Content-Type` informados pelo cliente não bastam para aceitar conteúdo inválido; o backend deve inspecionar os bytes.
- [ ] Conferir que não existe ação de remoção do logo publicado e que a resposta não revela nome interno ou caminho de arquivo.

## Fluxo 5 — Cadastrar profissionais e editar o perfil

Pré-condição: proprietário A autenticado com trial ativo. Mantenha-o neste perfil e use outro navegador/perfil para o fluxo do profissional.

- [ ] Abrir `/painel/profissionais`: conferir lista vazia inicialmente, rótulos, dicas dos campos e controles de escrita liberados somente após confirmação de `can_write`.
- [ ] Tentar nome vazio, nome acima de 120 pontos de código, e-mail inválido e senha temporária fora da política: conferir erros sem criar cadastro parcial.
- [ ] Cadastrar profissional A com nome contendo espaços nas extremidades, e-mail exclusivo e senha temporária válida: esperar `201`, nome normalizado na lista, estado ativo e senha limpa do formulário.
- [ ] Conferir que a resposta contém apenas ID, nome, e-mail e estado ativo, sem senha, hash ou sessão. A senha temporária não deve ser recuperável pela interface.
- [ ] Tentar cadastrar novamente com o mesmo e-mail, inclusive mudando a caixa: conferir conflito no e-mail, sem duplicação.
- [ ] Abrir “Editar”: carregar o registro persistido e mostrar e-mail somente leitura. Salvar sem alterações não deve enviar PATCH.
- [ ] Alterar o nome e cancelar: não gravar e devolver foco à ação de origem ou à lista. Reabrir e salvar outro nome: lista e editor devem mostrar o registro confirmado, preservando ID, e-mail, estado e credencial.
- [ ] Durante cadastro ou edição em andamento, tentar outro envio ou mudança de página: os controles conflitantes devem permanecer bloqueados e não gerar mutações simultâneas.
- [ ] Conferir paginação sem itens duplicados ou de outro estabelecimento. Para poucas entradas, usar a API com `limit=1&offset=0` e `offset=1`; para testar os controles do painel, preparar mais de 20 profissionais fictícios com e-mails exclusivos.

## Fluxo 6 — Primeiro acesso e troca de senha do profissional

- [ ] No perfil separado, entrar em `/login` com o profissional A e sua senha temporária: abrir `/testes`, exibir a exigência de troca e manter a consulta do estabelecimento indisponível.
- [ ] Antes da troca, consultar `GET /auth/me`: retornar a identidade com `password_change_required: true`. Tentar `GET /establishment`: recusar o acesso ao domínio.
- [ ] Informar senha inválida e depois a mesma temporária: recusar, com erro no campo e sem remover a exigência de troca.
- [ ] Antes de concluir a troca, abrir uma segunda sessão independente do profissional, para conferir sua revogação.
- [ ] Definir senha válida diferente da temporária: esperar `204`, atualização de `/auth/me`, desaparecimento do formulário e liberação da consulta do estabelecimento de A.
- [ ] Na segunda sessão anterior, consultar `/auth/me`: esperar `401`. A sessão usada na troca deve continuar válida.
- [ ] Sair e tentar login com a temporária anterior: recusar. Entrar com a nova senha: aceitar sem exigir outra troca.
- [ ] Tentar abrir as páginas de profissionais, serviços e configurações: o profissional deve voltar a `/testes`; chamadas diretas à API desses recursos administrativos devem ser recusadas mesmo após a troca.

## Fluxo 7 — Desativar e reativar o profissional

Pré-condição: sessão do profissional A aberta em outro perfil e proprietário A em `/painel/profissionais`.

- [ ] Abrir “Desativar”: conferir identificação e aviso de perda de acesso. Cancelar: manter estado e sessão do profissional.
- [ ] Reabrir e confirmar: lista e editor devem mostrar inativo, preservando nome, e-mail e ID. Se houver rascunho de nome no editor, ele deve ser preservado sem ser publicado na lista.
- [ ] Consultar a sessão anterior do profissional: esperar `401`; novo login com a senha anterior deve ser recusado.
- [ ] Consultar e editar o nome do profissional inativo: deve continuar possível com escrita ativa. Pela API, repetir a desativação: retornar o estado inativo com segurança, sem excluir o registro.
- [ ] Abrir “Reativar”, preencher senha temporária e cancelar. Reabrir: a entrada deve estar vazia.
- [ ] Reativar com senha temporária válida: esperar `200`, lista/editor ativos e campo de senha limpo. Sessões antigas continuam inválidas.
- [ ] Entrar como profissional: exigir nova troca obrigatória; recusar senha definitiva igual à temporária e aceitar outra válida.
- [ ] Repetir o ciclo de desativação e reativação usando, como temporária, a senha vigente antes da desativação: a reativação deve aceitar a repetição, mas o primeiro acesso continua exigindo uma senha diferente dela.
- [ ] Pela API, tentar reativar um profissional já ativo: esperar `409 professional_already_active`, sem trocar sua senha ou revogar sua sessão válida.
- [ ] Com duas sessões do proprietário, abrir reativação enquanto inativo; concluir em uma e enviar na outra. A segunda deve informar o conflito e exigir consulta do estado, sem anunciar que sua senha foi aplicada.

## Fluxo 8 — Criar, editar, desativar e reativar serviços

- [ ] Como proprietário A, abrir `/painel/servicos`: conferir estado vazio ou lista com nome, preço BRL, duração e estado. Conferir limites visíveis nos formulários.
- [ ] Cadastrar serviço gratuito com duração vazia: esperar `201`, estado ativo, preço `R$ 0,00` e duração de 30 minutos.
- [ ] Cadastrar outro serviço com preço `35,90` e duração `17`: conferir `price_cents: 3590`, duração preservada e apresentação em BRL. Não exigir duração múltipla de 15.
- [ ] Cadastrar nome repetido: permitir. Conferir normalização de espaços nas extremidades e limite de 120 pontos de código sem nome vazio ou caractere NUL.
- [ ] Digitar preços `0,01` e `99999,99`, sem separador de milhar: aceitar exatamente e apresentar o maior como `R$ 99.999,99` na lista. Tentar negativo, excesso de duas casas e valor acima desse teto: recusar sem arredondar silenciosamente.
- [ ] Testar duração `1` e `1440`: aceitar. Tentar `0`, `1441`, negativa e fracionária: recusar. Pela API, omissão ou `null` no cadastro devem aplicar 30 minutos.
- [ ] Abrir “Editar”, alterar somente preço e salvar: o PATCH deve preservar nome, duração, ID e estado. Reabrir, limpar duração e salvar: aplicar 30 minutos.
- [ ] Alterar campos e cancelar: descartar sem gravar e devolver foco à origem ou à lista.
- [ ] Abrir “Desativar” e cancelar: manter estado. Confirmar depois: preservar o serviço como inativo na lista e no editor; permitir consulta e edição do inativo.
- [ ] Reativar: conferir estado ativo na lista/editor. Pela API, repetir `PUT /services/{service_id}/status` com o mesmo `active`: deve ser idempotente e preservar os demais campos.
- [ ] Conferir paginação pela API com `limit=1` e offsets sucessivos; para validar os controles do painel, preparar mais de 20 serviços fictícios. Ativos e inativos devem continuar listados.
- [ ] Durante operação pendente, conferir bloqueio de outros envios, paginação e atualização da lista. Nenhuma ação deve mostrar contagem fictícia de agendamentos afetados.

## Fluxo 9 — Sessões, logout e recuperação do proprietário

Use duas sessões independentes do proprietário A. Evite sequências de login inválido até concluir este fluxo; rate limiting será validado ao final.

- [ ] Fazer login com A usando variação de caixa no e-mail: deve encontrar a mesma conta. Login com senha errada e conta ausente devem mostrar a mesma recusa genérica.
- [ ] Inspecionar `Set-Cookie`: access `__Secure-barbershop_access` com `Path=/api/v1` e `HttpOnly`; CSRF `__Secure-barbershop_csrf` com `Path=/` e legível para envio no header. Ambos host-only, `Secure`, `SameSite=Strict` e `Max-Age=1296000` (15 dias).
- [ ] Em `/testes`, consultar novamente a sessão e o estabelecimento: não expor senha, token ou e-mail na identidade de diagnóstico. Uso e consultas não devem renovar o prazo da sessão nem chamar endpoint de refresh.
- [ ] Sair em uma sessão: esperar `204`, cookies limpos e redirecionamento a `/login`. A outra sessão independente deve continuar válida.
- [ ] Pelo cliente HTTP, repetir logout com origem e CSRF válidos, mas access ausente ou inválido: continuar retornando `204` e limpando cookies, sem revogar outra sessão não identificada.
- [ ] Voltar a uma rota protegida ou usar o histórico após logout: não recuperar o conteúdo protegido nem uma sessão revogada.
- [ ] Em `/login`, abrir “Esqueci minha senha” e enviar o e-mail confirmado de A: esperar `202` e resposta genérica, encerrando o formulário de solicitação.
- [ ] Em acessos deslogados, repetir com conta ausente, proprietário pendente e profissional: mesma resposta `202`, sem revelar elegibilidade; esses casos não devem receber e-mail de recuperação.
- [ ] Solicitar outro link para A; abrir o anterior: recusar. Abrir o mais recente no Mailpit: `/redefinir-senha` deve remover fragmento e query antes de montar o formulário ou fazer chamadas.
- [ ] Tentar senha inválida e senha igual à vigente: recusar com erro de campo. Definir outra válida: esperar `204`, indicação de sucesso e link de retorno ao login, sem autenticação automática.
- [ ] Na sessão anterior de A mantida no outro perfil, consultar recurso autenticado: esperar `401` e retirada do conteúdo protegido.
- [ ] Entrar com a senha anterior: recusar. Entrar com a nova: aceitar e encaminhar por `/painel` de acordo com a presença do logo.
- [ ] Reabrir o link consumido e acessar confirmação/redefinição sem token: conferir mensagem segura de link inválido, sem valor sensível na tela. Tokens substituídos, usados e inexistentes devem ter a mesma recusa na respectiva API.
- [ ] Conferir expiração do link de recuperação após 1 hora; quando não houver espera manual, registrar essa validação pela suíte de integração do fluxo 14.

## Fluxo 10 — Segundo estabelecimento, papéis e isolamento pela API

Cadastre e confirme B em outro perfil. Preserve IDs fictícios de A e B apenas para estas chamadas locais.

- [ ] Em B ainda sem logo, tentar primeiro upload inválido: conferir recusa definitiva e limpeza de seleção/pré-visualização, sem publicar a página. Depois, enviar logo válido, publicar B e criar um profissional e um serviço exclusivos dele.
- [ ] Como B, conferir configurações, lista de profissionais e catálogo: mostrar somente dados de B. Alternar de conta no mesmo perfil e conferir ausência de dados remanescentes de A.
- [ ] Como B, consultar IDs de profissional e serviço de A: esperar `404`, igual ao caso de UUID válido inexistente, sem revelar dados do recurso.
- [ ] Como B, tentar editar, desativar e reativar o profissional de A e editar ou alterar estado do serviço de A: recusar sem modificar A.
- [ ] Conferir que consulta/alteração de estabelecimento e upload de logo derivam o tenant da sessão, sem aceitar um identificador enviado pelo cliente para operar A como B.
- [ ] Sem sessão, consultar recursos protegidos: esperar `401`. Com o profissional já tendo trocado a senha, consultar e tentar modificar configurações, profissionais e serviços: esperar `403`.
- [ ] Como proprietário, tentar consultar/editar seu próprio ID pelas rotas de profissionais: não tratar o proprietário como perfil profissional válido.
- [ ] Nas listas de profissionais e serviços, tentar `limit=0`, `limit=101`, `offset=-1`, offset acima de `2147483647` e parâmetros não numéricos: esperar `400` com erro seguro. Conferir padrão `limit=20`, `offset=0` e `has_more` coerente.
- [ ] Em mutação autenticada, omitir ou adulterar `X-CSRF-Token`, preservando o restante válido: esperar `403` sem efeito. Repetir com origem ausente ou fora de `TRUSTED_ORIGINS`; mutações públicas também devem exigir origem confiável.
- [ ] Pela API, tentar PATCH de serviço vazio, nome/preço nulos e campos desconhecidos: recusar. Enviar um campo válido junto de outro inválido: não persistir atualização parcial.
- [ ] Pela API, tentar alterar e-mail, estado ou credenciais pelo PATCH de profissional; tentar alterar slug ou marcador operacional pelo PATCH de configurações: recusar e preservar os valores.
- [ ] Sem sessão, consultar os perfis públicos de A e B: cada slug deve resolver apenas o seu perfil/logo. Slug ausente ou reservado não deve publicar um estabelecimento.
- [ ] Conferir erros com código estável, mensagem em português e `request_id`; validações devem indicar os campos. Não exibir SQL, stack trace, credenciais ou dados de outro estabelecimento.

## Fluxo 11 — Falhas temporárias, resultados incertos e concorrência da interface

Execute com A ainda em trial ativo. Restaure cada falha antes do próximo cenário. Bloquear uma requisição antes do envio testa indisponibilidade; para testar resultado incerto após gravação, é necessário interromper/alterar a resposta depois que o backend receber a mutação, usando proxy local ou ferramenta equivalente. O modo offline sozinho não comprova que houve commit.

- [ ] Simular falha na consulta de `/auth/me`: mostrar erro e nova tentativa, sem afirmar logout em falha de rede ou `5xx`. Restaurar e recuperar a sessão.
- [ ] Simular falha temporária no logout: preservar a identidade exibida, permitir nova tentativa e não afirmar que os cookies foram removidos. Restaurar e confirmar o encerramento com `204`.
- [ ] Simular falha na consulta de configurações ao entrar em `/painel`: mostrar erro e nova tentativa; não redirecionar ao login por indisponibilidade. Restaurar e seguir ao destino correto.
- [ ] Interromper leitura de profissionais/serviços: mostrar erro e tentativa de recarga, com escritas bloqueadas enquanto `can_write` não estiver confirmado. Restaurar e conferir a liberação.
- [ ] Confirmar cadastro de profissional/serviço e falhar somente a recarga posterior da lista: manter sucesso e senha temporária limpa; tentar novamente deve recuperar a leitura, sem repetir o cadastro.
- [ ] Perder a resposta do cadastro: não anunciar sucesso nem reenviar automaticamente; orientar conferir a lista antes de outro cadastro.
- [ ] Perder resposta da edição de profissional/serviço: preservar rascunho, manter a lista com dados confirmados e exigir “Recarregar profissional”/“Recarregar serviço” antes de novo envio.
- [ ] Em duas sessões do proprietário, editar preço no serviço em uma e duração na outra. Após reconciliação, campos não editados devem receber os valores persistidos atuais; novo PATCH não deve sobrescrever a mudança concorrente.
- [ ] Perder resposta de desativação/reativação de profissional ou mudança de estado de serviço: exigir “Consultar estado” antes de liberar outra ação, sem repetir mutação ou afirmar sucesso apenas com base na intenção enviada.
- [ ] Na reativação incerta, falhar também a consulta: manter reenvio e fechamento bloqueados. Recuperar consulta: limpar senha e confirmar somente o estado; se ativo, não afirmar qual senha foi aplicada; se inativo, exigir novo preenchimento para outra tentativa.
- [ ] Confirmar PATCH de configurações e falhar a consulta posterior: bloquear novas edições/upload até “Atualizar configurações” concluir; não repetir o PATCH.
- [ ] Perder a resposta do upload: consultar configurações antes de decidir o logo atual. Se a consulta falhar, preservar seleção e oferecer “Atualizar estado do logo”, sem novo upload automático; após reconciliar, mostrar o logo persistido.
- [ ] Revogar uma sessão em outro perfil durante leitura lenta: ao receber `401`, retirar todo conteúdo protegido. Uma resposta antiga que termine depois não deve restaurá-lo.
- [ ] Alternar rapidamente página/seleção nas listas com rede lenta e depois trocar de conta: respostas antigas não devem substituir a seleção recente nem publicar dados da identidade anterior.
- [ ] Desligar temporariamente o backend, abrir uma página pública conhecida e conferir erro temporário com nova tentativa. Religá-lo e usar “Tentar novamente”: deve buscar novo render no servidor e recuperar o conteúdo, sem tratar falha como estabelecimento inexistente.
- [ ] Em conta fictícia adicional, parar o Mailpit antes do cadastro/reenvio/recuperação elegível: falha de envio após persistência deve preservar o resultado genérico documentado. Religá-lo, solicitar novo link e concluir o fluxo, sem recriar a conta persistida.

Comandos para interromper e restaurar serviços sem remover dados:

```sh
docker compose -f compose.yaml -f compose.dev.yaml stop backend
docker compose -f compose.yaml -f compose.dev.yaml start backend
docker compose -f compose.yaml -f compose.dev.yaml stop mailpit
docker compose -f compose.yaml -f compose.dev.yaml start mailpit
```

Execute somente o par do serviço necessário ao cenário; não pare ambos ao mesmo tempo.

## Fluxo 12 — Trial expirado e preservação dos dados

Deixe este fluxo para depois das escritas de A. Use exclusivamente o estabelecimento fictício descartável A e a preparação SQL abaixo para simular a passagem do tempo; B deve permanecer ativo como controle. Essa preparação altera a janela no banco local e não é uma operação disponível no produto nem uma forma de renovar o trial.

Abra o PostgreSQL local com o comando abaixo. `POSTGRES_USER` e `POSTGRES_DB` são lidos dentro do container, sem imprimir seus valores:

```sh
docker compose -f compose.yaml -f compose.dev.yaml exec postgres sh -c 'exec psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

Substitua somente o slug fictício em `target_slug` e execute no `psql`. Antes de confirmar, `UPDATE 1` e o booleano `expired = t` devem indicar uma única fixture; em qualquer outro caso use `ROLLBACK` em vez de `COMMIT`. O procedimento mantém duração de 14 dias e não exclui registros.

```sql
\set target_slug 'validacao-a-rodada-01'
BEGIN;
UPDATE establishment_trials AS trial
SET starts_at = now() - interval '15 days',
    ends_at = now() - interval '1 day'
FROM establishments AS establishment
WHERE establishment.id = trial.establishment_id
  AND establishment.slug = :'target_slug'
RETURNING trial.ends_at <= now() AS expired;
COMMIT;
```

- [ ] Recarregar `/testes` e consultar o estabelecimento: trial deve aparecer expirado; login, consulta de identidade e logout continuam funcionando.
- [ ] Reabrir configurações, profissionais e serviços: todos os dados e o logo permanecem consultáveis, com indicação de modo somente leitura e controles de escrita bloqueados.
- [ ] Pela API, tentar salvar configurações, substituir logo, cadastrar/editar/desativar/reativar profissional e cadastrar/editar/ativar/desativar serviço: esperar recusa `read_only`, sem qualquer alteração.
- [ ] Deixar uma página aberta antes da expiração e tentar gravar depois: a recusa deve atualizar os direitos de toda a página, bloqueando os demais formulários.
- [ ] Abrir perfil e logo públicos de A sem sessão: permanecer disponíveis e informativos após expiração.
- [ ] Entrar de novo e tentar reutilizar confirmação antiga: não reiniciar o trial. Não esperar tela de assinatura/renovação nesta entrega.
- [ ] Como B, confirmar leitura e escrita normais: a preparação de A não deve afetar outro estabelecimento.

## Fluxo 13 — Navegação, acessibilidade, persistência e proteção contra abuso

- [ ] Percorrer com Tab e Shift+Tab home, cadastro, login, recuperação, configurações, profissionais, serviços, confirmações de ação e `/testes`: conferir rótulos, foco visível, ordem coerente e acionamento por teclado.
- [ ] Conferir `aria-current` no link correto da navegação do painel; a entrada `/painel` não deve marcar Profissionais antecipadamente.
- [ ] Conferir erros associados aos campos, mensagens de carregamento/vazio/erro/sucesso e foco devolvido após cancelamentos; estados desabilitados não devem permitir envio duplicado.
- [ ] Repetir os fluxos visuais nos temas claro e escuro do sistema e em viewport estreito e desktop: sem campos cortados ou ações inacessíveis, com nomes/e-mails longos quebrando adequadamente.
- [ ] Abrir `/design-system` em desenvolvimento: conferir componentes reais e estados usados no produto. Na verificação de build de produção do fluxo 14, a rota deve resultar em página não encontrada.
- [ ] Executar `make dev-down` e subir novamente com `make dev`, preservando volumes: contas, serviços, profissionais, configurações, trial expirado de A e logos devem permanecer. Conferir logo público também após reiniciar somente o backend.
- [ ] Conferir que somente as portas do Caddy e a UI local do Mailpit estão publicadas; PostgreSQL, SMTP e aplicações internas não devem estar expostos diretamente pelo Compose.
- [ ] Inspecionar os logs locais dos fluxos executados: conferir método, rota parametrizada, status, duração e `request_id`, sem corpos, senhas, tokens, e-mails completos ou caminhos internos de storage.
- [ ] Fazer o teste de limitação de login por último: com conta fictícia, provocar 5 falhas em 15 minutos e tentar novamente; esperar `429` com `Retry-After`. A dimensão de IP pode afetar outros logins locais, portanto aguardar o prazo antes de retomar testes de autenticação.
- [ ] Para cadastro, reenvio de confirmação e recuperação, repetir solicitações fictícias até o limite do respectivo escopo: conferir `429`, `Retry-After` e mensagem segura, inclusive para conta ausente. Não apagar eventos do banco para contornar a proteção.
- [ ] Conferir no navegador que tokens transacionais não ficam em query, path ou `Referer`, que senhas não aparecem nas URLs e que tokens/credenciais não são persistidos em Web Storage. Não exportar HAR ou capturas com conteúdo sensível.

Nunca use `docker compose down -v` neste roteiro: não há backup no escopo inicial e isso remove os volumes persistentes.

## Fluxo 14 — Verificações automatizadas complementares

Estes comandos são verificações a executar no host; este documento não registra que elas já passaram. Instale as versões fixadas e dependências com `make setup`. Docker deve estar acessível ao processo que executa a integração; seu PostgreSQL é descartável e separado do banco local usado acima.

- [ ] Executar `make test-unit`: testes unitários do Go e Vitest devem passar.
- [ ] Executar `make test-integration`: fluxos HTTP reais, migrations e PostgreSQL devem passar, incluindo isolamento, autorização, sessões, tokens de uso único/expiração e recusas de escrita.
- [ ] Na integração, conferir os cenários já existentes de sessão expirada, trial ausente, bloqueio de fuso após marcador operacional e slug reservado legado preservado sem publicação. Eles exigem fixtures específicas e complementam os caminhos criados pela interface.
- [ ] Na integração, conferir os cenários de concorrência e rollback já existentes: confirmação/reenvio, recuperação concorrente, login versus desativação, troca de senha em andamento versus reativação, reativações concorrentes, associação concorrente de logo, atualização parcial e snapshot consistente de configurações. Esses cenários complementam a validação manual e não exigem introduzir E2E.
- [ ] Executar `make check`: geração e comparação de artefatos, `gofmt`, `go vet -tags=integration ./...`, testes Go e integração, Prettier, ESLint, tipos e Vitest devem passar. O comando já repete as suítes anteriores; pode ser usado como execução consolidada.
- [ ] Executar `make build`: builds de produção do backend e frontend devem concluir.
- [ ] Para conferir o frontend de produção com o ambiente local já ativo, usar o Compose de produção com o estágio `runtime`, mantendo a API em desenvolvimento e o Mailpit local conforme abaixo. Conferir home, login, painel e página pública, e `/design-system` como não encontrada. Ao terminar, restaurar o frontend de desenvolvimento.
- [ ] Registrar falhas e itens não executados com fluxo, passo, resultado esperado/observado e, quando disponível, `request_id`, sem dados sensíveis. Não marcar cenários apenas porque a tela abriu.

Frontend em produção, preservando os demais serviços locais:

```sh
docker compose -f compose.yaml up -d --build --no-deps frontend
```

Restaurar frontend de desenvolvimento após essa conferência:

```sh
docker compose -f compose.yaml -f compose.dev.yaml up -d --build --no-deps frontend
```

## Registro da rodada

| Campo                          | Preencher |
| ------------------------------ | --------- |
| Data / responsável             |           |
| Branch / commit validado       |           |
| Navegador / sistema            |           |
| Fluxos concluídos              |           |
| Falhas encontradas             |           |
| Itens não executados e motivo  |           |
| Evidências sem dados sensíveis |           |
