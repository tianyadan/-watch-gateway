#!/usr/bin/env node
/**
 * 最小 MCP 服务：把口语别名转成 watch-gateway 的 hash 查询。
 * 环境变量：
 *   WATCH_GATEWAY_BASE   默认 http://127.0.0.1:28085
 *   WATCH_GATEWAY_HASHES 例如 home=demo-home,work=demo-work
 */
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js'

const base = (process.env.WATCH_GATEWAY_BASE || 'http://127.0.0.1:28085').replace(/\/$/, '')
const aliasMap = Object.fromEntries(
  String(process.env.WATCH_GATEWAY_HASHES || '')
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => {
      const i = item.indexOf('=')
      if (i <= 0) return [item, item]
      return [item.slice(0, i).trim(), item.slice(i + 1).trim()]
    }),
)

async function queryBus(hash) {
  const res = await fetch(`${base}/watch/bus/q/${encodeURIComponent(hash)}`)
  const text = await res.text()
  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${text}`)
  }
  return text
}

const server = new Server(
  { name: 'watch-gateway', version: '1.0.0' },
  { capabilities: { tools: {} } },
)

server.setRequestHandler(ListToolsRequestSchema, async () => ({
  tools: [
    {
      name: 'query_bus',
      description:
        'Query upcoming buses via local watch-gateway. Pass an alias from WATCH_GATEWAY_HASHES or a raw hash.',
      inputSchema: {
        type: 'object',
        properties: {
          route: {
            type: 'string',
            description: 'Alias such as home/work, or the public hash itself',
          },
        },
        required: ['route'],
      },
    },
  ],
}))

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  if (request.params.name !== 'query_bus') {
    throw new Error(`Unknown tool: ${request.params.name}`)
  }
  const route = String(request.params.arguments?.route || '').trim()
  if (!route) {
    throw new Error('route is required')
  }
  const hash = aliasMap[route] || route
  const body = await queryBus(hash)
  return {
    content: [{ type: 'text', text: body }],
  }
})

const transport = new StdioServerTransport()
await server.connect(transport)
