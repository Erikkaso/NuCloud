# NuCloud — Backend inicial (protótipo de build via GitLab CI)

Backend mínimo em Go que recebe a URL de um repositório GitHub e dispara uma pipeline no GitLab. A pipeline clona o repositório, detecta a tecnologia com **Railpack**, constrói a imagem com **BuildKit** (rootless) e publica no **Docker Hub**.

> Status: protótipo. O deploy no K3s ainda não está implementado (o job `deploy-job` é demonstrativo).

---

## 1. Visão geral

```text
Cliente ──POST /buildapp (url)──► Backend Go
                                     │  Pipeline Trigger API (REST, HTTPS)
                                     ▼
                                 GitLab.com (projeto "builder")
                                     │  fila de jobs
                                     ▼
                         Runner (compartilhado do GitLab OU worker local)
                                     │
        clona REPO_URL ► railpack prepare ► buildctl + railpack-frontend
                                     │
                                     ▼
                                 Docker Hub (imagem)
```

Pontos de design:

- O repositório do cliente **fica no GitHub**. Ele é clonado dentro do job (container descartável) e não é espelhado no GitLab.
- O projeto GitLab guarda só o `.gitlab-ci.yml` genérico. Cada pedido dispara uma pipeline nesse mesmo projeto, passando `REPO_URL` como variável.
- O código **não** é passado entre jobs como artifact (artifacts sobem para o GitLab Cloud). Só o `railpack-plan.json` e o `railpack-info.json` passam.

---

## 2. Pré-requisitos

- Go 1.22+ (ou a versão que o seu `go.mod` indicar).
- Conta no GitLab.com e um **projeto privado** para as pipelines.
- Conta no Docker Hub e um **Personal Access Token** com permissão *Read & Write*.
- Para o worker local: Docker Engine instalado.

---

## 3. Configuração do projeto GitLab

Faça isto uma vez.
Crie um projeto novo apenas com um `.gitlab-ci.yml`.

### 3.1 Pipeline trigger token

*Settings > CI/CD > Pipeline trigger tokens > Add new token*. É o token que o backend usa para disparar pipelines. Exige papel Maintainer.

### 3.2 Permitir variáveis de pipeline

*Settings > CI/CD > Variables > Minimum role to use pipeline variables*.

Se estiver em **No one allowed**, a API recusa o disparo com:

```text
{"message":{"base":["Insufficient permissions to set pipeline variables"]}}
```

Escolha um papel que inclua o dono do trigger token (por exemplo, Developer ou Maintainer).

### 3.3 Variável de CI/CD com o token do Docker Hub

*Settings > CI/CD > Variables*: crie `DOCKER_TOKEN` (tipo *Variable*, marque **Masked**). O valor é o Personal Access Token do Docker Hub.

> Masked não impede que um script malicioso leia a variável. Em produção, não guarde segredos de produção no GitLab Cloud.

### 3.4 `.gitlab-ci.yml`

Na raiz do projeto GitLab. A versão abaixo é a que funcionou nos testes.

