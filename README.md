# watch-gateway

给 Apple Watch（或任意 HTTP 客户端）查公交到站的小网关。线路写在配置里，不写死在代码里。

推荐用法：**本网关负责查车，MCP 负责让 AI 助手调用它。** 手表快捷指令、Shortcuts、Cursor / Claude 都可以打同一套接口。

## 它做什么

- `GET /watch/bus/q/{hash}`：公开查询，返回 JSON，自带简单限流
- `GET /watch/bus/home`、`/watch/bus/work`：旧接口，需要 `Authorization`
- 可选 Bark 推送（仅 legacy 接口）

仓库里的 `config.example.yaml` 是虚构示例。真实站点、坐标、token 请放在本地 `config.yaml`，该文件不会进 Git。

## 5 分钟跑起来

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml：换成你的线路 / 站点 / hash / auth_token
go run . -config config.yaml
```

健康检查：

```bash
curl -s http://127.0.0.1:28085/health
```

查一条线路（把 `demo-home` 换成你自己的 hash）：

```bash
curl -s http://127.0.0.1:28085/watch/bus/q/demo-home
```

## 搭配 MCP（推荐）

MCP（Model Context Protocol）让 Cursor、Claude Desktop 等助手直接调用你的本地网关，不用每次手敲 curl。

1. 先在本机把 `watch-gateway` 跑起来。
2. 安装示例 MCP 服务：

```bash
cd examples/mcp
npm install
```

3. 在 Cursor 的 MCP 配置里加入（路径按本机修改）：

```json
{
  "mcpServers": {
    "watch-gateway": {
      "command": "node",
      "args": ["/ABS/PATH/watch-gateway/examples/mcp/server.mjs"],
      "env": {
        "WATCH_GATEWAY_BASE": "http://127.0.0.1:28085",
        "WATCH_GATEWAY_HASHES": "home=demo-home,work=demo-work"
      }
    }
  }
}
```

4. 重启 MCP 后，对助手说：「查一下回家那班公交还有几站。」

`WATCH_GATEWAY_HASHES` 把口语别名映射到你的 hash，别名和真实站点只留在你自己机器上。

## Apple Watch / 快捷指令

在快捷指令里发 GET 请求即可，例如：

`http://你的服务器:28085/watch/bus/q/你的hash`

不要把带真实 hash 的完整 URL 贴进公开 Issue 或截图。

## Docker

```bash
docker build -t watch-gateway .
docker run --rm -p 28085:28085 \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  watch-gateway
```

## 隐私

开源版本已去掉私人上下班站点、经纬度和鉴权 token。若你 fork 后本地填了真实配置，请确认 `config.yaml` 仍在 `.gitignore` 中。

## License

MIT
