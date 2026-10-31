import type {
  ApiError,
  DeveloperDocument,
  DeveloperEndpoint,
  LocalizedText,
  TranslationRecord,
} from './types'

import gettingStartedZh from './content/zh/getting-started.md?raw'
import gettingStartedEn from './content/en/getting-started.md?raw'
import authenticationZh from './content/zh/authentication.md?raw'
import authenticationEn from './content/en/authentication.md?raw'
import quickStartZh from './content/zh/quick-start.md?raw'
import quickStartEn from './content/en/quick-start.md?raw'
import streamingZh from './content/zh/streaming.md?raw'
import streamingEn from './content/en/streaming.md?raw'
import errorsZh from './content/zh/errors-rate-limits.md?raw'
import errorsEn from './content/en/errors-rate-limits.md?raw'
import troubleshootingZh from './content/zh/troubleshooting.md?raw'
import troubleshootingEn from './content/en/troubleshooting.md?raw'
import configurationZh from './content/zh/configuration.md?raw'
import configurationEn from './content/en/configuration.md?raw'
import changelogZh from './content/zh/changelog.md?raw'
import changelogEn from './content/en/changelog.md?raw'

const CURRENT: TranslationRecord = {
  status: 'current',
  sourceHash: '2026-10-31-v1',
  updatedAt: '2026-10-31',
}

const MISSING_JA: TranslationRecord = {
  status: 'missing',
  sourceHash: '2026-10-31-v1',
  updatedAt: '2026-10-31',
}

const API_AUTH_ERRORS: ApiError[] = [
  {
    status: 401,
    code: 'invalid_api_key',
    description: {
      zh: 'API Key 缺失、已失效或没有权限。',
      en: 'The API key is missing, invalid, or not permitted for this request.',
    },
  },
  {
    status: 429,
    code: 'rate_limit_exceeded',
    description: {
      zh: '请求超过当前 Key 或分组的速率限制。',
      en: 'The request exceeded the rate limit for the current key or group.',
    },
  },
]

const API_KEY_HEADER = {
  name: 'Authorization',
  location: 'header' as const,
  type: 'string',
  required: true,
  description: {
    zh: '使用 Bearer 方案传入 Sub2API API Key。',
    en: 'Pass a Sub2API API key using the Bearer scheme.',
  },
}

function doc(
  id: string,
  slug: string,
  category: DeveloperDocument['category'],
  title: LocalizedText,
  description: LocalizedText,
  content: LocalizedText,
  tags: string[],
  relatedDocIds: string[],
): DeveloperDocument {
  return {
    id,
    slug,
    category,
    title,
    description,
    content,
    version: 'v1',
    updatedAt: '2026-10-31',
    tags,
    relatedDocIds,
    translations: {
      zh: CURRENT,
      en: CURRENT,
      ja: MISSING_JA,
    },
  }
}