```yaml
stages:
  - test
  - prepare
  - build
  - deploy

variables:
  IMAGE_NAME: "docker.io/nuclic/testbuild"   # adapte: docker.io/<seu-usuario>/<repositorio>

lint-test-job:
  stage: test
  image: alpine:3.21
  # tags: [wsl]        # descomente para forçar o worker local (ver seção 6)
  script:
    - echo "Running lint tests..."
    - sleep 10
    - echo "No lint issues found."

railpack-prepare:
  stage: prepare
  image: alpine:3.21
  # tags: [wsl]
  before_script:
    - apk add --no-cache git curl ca-certificates tar
    - curl -fsSL https://railpack.com/install.sh | sh
    - railpack --version
  script:
    - git clone --depth 1 -- "$REPO_URL" app
    - railpack prepare app --plan-out railpack-plan.json --info-out railpack-info.json
  artifacts:
    paths:
      - railpack-plan.json
      - railpack-info.json
    expire_in: 1 hour

build-job:
  stage: build
  # tags: [wsl]
  image:
    name: moby/buildkit:rootless
    entrypoint: [""]
  variables:
    BUILDKITD_FLAGS: "--oci-worker-no-process-sandbox"
  needs:
    - job: railpack-prepare
      artifacts: true
  script:
    - git --version
    - git clone --depth 1 -- "$REPO_URL" app
    - cp railpack-plan.json app/
    - |
      mkdir -p "$HOME/.docker"
      AUTH="$(printf '%s' "nuclic:$DOCKER_TOKEN" | base64 | tr -d '\n')"
      printf '{"auths":{"https://index.docker.io/v1/":{"auth":"%s"}}}\n' "$AUTH" > "$HOME/.docker/config.json"
      chmod 600 "$HOME/.docker/config.json"
      export DOCKER_CONFIG="$HOME/.docker"
    - |
      buildctl-daemonless.sh build \
        --frontend=gateway.v0 \
        --local context=app \
        --local dockerfile=app \
        --opt source=ghcr.io/railwayapp/railpack-frontend:latest \
        --output type=image,name=$IMAGE_NAME:v1,push=true

deploy-job:
  stage: deploy
  image: alpine:3.21
  environment: production
  script:
    - echo "Deploying application..."
    - echo "Application successfully deployed."
  rules:
    - if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH'
```

Pontos de atenção neste arquivo:

- **Chave de autenticação do Docker Hub:** use `https://index.docker.io/v1/` (a chave canônica). Com `docker.io` o push falhou com `insufficient_scope` nos testes iniciais.
- **Versões:** o railpack frontend está em `:latest`. Fixe a versão do frontend na mesma do `railpack --version` mostrada no log antes de usar em produção.
- **Tag `v1` fixa:** cada build sobrescreve a imagem. Para algo além de testes, use uma tag única, como `$CI_PIPELINE_ID`.

---

## 4. Backend Go

### 4.1 Variáveis de ambiente (`.env`)

```dotenv
GITLAB_CLIENT_TOKEN=<token usado para criar o client da API>
GITLAB_PIPELINE_TOKEN=<pipeline trigger token do projeto>
PROJECT_ID=<ID numérico do projeto GitLab>
```

- O `PROJECT_ID` aparece em *Settings > General* do projeto.
- Adicione `.env` ao `.gitignore`. **Nunca versione tokens.**
- O disparo usa o `GITLAB_PIPELINE_TOKEN`. Para ler o status de pipelines (próxima etapa), prefira um **project access token** com escopo `read_api` e papel mínimo, em vez de um token pessoal.


### 4.2 Executar e testar

```bash
go mod tidy
go run .
```

Em outro terminal:

```bash
curl -X POST -d "url=https://github.com/<dono>/<repo>" http://localhost:8080/buildapp
```

O endpoint responde assim que o GitLab aceita o disparo. A pipeline segue de forma assíncrona. Acompanhe em *Build > Pipelines* no GitLab.

### 4.3 Testar o disparo direto, sem o backend

```bash
read -rs TRIGGER_TOKEN
curl --request POST \
  --form "token=$TRIGGER_TOKEN" \
  --form "ref=main" \
  --form "variables[REPO_URL]=https://github.com/<dono>/<repo>" \
  "https://gitlab.com/api/v4/projects/<PROJECT_ID>/trigger/pipeline"
```

`read -rs` evita que o token apareça na tela e no histórico do shell.

---

## 5. Opção A — Runners compartilhados do GitLab

É o caminho mais simples: não precisa preparar nada na sua máquina.

1. Deixe *Settings > CI/CD > Runners > Enable instance runners* **ligado**.
2. Mantenha as linhas `tags:` comentadas no `.gitlab-ci.yml`.
3. Dispare a pipeline.

