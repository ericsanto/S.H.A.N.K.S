# Especificação do comando `prune`

## 1. Objetivo

O comando `prune` deve desfazer as alterações realizadas pelo comando `start` e devolver o namenode e os datanodes ao estado anterior à criação do cluster.

Uso proposto:

```bash
./shanks prune --config ./config.yml
```

O `prune` é diferente do `stop`:

| Comando | Containers | Volumes e dados HDFS | Imagens Docker | `/etc/hosts` | `.env` |
| --- | --- | --- | --- | --- | --- |
| `stop` | Para e remove | Preserva | Preserva | Preserva | Preserva |
| `prune` | Para e remove | Remove por padrão | Remove o que o `start` criou | Remove | Remove ou restaura backup |

> **Atenção:** remover os volumes apaga definitivamente os dados do HDFS e os dados persistidos pelo Grafana. O comando deve pedir confirmação antes dessa etapa, exceto quando `--yes` for informado.

## 2. O que o `start` modifica atualmente

O fluxo atual está definido em `cli/backend/cmd/start.go`:

1. Lê o YAML com `ReadYaml`.
2. Atualiza `/etc/hosts` com `ConfigHosts`.
3. Calcula os limites do YARN e distribui um arquivo `.env` com `ConfigureYarnLimits`.
4. Constrói imagens e inicia o Docker Compose com `StartCluster`.

### 2.1 Arquivo `/etc/hosts`

O `start` altera o `/etc/hosts` do namenode local e de cada datanode via SSH. O conteúdo gerenciado fica delimitado por:

```text
# START S.H.A.N.K.S #
<IP do namenode> <nome do namenode>
<IP do datanode> <nome do datanode>
# END S.H.A.N.K.S #
```

Antes de escrever, o código remove qualquer bloco anterior que use os mesmos marcadores. Portanto, o `prune` deve remover somente o intervalo entre esses marcadores, sem alterar as demais linhas do arquivo.

Hosts afetados:

- namenode: máquina local na qual a CLI foi executada;
- datanodes: todas as entradas de `cluster.datanodes` no YAML.

### 2.2 Arquivo `.env`

O `start` cria ou sobrescreve parcialmente o arquivo `.env` no diretório em que a CLI foi executada. O arquivo contém valores como:

```dotenv
master_YARN_LIMIT=2000
datanode_1_YARN_LIMIT=2000
```

Depois, o arquivo é copiado para cada datanode em:

```text
$HOME/S.H.A.N.K.S/.env
```

No namenode, os arquivos Compose leem `S.H.A.N.K.S/.env`. O `prune` deve:

1. remover o `.env` remoto de todos os datanodes;
2. remover o `.env` local se ele tiver sido criado pelo `start`;
3. restaurar o conteúdo anterior quando o arquivo já existia antes do `start`.

### 2.3 Recursos Docker no namenode

O Compose `new_cluster/master/docker-compose.master.yml` inicia:

- `hadoop-master`;
- `prometheus`;
- `grafana`;
- `exporter-master`.

Ele também cria ou reutiliza os volumes:

- `master_namenode_data`;
- `master_namenode_data_datanode`;
- `master_grafana`.

Imagens envolvidas:

- `hadoop-base`, construída diretamente pelo `start` se ainda não existir;
- `master-master`, construída pelo Compose;
- `prom/prometheus`;
- `grafana/grafana`;
- `prom/node-exporter`.

### 2.4 Recursos Docker em cada datanode

O Compose `new_cluster/worker/docker-compose.worker.yml` inicia:

- `worker`;
- `worker-node-exporter-worker-1` (nome normalmente gerado pelo Compose).

Ele também cria ou reutiliza:

- volume `worker_datanode_data`;
- imagem `worker-worker`;
- imagens `hadoop-base` e `prom/node-exporter`.

Como todos os serviços usam rede `host`, não existe uma rede exclusiva do Hadoop que precise ser preservada. Mesmo assim, `docker compose down` deve remover qualquer rede Compose residual.