export const developerDocuments: DeveloperDocument[] = [
  doc(
    'getting-started',
    'getting-started',
    'start',
    { zh: '快速开始', en: 'Getting started' },
    { zh: '从创建 API Key 到发出第一条请求。', en: 'Go from creating an API key to your first request.' },
    { zh: gettingStartedZh, en: gettingStartedEn },
    ['getting-started', 'quick-start', 'api-key'],
    ['authentication', 'quick-start', 'errors-rate-limits'],
  ),
  doc(
    'authentication',
    'authentication',
    'start',
    { zh: '认证', en: 'Authentication' },
    { zh: '使用 API Key 安全地调用网关。', en: 'Call the gateway securely with an API key.' },
    { zh: authenticationZh, en: authenticationEn },
    ['authentication', 'api-key', 'security', 'bearer'],
    ['getting-started', 'troubleshooting', 'errors-rate-limits'],
  ),
  doc(
    'quick-start',
    'quick-start',
    'start',
    { zh: '发送请求', en: 'Make a request' },
    { zh: '使用 cURL、Python 或 JavaScript 调用真实网关。', en: 'Call the real gateway with cURL, Python, or JavaScript.' },
    { zh: quickStartZh, en: quickStartEn },
    ['quick-start', 'curl', 'python', 'javascript'],
    ['getting-started', 'authentication', 'streaming'],
  ),
  doc(
    'streaming',
    'streaming',
    'guides',
    { zh: '流式响应', en: 'Streaming responses' },
    { zh: '处理 Server-Sent Events 与中断。', en: 'Handle Server-Sent Events and cancellation.' },
    { zh: streamingZh, en: streamingEn },
    ['streaming', 'sse', 'stream'],
    ['quick-start', 'troubleshooting', 'errors-rate-limits'],
  ),
  doc(
    'errors-rate-limits',
    'errors-rate-limits',
    'operations',
    { zh: '错误与限流', en: 'Errors and rate limits' },
    { zh: '理解常见 HTTP 状态、429 和排查顺序。', en: 'Understand common HTTP statuses, 429 responses, and diagnosis order.' },
    { zh: errorsZh, en: errorsEn },
    ['errors', '429', 'rate-limit', '401', 'troubleshooting'],
    ['troubleshooting', 'authentication', 'debug-center'],
  ),
  doc(
    'troubleshooting',
    'troubleshooting',
    'operations',
    { zh: '故障排查', en: 'Troubleshooting' },
    { zh: '定位认证、CORS、超时和流式错误。', en: 'Diagnose authentication, CORS, timeout, and streaming failures.' },
    { zh: troubleshootingZh, en: troubleshootingEn },
    ['troubleshooting', 'cors', 'timeout', 'network'],
    ['errors-rate-limits', 'streaming', 'authentication'],
  ),
  doc(
    'configuration',
    'configuration',
    'guides',
    { zh: '客户端配置', en: 'Client configuration' },
    { zh: '配置 Base URL、协议兼容层和本地环境。', en: 'Configure base URLs, protocol compatibility, and local environments.' },
    { zh: configurationZh, en: configurationEn },
    ['configuration', 'base-url', 'environment'],
    ['getting-started', 'troubleshooting'],
  ),
  doc(
    'changelog',
    'changelog',
    'reference',
    { zh: '更新记录', en: 'Changelog' },
    { zh: 'Developer Portal 与 API 文档变化。', en: 'Changes to the Developer Portal and API documentation.' },
    { zh: changelogZh, en: changelogEn },
    ['changelog', 'release'],
    ['getting-started', 'errors-rate-limits'],
  ),
]