Nos testes, o BuildKit rootless funcionou nesses runners sem nenhum ajuste.

**Contrapartidas:**
- Consome os minutos de computação da sua conta GitLab.
- O repositório do cliente é clonado **na infraestrutura do GitLab**. Isso conflita com o requisito de privacidade do PaaS. Use só com repositórios de teste.
- O `DOCKER_TOKEN` é exposto aos jobs rodando nessa infraestrutura.

---

## 6. Opção B — Worker local (GitLab Runner em Docker)

O GitLab.com só coordena. O **Runner** busca jobs com chamadas HTTPS de saída (sem abrir portas) e o **executor `docker`** cria um container por job.

### 6.1 Criar o runner no GitLab

*Settings > CI/CD > Runners > Create project runner*. Informe uma tag (por exemplo, `wsl`) ou marque *Run untagged*. Copie o token (`glrt-...`) na hora: ele aparece na interface por pouco tempo.

> Trate o token como segredo. Não o cole em chats ou comandos com texto puro. Se vazar, reset ou recrie o runner.

### 6.2 Registrar o runner

> Crie o registro na máquina que deseja criar o worker.

O registro só grava o `config.toml` em um volume nomeado. Não precisa do `docker.sock` nesta etapa.

```bash
read -rs RUNNER_TOKEN

docker run --rm -it \
  -v gitlab-runner-config:/etc/gitlab-runner \
  gitlab/gitlab-runner:alpine register \
  --non-interactive \
  --url https://gitlab.com \
  --token "$RUNNER_TOKEN" \
  --executor docker \
  --docker-image alpine:3.21
```

> O `-v` aceita **duas** partes (`origem:destino`). Um volume com três partes (`nome:/caminho:/destino`) gera `invalid mode`.

### 6.3 Subir o runner (`docker-compose.yml`)

```yaml
services:
  gitlab-runner:
    image: gitlab/gitlab-runner:alpine
    container_name: gitlab-runner
    restart: always
    volumes:
      - gitlab-runner-config:/etc/gitlab-runner       # volume nomeado (config.toml)
      - /var/run/docker.sock:/var/run/docker.sock     # bind mount: acesso ao Docker do host

volumes:
  gitlab-runner-config:
    external: true    # reaproveita o volume criado no registro
```

```bash
docker compose up -d
```

O `docker.sock` precisa ser um **bind mount** (é um arquivo do host). É por ele que o executor `docker` cria os containers dos jobs, ao lado do container do runner.

Acesso ao `docker.sock` equivale a acesso root no host. Aceitável no experimento local, não em produção.

### 6.4 Liberar o BuildKit rootless

Sem isto, o `build-job` falha com:

```text
[rootlesskit:parent] error: failed to start the child: fork/exec /proc/self/exe: operation not permitted
could not connect to unix:///run/user/1000/buildkit/buildkitd.sock after 10 trials
```

O perfil `seccomp` padrão do Docker bloqueia o que o BuildKit rootless precisa. Insira a opção na seção `[runners.docker]` do `config.toml` (rode **uma vez**):

```bash
docker exec gitlab-runner sed -i '/\[runners\.docker\]/a security_opt = ["seccomp:unconfined", "apparmor:unconfined"]' /etc/gitlab-runner/config.toml
docker restart gitlab-runner
```

Conferir sem expor o token:

```bash
docker exec gitlab-runner grep -n -A3 'runners.docker' /etc/gitlab-runner/config.toml
```

> **Segurança:** isso vale para todos os jobs desse runner e reduz o isolamento. Para builds de clientes não confiáveis em produção, use isolamento adicional (VMs ou Kubernetes com sandbox).

### 6.5 Forçar os jobs para o seu runner

Se os runners compartilhados estiverem ligados e os jobs não tiverem tag, o GitLab pode entregá-los a um runner dele (você percebe porque o `docker ps` local não mostra containers).

