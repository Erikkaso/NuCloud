Servidor será escrito em Go
frontend a definir
o orquestrador de containers será em kubernetes
ci/cd será usando gitlab
build será usando railpack

Usuário

* usuário entra em tela de login ou registrar
* cria a conta
* seleciona repositório do github que deseja fazer o build
* usuário faz configurações de deploy
* usuário informa variaveis de ambiente quando necessário

Aplicação

* aplicação usa gitlab para fazer fluxo de ci/cd
* gitlab pega código o código do repositório
* gitlab utiliza railpack pra compilar código


RF:

* O usuário deve se registrar para usar os serviços.

* O usuário deve poder escolher entre os planos fornecidos pelo sistema, e só usar o serviço após o pagamento

* O usuário deve poder fazer um deploy fácil da sua aplicação com apenas poucos clicks

* O usuário deve poder configurar como desejar seu container dependendo do seu plano

* O usuário deve poder conseguir fazer deploy de aplicações backends com varias linguagens e arquiteturas diferentes

* O usuário deve receber uma url https válida que pode ser usada externamente para conectar seu frontend ao servidor

* O sistema deve vasculhar o código do usuário em busca de possíveis riscos e virus

* O sistema deve pegar o código do usuário do GitHub e gerar uma imagem em cima dele usando ferramentas como: railpack Cloud Native Buildpacks (CNB) Paketo Buildpacks

* O sistema deve automatizar o deploy da imagem utilizando gitlab

* O sistema deve orquestrar os containers utilizando um cluster kubernetes

* Próximo do fim do plano de um usuário o sistema deve notificar o usuário

* No fim do plano de um usuário o sistema deve parar o container e exclui-lo em caso de inatividade para liberar espaço para outros

* Se possível, ter um painel de métricas para o usuário ver

* O usuário deve poder acompanhar o console do container em tempo real

* O sistema deve fornecer tanto serviços stateful quanto stateless.

* O sistema deve estar apto a dar rollback pra uma versão estável em caso de falhas de build





RNF:

* O sistema não deve se sobrecarregar se uma aplicação consumir mais do que deveria

* O sistema deve armazenar de forma privada e segura as variáveis de ambiente da aplicação

* O sistema deve ter proteções de DDos

* O sistema deve aguentar no mínimo 500 usuários simultaneos

* O sistema deve ter uma interface fácil e amigável para o usuário

* O sistema deve isolar adequadamente cada aplicação

* O sistema deve seguir a filosofia Zero-Downtime Deployments







TODO:

* COLOCAR gitlab CI/CD no k8s

* Testar CI/CD do gitlab

* Testar geração de imagem automática

* Testar deploy automático com geração de: deployment,service e ingress no k8s

* Criar backend e testar comunicação com o K8s