export const developerEndpoints: DeveloperEndpoint[] = [
  {
    id: 'models-list',
    category: 'models',
    method: 'GET',
    path: '/v1/models',
    backendGroup: '/v1',
    backendPath: '/models',
    summary: { zh: '列出可用模型', en: 'List available models' },
    description: {
      zh: '返回当前 API Key 和分组可发现的模型。',
      en: 'Returns models discoverable by the current API key and group.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'GatewayHandler.Models',
    parameters: [API_KEY_HEADER],
    requestExample: null,
    responseExample: {
      object: 'list',
      data: [{ id: 'example-model', object: 'model', owned_by: 'gateway' }],
    },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '限流由 Key、分组和实例策略共同决定。',
      en: 'Rate limits depend on the key, group, and instance policy.',
    },
    relatedDocIds: ['getting-started', 'configuration'],
  },
  {
    id: 'models-get',
    category: 'models',
    method: 'GET',
    path: '/v1/models/{model}',
    backendGroup: '/v1',
    backendPath: '/models/:model',
    summary: { zh: '获取模型信息', en: 'Get model details' },
    description: {
      zh: '查询当前 Key 可发现的单个模型。',
      en: 'Returns one model discoverable by the current key.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'GatewayHandler.Models',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'model',
        location: 'path',
        type: 'string',
        required: true,
        description: { zh: '模型标识。', en: 'Model identifier.' },
      },
    ],
    requestExample: null,
    responseExample: { id: 'example-model', object: 'model', owned_by: 'gateway' },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '限流由 Key、分组和实例策略共同决定。',
      en: 'Rate limits depend on the key, group, and instance policy.',
    },
    relatedDocIds: ['configuration'],
  },
  {
    id: 'chat-completions',
    category: 'chat',
    method: 'POST',
    path: '/v1/chat/completions',
    backendGroup: '/v1',
    backendPath: '/chat/completions',
    summary: { zh: '创建对话补全', en: 'Create chat completion' },
    description: {
      zh: '使用 OpenAI Chat Completions 兼容请求创建对话补全，支持流式响应。',
      en: 'Creates a chat completion with an OpenAI Chat Completions-compatible request and optional streaming.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'OpenAIGatewayHandler.ChatCompletions',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'model',
        location: 'body',
        type: 'string',
        required: true,
        description: { zh: '当前 Key 可用的模型标识。', en: 'A model identifier available to the current key.' },
      },
      {
        name: 'messages',
        location: 'body',
        type: 'array<Message>',
        required: true,
        description: { zh: '对话消息数组。', en: 'Conversation message array.' },
      },
      {
        name: 'stream',
        location: 'body',
        type: 'boolean',
        required: false,
        description: { zh: '启用 SSE 流式响应。', en: 'Enables an SSE streaming response.' },
      },
    ],
    requestExample: {
      model: 'your-model',
      messages: [{ role: 'user', content: 'Hello' }],
      stream: false,
    },
    responseExample: {
      id: 'chatcmpl-example',
      object: 'chat.completion',
      choices: [{ index: 0, message: { role: 'assistant', content: 'Hello!' } }],
    },
    errors: [
      ...API_AUTH_ERRORS,
      {
        status: 400,
        code: 'invalid_request_error',
        description: { zh: '请求体或参数无效。', en: 'The request body or parameters are invalid.' },
      },
    ],
    rateLimit: {
      zh: '按 Key、分组、渠道和上游策略限流。',
      en: 'Limits apply by key, group, channel, and upstream policy.',
    },
    relatedDocIds: ['quick-start', 'streaming', 'errors-rate-limits'],
  },
  {
    id: 'messages-create',
    category: 'chat',
    method: 'POST',
    path: '/v1/messages',
    backendGroup: '/v1',
    backendPath: '/messages',
    summary: { zh: '创建消息', en: 'Create message' },
    description: {
      zh: '使用 Anthropic Messages 兼容协议创建消息。',
      en: 'Creates a message using the Anthropic Messages-compatible protocol.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'GatewayHandler.Messages',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'model',
        location: 'body',
        type: 'string',
        required: true,
        description: { zh: '当前 Key 可用的模型标识。', en: 'A model identifier available to the current key.' },
      },
      {
        name: 'messages',
        location: 'body',
        type: 'array<Message>',
        required: true,
        description: { zh: '消息数组。', en: 'Message array.' },
      },
      {
        name: 'max_tokens',
        location: 'body',
        type: 'integer',
        required: true,
        description: { zh: '本次响应的最大 token 数。', en: 'Maximum number of response tokens.' },
      },
    ],
    requestExample: {
      model: 'your-model',
      max_tokens: 256,
      messages: [{ role: 'user', content: 'Hello' }],
    },
    responseExample: {
      id: 'msg_example',
      type: 'message',
      role: 'assistant',
      content: [{ type: 'text', text: 'Hello!' }],
    },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key、分组、渠道和上游策略限流。',
      en: 'Limits apply by key, group, channel, and upstream policy.',
    },
    relatedDocIds: ['quick-start', 'errors-rate-limits'],
  },
  {
    id: 'responses-create',
    category: 'responses',
    method: 'POST',
    path: '/v1/responses',
    backendGroup: '/v1',
    backendPath: '/responses',
    summary: { zh: '创建响应', en: 'Create response' },
    description: {
      zh: '使用 OpenAI Responses 兼容请求创建响应，可返回普通 JSON 或流式事件。',
      en: 'Creates a response with an OpenAI Responses-compatible request, returning JSON or streaming events.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'OpenAIGatewayHandler.Responses',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'model',
        location: 'body',
        type: 'string',
        required: true,
        description: { zh: '当前 Key 可用的模型标识。', en: 'A model identifier available to the current key.' },
      },
      {
        name: 'input',
        location: 'body',
        type: 'string | array',
        required: true,
        description: { zh: 'Responses 输入内容。', en: 'Input for the Responses API.' },
      },
      {
        name: 'stream',
        location: 'body',
        type: 'boolean',
        required: false,
        description: { zh: '启用 SSE 流式响应。', en: 'Enables an SSE streaming response.' },
      },
    ],
    requestExample: { model: 'your-model', input: 'Hello', stream: false },
    responseExample: {
      id: 'resp_example',
      object: 'response',
      status: 'completed',
      output_text: 'Hello!',
    },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key、分组、渠道和上游策略限流。',
      en: 'Limits apply by key, group, channel, and upstream policy.',
    },
    relatedDocIds: ['quick-start', 'streaming'],
  },
  {
    id: 'messages-count-tokens',
    category: 'responses',
    method: 'POST',
    path: '/v1/messages/count_tokens',
    backendGroup: '/v1',
    backendPath: '/messages/count_tokens',
    summary: { zh: '统计消息 Token', en: 'Count message tokens' },
    description: {
      zh: '估算或转发消息请求的 token 数，具体策略取决于分组平台。',
      en: 'Estimates or forwards message token counts depending on the group platform.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'countTokensHandler',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'model',
        location: 'body',
        type: 'string',
        required: true,
        description: { zh: '需要计数的模型。', en: 'Model to count against.' },
      },
      {
        name: 'messages',
        location: 'body',
        type: 'array<Message>',
        required: true,
        description: { zh: '需要计数的消息。', en: 'Messages to count.' },
      },
    ],
    requestExample: {
      model: 'your-model',
      messages: [{ role: 'user', content: 'Hello' }],
    },
    responseExample: { input_tokens: 8 },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key、分组和实例策略限流。',
      en: 'Limits apply by key, group, and instance policy.',
    },
    relatedDocIds: ['configuration'],
  },
  {
    id: 'key-billing',
    category: 'billing',
    method: 'GET',
    path: '/v1/sub2api/billing',
    backendGroup: '/v1',
    backendPath: '/sub2api/billing',
    summary: { zh: '获取 Key 计费信息', en: 'Get key billing' },
    description: {
      zh: '返回当前 API Key 生效的计费倍率信息。',
      en: 'Returns billing multiplier information effective for the current API key.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'GatewayHandler.KeyBillingInfo',
    parameters: [API_KEY_HEADER],
    requestExample: null,
    responseExample: { rate_multiplier: 1, platform: 'openai' },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key 和实例策略限流。',
      en: 'Limits apply by key and instance policy.',
    },
    relatedDocIds: ['getting-started', 'errors-rate-limits'],
  },
  {
    id: 'usage',
    category: 'billing',
    method: 'GET',
    path: '/v1/usage',
    backendGroup: '/v1',
    backendPath: '/usage',
    summary: { zh: '获取 Key 用量', en: 'Get key usage' },
    description: {
      zh: '返回当前 API Key 的用量信息。字段由当前实例版本和账号类型决定。',
      en: 'Returns usage information for the current API key. Fields depend on the instance version and account type.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'GatewayHandler.Usage',
    parameters: [API_KEY_HEADER],
    requestExample: null,
    responseExample: { usage: { requests: 0, tokens: 0 } },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key 和实例策略限流。',
      en: 'Limits apply by key and instance policy.',
    },
    relatedDocIds: ['getting-started'],
  },
  {
    id: 'images-generations',
    category: 'images',
    method: 'POST',
    path: '/v1/images/generations',
    backendGroup: '/v1',
    backendPath: '/images/generations',
    summary: { zh: '生成图片', en: 'Generate image' },
    description: {
      zh: '为支持图片生成的平台创建图片生成请求。',
      en: 'Creates an image generation request for platforms that support image generation.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'OpenAIGatewayHandler.Images',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'model',
        location: 'body',
        type: 'string',
        required: true,
        description: { zh: '当前分组支持的图片模型。', en: 'An image model supported by the current group.' },
      },
      {
        name: 'prompt',
        location: 'body',
        type: 'string',
        required: true,
        description: { zh: '图片生成提示词。', en: 'Image generation prompt.' },
      },
    ],
    requestExample: { model: 'your-image-model', prompt: 'A minimal optical wave artwork' },
    responseExample: { created: 0, data: [{ url: 'https://example.com/image.png' }] },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '图片请求可能消耗更高配额，并受分组和上游限制。',
      en: 'Image requests may consume higher quotas and are subject to group and upstream limits.',
    },
    relatedDocIds: ['configuration', 'errors-rate-limits'],
  },
  {
    id: 'images-edits',
    category: 'images',
    method: 'POST',
    path: '/v1/images/edits',
    backendGroup: '/v1',
    backendPath: '/images/edits',
    summary: { zh: '编辑图片', en: 'Edit image' },
    description: {
      zh: '为支持图片编辑的平台提交编辑请求。请求编码由目标平台决定。',
      en: 'Submits an image edit request for supported platforms. Encoding depends on the target platform.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'OpenAIGatewayHandler.Images',
    parameters: [API_KEY_HEADER],
    requestExample: 'multipart/form-data or platform-specific JSON',
    responseExample: { created: 0, data: [{ url: 'https://example.com/edited.png' }] },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '图片请求可能消耗更高配额，并受分组和上游限制。',
      en: 'Image requests may consume higher quotas and are subject to group and upstream limits.',
    },
    relatedDocIds: ['configuration'],
  },
  {
    id: 'images-task',
    category: 'images',
    method: 'GET',
    path: '/v1/images/tasks/{task_id}',
    backendGroup: '/v1',
    backendPath: '/images/tasks/:task_id',
    summary: { zh: '查询异步图片任务', en: 'Get async image task' },
    description: {
      zh: '查询异步图片任务状态。',
      en: 'Returns the status of an asynchronous image task.',
    },
    auth: 'api-key',
    version: 'v1',
    sourceHandler: 'AsyncImageHandler.Get',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'task_id',
        location: 'path',
        type: 'string',
        required: true,
        description: { zh: '异步图片任务 ID。', en: 'Asynchronous image task ID.' },
      },
    ],
    requestExample: null,
    responseExample: { task_id: 'task-example', status: 'completed' },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key、分组和实例策略限流。',
      en: 'Limits apply by key, group, and instance policy.',
    },
    relatedDocIds: ['configuration'],
  },
  {
    id: 'gemini-models-list',
    category: 'gemini',
    method: 'GET',
    path: '/v1beta/models',
    backendGroup: '/v1beta',
    backendPath: '/models',
    summary: { zh: '列出 Gemini 模型', en: 'List Gemini models' },
    description: {
      zh: '返回 Gemini 原生兼容层可发现的模型。',
      en: 'Returns models discoverable through the native Gemini compatibility layer.',
    },
    auth: 'api-key',
    version: 'v1beta',
    sourceHandler: 'GatewayHandler.GeminiV1BetaListModels',
    parameters: [API_KEY_HEADER],
    requestExample: null,
    responseExample: { models: [{ name: 'models/example-model' }] },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key、分组和 Gemini 渠道策略限流。',
      en: 'Limits apply by key, group, and Gemini channel policy.',
    },
    relatedDocIds: ['configuration'],
  },
  {
    id: 'gemini-model-action',
    category: 'gemini',
    method: 'POST',
    path: '/v1beta/models/{model}:{action}',
    backendGroup: '/v1beta',
    backendPath: '/models/*modelAction',
    summary: { zh: '调用 Gemini 模型动作', en: 'Call Gemini model action' },
    description: {
      zh: '调用 Gemini 原生模型的生成或计数动作。实际可用动作由当前分组和平台决定。',
      en: 'Calls a native Gemini model generation or counting action. Supported actions depend on the current group and platform.',
    },
    auth: 'api-key',
    version: 'v1beta',
    sourceHandler: 'GatewayHandler.GeminiV1BetaModels',
    parameters: [
      API_KEY_HEADER,
      {
        name: 'contents',
        location: 'body',
        type: 'array',
        required: true,
        description: { zh: 'Gemini 内容数组。', en: 'Gemini content array.' },
      },
    ],
    requestExample: {
      contents: [{ role: 'user', parts: [{ text: 'Hello' }] }],
    },
    responseExample: {
      candidates: [{ content: { role: 'model', parts: [{ text: 'Hello!' }] } }],
    },
    errors: API_AUTH_ERRORS,
    rateLimit: {
      zh: '按 Key、分组和 Gemini 渠道策略限流。',
      en: 'Limits apply by key, group, and Gemini channel policy.',
    },
    relatedDocIds: ['configuration', 'errors-rate-limits'],
  },
]

export function getDeveloperDocument(slug: string): DeveloperDocument | undefined {
  return developerDocuments.find((document) => document.slug === slug || document.id === slug)
}

export function getDeveloperEndpoint(id: string): DeveloperEndpoint | undefined {
  return developerEndpoints.find((endpoint) => endpoint.id === id)
}

export const developerCategories = [
  { id: 'start', zh: '开始使用', en: 'Start' },
  { id: 'guides', zh: '指南', en: 'Guides' },
  { id: 'reference', zh: '参考', en: 'Reference' },
  { id: 'operations', zh: '运维与排查', en: 'Operations' },
] as const