1. Desligue *Enable instance runners* no projeto.
2. Descomente `tags: [wsl]` em todos os jobs do `.gitlab-ci.yml` (use a tag do seu runner) *apenas se tiver criado uma tag no create runner*.

### 6.6 Verificar

- No log do job, procure `Running with gitlab-runner ... on <nome-do-seu-runner>` e `Using Docker executor`.
- Durante a execução, `docker ps` deve mostrar containers temporários além do `gitlab-runner`.
- `docker logs --tail 30 gitlab-runner` mostra os jobs recebidos.

### 6.7 Configurações opcionais do runner (`config.toml`)

| Opção | Efeito |
|---|---|
| `concurrent = N` (topo do arquivo) | Quantos jobs rodam ao mesmo tempo. Com `1`, o segundo job fica em *pending* até o primeiro terminar. |
| `request_concurrency = 2` (em `[[runners]]`) | Evita atrasos de long polling (o runner avisa disso no log). |

Builds consomem CPU, RAM e disco. No WSL2, a memória é limitada pelo `.wslconfig`. Aumente o `concurrent` aos poucos e meça.

---

## 7. Solução de problemas

| Sintoma | Causa | Correção |
|---|---|---|
| `docker: invalid mode: /etc/gitlab-runner` | `-v` com três partes | Use `volume:/destino` (duas partes) |
| `Expected string \| array` no editor de CI | Linha de `script` com `: ` sem aspas | Envolva a linha em aspas simples |
| `Insufficient permissions to set pipeline variables` | *Minimum role to use pipeline variables* em "No one allowed" | Ajuste a opção (seção 3.2) |
| `insufficient_scope` no push | Chave de auth incorreta no `config.json` | Use `https://index.docker.io/v1/` |
| `operation not permitted` / `buildkitd.sock` | Isolamento do Docker bloqueia o rootless | Seção 6.4 |
| Job roda mas `docker ps` está vazio | Runner compartilhado pegou o job | Seção 6.5 |
| Job em *pending* | Falta a tag no job, runner offline ou `concurrent` ocupado | Confira tags, `docker logs gitlab-runner` e o `concurrent` |
| HTTP 429 na API | Rate limit do GitLab.com | Fila no backend e espera com backoff |

---

## 8. Segurança e privacidade

- **Tokens:** cada um com o menor escopo possível. Resumo:
  - `glrt-` (runner): identifica o worker.
  - Pipeline trigger token: só dispara pipelines.
  - Personal/Project access token: age na API (prefira o de projeto, com `read_api`).
  - `CI_JOB_TOKEN`: automático e temporário, usado pelo job.
  - Docker Hub PAT: só publicar imagens.
- Não cole tokens em terminais, chats ou logs. Prefira `read -rs` e variáveis de ambiente.
- Valide toda entrada do cliente no backend (a URL, no mínimo).
- O GitLab Cloud ainda recebe nomes de projeto, status, **logs** e artifacts. Evite imprimir segredos nos jobs.
- **Rate limits:** os limites do GitLab.com valem por usuário/token e são compartilhados entre todos os clientes do backend. Use fila, backoff no 429 e, no futuro, webhooks de pipeline em vez de polling.

---

## 9. Limitações e próximos passos

- Fixar as versões do Railpack CLI e do frontend (sem `:latest`).
- Usar tags de imagem únicas por build.
- Passar o `COMMIT_SHA` para que os dois jobs clonem o mesmo commit.
- Consultar o status da pipeline (token de projeto com `read_api`) ou receber webhooks de pipeline.
- Autenticar o acesso aos repositórios privados dos clientes com um **GitHub App** e tokens de instalação de curta duração.
- Deploy real no K3s com `client-go` e ServiceAccount com RBAC limitado (nunca usar o kubeconfig administrativo).
- Migrar o runner para o **Kubernetes Executor** no K3s, com isolamento entre clientes.
- Cache de build exportado para o registry (hoje cada job baixa tudo do zero).