### 2.5 Configurações internas dos containers

Durante o entrypoint, os containers geram ou alteram:

- `core-site.xml`;
- `hdfs-site.xml`;
- `yarn-site.xml`;
- `mapred-site.xml`;
- `hadoop-env.sh`;
- `yarn-env.sh`;
- arquivo `workers`;
- diretórios de dados do NameNode e DataNodes;
- formatação inicial do NameNode;
- processos NameNode, DataNode, ResourceManager e NodeManager.

Os arquivos de configuração ficam na camada gravável dos containers. Eles desaparecem quando os containers são removidos. Os dados do HDFS ficam nos volumes e só desaparecem quando os volumes também são removidos.

### 2.6 Entrada em `known_hosts`

A cópia do `.env` usa `scp` com `StrictHostKeyChecking=accept-new`. Isso pode adicionar a chave de um datanode ao arquivo local `~/.ssh/known_hosts`.

Essa entrada só pode ser removida com segurança se o `start` registrar que ela não existia anteriormente e qual chave foi adicionada. Sem esse registro, o `prune` não deve remover entradas de `known_hosts`, pois elas podem pertencer ao usuário ou a outro projeto.

### 2.7 Cache de build do Docker

Os comandos `docker build --no-cache` e `docker compose ... up --build` podem deixar cache de build além das imagens finais. Um `docker builder prune` sem filtros afetaria builds de outros projetos e, por isso, não deve fazer parte do fluxo automático.

Para que esse cache possa ser removido com segurança no futuro, o `start` deve construir as imagens com labels do cluster e usar um builder dedicado ou outro mecanismo de identificação. Sem isso, o `prune` deve apenas informar que pode existir cache Docker residual.

### 2.8 Estados parciais deixados por falhas do `start`

O `start` executa suas etapas em sequência, mas não faz rollback quando uma delas falha. O `prune` não pode presumir que o cluster chegou a iniciar:

| Ponto da falha | Estado que pode ter ficado no ambiente |
| --- | --- |
| Durante `ConfigHosts` | bloco instalado no namenode e somente em parte dos datanodes |
| Durante os cálculos do YARN | blocos em todos os hosts, mas nenhum `.env` completo |
| Durante o `scp` | `.env` local e arquivo remoto somente em parte dos datanodes |
| Durante o build | hosts e `.env` configurados, imagens presentes apenas em alguns nós |
| Durante o Compose | combinação parcial de containers, imagens e volumes entre os nós |

Por isso, cada etapa do `prune` deve primeiro consultar o estado real do recurso, considerar ausência como sucesso e continuar mesmo se outro recurso não existir.

## 3. Estado necessário para uma reversão segura

O código atual não registra quais recursos já existiam antes do `start`. Sem essa informação, não é possível distinguir com total segurança um recurso criado pelo S.H.A.N.K.S de um recurso anterior com o mesmo nome.

Antes de implementar o `prune`, o `start` deve criar um manifesto local, por exemplo:

```text
.shanks/state/<cluster-id>.json
```

O manifesto deve registrar:

- caminho absoluto e hash do YAML utilizado;
- data e estado da execução;
- namenode e datanodes alcançados;
- se o bloco S.H.A.N.K.S já existia no `/etc/hosts` e seu conteúdo anterior;
- se o `.env` local e remoto já existia, com backup, permissões e proprietário;
- IDs das imagens Docker existentes antes do `start`;
- imagens criadas ou baixadas pelo `start`;
- volumes que já existiam e volumes criados pelo `start`;
- entradas adicionadas ao `known_hosts`;
- resultado de cada etapa, inclusive execuções parcialmente concluídas.

O manifesto deve ser atualizado logo após cada alteração. Assim, um `start` que falhe no meio também pode ser revertido.

Na ausência do manifesto, o `prune` deve operar em modo conservador:

- pode remover o bloco marcado do `/etc/hosts`;
- pode executar `docker compose down`;
- deve avisar antes de excluir `.env`, volumes ou imagens;
- não deve modificar `known_hosts` automaticamente;
- deve informar que a restauração exata do estado anterior não é garantida.

