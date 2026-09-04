# TraitAPI - Multichat

App nativo em Go (sem runtime externo) que unifica o chat da **Twitch**, **YouTube** e **Kick** num overlay unico pra transmissao, com uma janela nativa de leitura, icone de bandeja com alerta e configuracao 100% local.

## Recursos

- **Overlay para OBS** (`/overlay`, fundo transparente) e **monitor pelo navegador** (`/monitor`, fundo escuro, historico com scroll) servidos pelo proprio app.
- **Janela nativa** (Fyne): tela de login, tela de configuracao (agrupada por rede, com cor suave da marca) e abas — Chat geral, uma por rede configurada, e Log.
- **Icone na bandeja** com ponto vermelho de "nao lido" e notificacao com temporizador (evita alerta repetido a cada mensagem).
- **Twitch**: IRC anonimo (sem precisar de token).
- **YouTube**: YouTube Data API v3, com descoberta automatica da live e polling configuravel.
- **Kick**: websocket publico (Pusher) usado pelo proprio site da Kick, com descoberta automatica do chatroom ou ID manual.
- **Porta com fallback automatico**: se a porta preferida estiver ocupada, tenta as seguintes e mostra a URL real na janela.
- **Configuracao local**: `config.json` criptografado por inteiro com DPAPI do Windows (so abre na mesma conta que salvou), guardado em `%AppData%\api-chat\<hash do caminho do exe>\` — cada copia do executavel fica isolada automaticamente, mesmo rodando na mesma maquina. A senha de login e guardada como hash salgado (nunca em texto puro, nem criptografado reversivel).

## Rodando

1. Compile (`go build -ldflags="-H=windowsgui" -o api-chat.exe .`) e execute o `.exe`.
2. Faca login (usuario/senha padrao: `admin` / `admin`, salvo com hash local).
3. Na primeira vez, preencha a tela de configuracao (canal da Twitch, API Key + Channel ID do YouTube, canal da Kick). Se existir um `.env` na mesma pasta (ver `.env.example`), os valores sao usados como sugestao inicial.
4. Copie a URL da overlay (barra no topo das abas) pro Browser Source do OBS.

## Build por cliente ("sob encomenda")

Cada build pode ter login proprio (a config em si ja fica isolada por natureza, ao lado de cada `.exe`):

```sh
go build -ldflags "-H=windowsgui \
  -X api-chat/internal/config.DefaultAuthUser=<usuario> \
  -X api-chat/internal/config.DefaultAuthPass=<senha>" \
  -o dist/<cliente>/api-chat.exe .
```

Sem um `.env` na pasta de saida, a tela de configuracao comeca vazia.

## Estrutura

```text
main.go                    orquestra: config, janela Fyne, servicos de chat, servidor HTTP
internal/config/           config local (login + canais), import de .env como semente
internal/chatmsg/          tipo de mensagem unificado entre as redes
internal/twitch/           cliente IRC (Twitch)
internal/youtube/          polling da YouTube Data API v3
internal/kick/             cliente websocket (Kick)
internal/hub/              distribui mensagens pros clientes websocket (overlay/monitor)
internal/server/           servidor HTTP (overlay, monitor, websocket) com fallback de porta
internal/desktopui/        janela nativa: login, config, abas, bandeja e alerta
internal/logbuf/           buffer de log em memoria (aba "Log")
internal/secure/           criptografia local (DPAPI no Windows)
internal/assets/           icones/logo embutidos no binario
web/                       overlay.html/css e monitor.html/css
```

## Logs

Gravados em `api-chat.log` ao lado do executavel — temporario: comeca vazio a cada execucao e e apagado ao fechar o app. Tambem aparecem em tempo real na aba "Log" da janela.
