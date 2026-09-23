Servidor será escrito em Go
frontend a definir
o orquestrador de containers será em kubernetes
ci/cd será usando gitlab 
build será usando railpack

Usuário
- usuário entra em tela de login ou registrar
- cria a conta
- seleciona repositório do github que deseja fazer o build
- usuário faz configurações de deploy
- usuário informa variaveis de ambiente quando necessário

Aplicação
- aplicação usa gitlab para fazer fluxo de ci/cd
- gitlab pega código o código do repositório
- gitlab utiliza railpack pra compilar código