## 4. Passo a passo do comando `prune`

### Passo 1 — Ler e validar a configuração

1. Exigir `--config` e rejeitar caminho vazio.
2. Ler o mesmo YAML utilizado no `start`.
3. Validar nome, usuário e IP de todos os nós.
4. Rejeitar nomes ou caminhos que possam gerar comandos de shell inseguros.
5. Resolver o diretório home do usuário original com a mesma regra de `getHomeDir`.
6. Localizar o manifesto correspondente ao cluster, quando existir.

Se o YAML não puder ser lido, nenhum recurso deve ser alterado.

### Passo 2 — Fazer o inventário antes da remoção

Verificar, sem alterar o sistema:

- conectividade SSH com todos os datanodes;
- disponibilidade de `sudo` para editar `/etc/hosts`;
- existência dos projetos Compose do master e dos workers;
- containers em execução ou parados;
- volumes e imagens presentes;
- existência do bloco S.H.A.N.K.S em cada `/etc/hosts`;
- existência dos arquivos `.env` local e remotos;
- existência do manifesto de estado.

O resultado deve formar um plano de remoção. Com `--dry-run`, o comando imprime esse plano e encerra sem modificar nada.

### Passo 3 — Confirmar a operação destrutiva

Antes de apagar volumes, mostrar claramente:

- quantidade de nós afetados;
- volumes que serão removidos;
- aviso de perda dos dados HDFS e Grafana;
- imagens que serão removidas;
- arquivos de host e `.env` que serão alterados.

Continuar somente após confirmação explícita. A flag `--yes` permite uso não interativo.

### Passo 4 — Encerrar os serviços Hadoop de forma ordenada

Antes de remover os containers, tentar uma parada graciosa. A ausência de um container ou daemon não deve ser tratada como erro fatal.

Em paralelo, em cada datanode:

```bash
docker compose -f docker-compose.worker.yml exec -T worker \
  yarn --daemon stop nodemanager

docker compose -f docker-compose.worker.yml exec -T worker \
  hdfs --daemon stop datanode
```

No container master, parar primeiro os serviços que também executam como worker:

```bash
docker compose -f docker-compose.master.yml exec -T master \
  yarn --daemon stop nodemanager

docker compose -f docker-compose.master.yml exec -T master \
  hdfs --daemon stop datanode
```

Depois, parar os serviços centrais:

```bash
docker compose -f docker-compose.master.yml exec -T master \
  yarn --daemon stop resourcemanager

docker compose -f docker-compose.master.yml exec -T master \
  hdfs --daemon stop namenode
```

Se a parada graciosa falhar, registrar o erro e continuar para `docker compose down`. O objetivo é não deixar os demais nós sem limpeza por causa da falha de um único nó.

### Passo 5 — Remover os projetos Docker Compose

Em cada datanode, via SSH:

```bash
cd "$HOME/S.H.A.N.K.S/new_cluster/worker"
docker compose -f docker-compose.worker.yml down \
  --volumes --rmi local --remove-orphans
```

No namenode:

```bash
cd "$HOME/S.H.A.N.K.S/new_cluster/master"
docker compose -f docker-compose.master.yml down \
  --volumes --rmi local --remove-orphans
```

Esse passo remove os containers, volumes, imagens construídas pelo Compose e eventuais redes residuais.

Regras importantes:

- `--keep-volumes` deve omitir `--volumes` para preservar HDFS e Grafana;
- uma execução repetida deve ser aceita mesmo quando os recursos já não existem;
- os workers podem ser processados em paralelo;
- os erros devem ser agregados por nó, sem interromper os demais.

### Passo 6 — Remover imagens externas ao Compose

A imagem `hadoop-base` é criada por um `docker build` separado e não é necessariamente removida por `docker compose down --rmi local`.

Remover `hadoop-base` no namenode e nos datanodes somente quando o manifesto indicar que ela foi criada pelo `start` que está sendo desfeito:

```bash
docker image rm hadoop-base
```

As imagens compartilhadas `prom/prometheus`, `grafana/grafana` e `prom/node-exporter` também só devem ser removidas se o manifesto comprovar que foram baixadas por essa execução e se nenhum outro container as utiliza.

A flag `--keep-images` deve pular toda esta etapa e omitir `--rmi local` na etapa anterior.

### Passo 7 — Restaurar ou remover os arquivos `.env`

Para cada datanode:

1. se havia `.env` antes do `start`, restaurar o backup registrado;
2. se o arquivo foi criado pelo `start`, remover `$HOME/S.H.A.N.K.S/.env`;
3. se não houver manifesto, pedir confirmação antes da remoção.

Aplicar a mesma regra ao `.env` local do namenode.

A limpeza deve ocorrer depois do `docker compose down`, pois o Compose precisa ler o arquivo enquanto encerra o projeto.

### Passo 8 — Limpar o `/etc/hosts`

Em todos os datanodes e no namenode, remover apenas o bloco delimitado pelos marcadores:

```text
# START S.H.A.N.K.S #
# END S.H.A.N.K.S #
```

Requisitos da operação:

- executar a alteração de forma atômica, usando arquivo temporário e `mv`;
- preservar proprietário e permissões de `/etc/hosts`;
- não remover entradas semelhantes fora dos marcadores;
- considerar sucesso quando o bloco já não existir;
- restaurar o bloco anterior se ele estiver registrado no manifesto;
- nunca imprimir ou armazenar a senha de `sudo` em logs.

### Passo 9 — Reverter `known_hosts` quando for seguro

Se o manifesto registrar exatamente uma chave adicionada pelo `start`, remover somente essa chave. Se não houver registro, apenas emitir um aviso e preservar `~/.ssh/known_hosts`.

O `prune` não deve remover todas as chaves associadas ao IP com `ssh-keygen -R`, pois isso também pode apagar uma chave que já existia antes do cluster.

### Passo 10 — Verificar o resultado

Ao final, verificar em todos os nós:

- nenhum container do projeto continua em execução;
- os containers do projeto não permanecem parados;
- volumes gerenciados foram removidos, exceto com `--keep-volumes`;
- imagens gerenciadas foram removidas, exceto com `--keep-images`;
- o bloco S.H.A.N.K.S não existe mais no `/etc/hosts`;
- o `.env` foi removido ou restaurado;
- as portas dos serviços deixaram de ser ocupadas pelos containers do projeto;
- não existem recursos Compose órfãos.

O resumo deve apresentar o resultado por host e por recurso, por exemplo:

```text
[OK] master: containers removidos
[OK] master: volumes removidos
[OK] master: bloco /etc/hosts removido
[OK] datanode_1: containers removidos
[FAIL] datanode_1: não foi possível remover .env

Prune concluído parcialmente: 1 erro.
```

O manifesto só deve ser excluído quando todas as etapas obrigatórias forem concluídas. Em caso de falha parcial, ele deve permanecer para permitir uma nova tentativa.

## 5. Flags recomendadas

| Flag | Comportamento |
| --- | --- |
| `--config`, `-c` | YAML usado na criação do cluster; obrigatório |
| `--dry-run` | Mostra o que seria removido sem alterar o sistema |
| `--yes`, `-y` | Confirma a remoção destrutiva sem prompt |
| `--keep-volumes` | Preserva os dados HDFS e Grafana |
| `--keep-images` | Preserva todas as imagens Docker |

Exemplos:

```bash
# Inspecionar o plano
./shanks prune --config ./config.yml --dry-run

# Limpeza total com confirmação interativa
./shanks prune --config ./config.yml

# Remover containers e configurações, preservando dados e imagens
./shanks prune --config ./config.yml --keep-volumes --keep-images

# Limpeza total não interativa
./shanks prune --config ./config.yml --yes
```

## 6. Comportamento em falhas

O `prune` deve ser **idempotente** e operar em modo **best effort**:

- recurso inexistente significa que aquela etapa já está limpa;
- falha em um datanode não impede a limpeza dos demais;
- erros concorrentes devem ser agregados, sem compartilhar variáveis mutáveis entre goroutines;
- cada erro deve identificar host, recurso e comando lógico executado;
- senhas, chaves privadas e conteúdo sensível nunca devem aparecer nos erros;
- uma segunda execução deve continuar a partir do manifesto restante.

Códigos de saída propostos:

- `0`: limpeza completa;
- `1`: limpeza parcial, com um ou mais recursos pendentes;
- `2`: configuração inválida, operação cancelada ou falha antes do início da limpeza.

## 7. Estrutura sugerida no código

```text
cli/backend/cmd/prune.go
cli/backend/usecase/prune_cluster.go
cli/backend/usecase/cleanup_hosts.go
cli/backend/usecase/cleanup_env.go
cli/backend/usecase/cluster_state.go
```

Fluxo principal sugerido:

```go
func PruneCluster(config models.Config, options PruneOptions) error {
    // 1. Carregar manifesto e inventariar recursos.
    // 2. Exibir plano ou confirmar operação.
    // 3. Parar daemons Hadoop.
    // 4. Derrubar Compose nos workers e no master.
    // 5. Remover apenas imagens pertencentes à execução.
    // 6. Restaurar/remover .env.
    // 7. Restaurar/remover blocos do /etc/hosts.
    // 8. Reverter known_hosts quando houver registro seguro.
    // 9. Verificar e agregar erros.
    // 10. Remover manifesto somente após sucesso total.
}
```

## 8. Critérios de aceite

O comando estará completo quando:

1. `prune --dry-run` não fizer nenhuma alteração.
2. `prune` exigir confirmação antes de apagar volumes.
3. Todos os containers de master e workers forem removidos.
4. Volumes forem removidos por padrão e preservados com `--keep-volumes`.
5. Imagens preexistentes nunca forem removidas sem prova de propriedade.
6. O bloco S.H.A.N.K.S for removido de todos os `/etc/hosts` sem afetar outras linhas.
7. Arquivos `.env` anteriores forem restaurados e arquivos criados pelo `start` forem removidos.
8. Uma falha em um worker não impedir a limpeza dos outros hosts.
9. Reexecutar o comando após sucesso não produzir erro nem remover recursos externos.
10. Uma execução parcial puder ser retomada usando o manifesto.
11. Nenhuma senha, chave ou segredo aparecer nos logs.
12. Testes cobrirem cluster vazio, um worker, múltiplos workers, recursos ausentes, falha SSH e falha parcial.

## 9. Limitações atuais que devem ser corrigidas antes do `prune`

1. O `start` não possui manifesto de estado e não sabe quais imagens, volumes e arquivos já existiam.
2. A escrita de `.env` usa `os.O_CREATE|os.O_WRONLY`, sem `os.O_TRUNC`; um arquivo anterior maior pode deixar conteúdo residual.
3. O `.env` preexistente não recebe backup antes de ser alterado.
4. O uso de `scp ... accept-new` pode alterar `known_hosts` sem registrar a mudança.
5. A senha de `sudo` é interpolada dentro do comando de shell usado para editar `/etc/hosts`; isso deve ser substituído por uma forma que não exponha a senha no comando ou em mensagens de erro.
6. Os caminhos remotos assumem que o repositório está sempre em `$HOME/S.H.A.N.K.S`.
7. O `stop` atual apenas executa `docker compose down`; ele não reverte arquivos, imagens nem volumes e não substitui o `prune`.
8. A montagem da lista `yarnWorkers` usa um slice com tamanho inicial `len(datanodes)-1` e depois faz `append`; com mais de um datanode isso cria entradas vazias, e sem datanodes pode causar falha. Isso aumenta a possibilidade de um `start` parcialmente aplicado.

Essas limitações não impedem uma limpeza forçada, mas impedem garantir que o sistema volte exatamente ao estado anterior sem risco de remover dados que não pertencem ao projeto.
