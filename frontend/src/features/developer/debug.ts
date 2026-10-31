import { sanitizeBody, sanitizeHeaders } from './security'
import type { DebugDiagnosis, DebugInput, LocalizedText } from './types'

function text(zh: string, en: string): LocalizedText {
  return { zh, en }
}

function diagnosis(
  code: DebugDiagnosis['code'],
  title: LocalizedText,
  summary: LocalizedText,
  confidence: DebugDiagnosis['confidence'],
  steps: LocalizedText[],
  relatedDocIds: string[],
): DebugDiagnosis {
  return { code, title, summary, confidence, steps, relatedDocIds }
}

export function classifyRequestFailure(input: DebugInput): DebugDiagnosis {
  const status = input.status
  if (status === 429) {
    return diagnosis(
      'rate_limit',
      text('触发速率限制', 'Rate limit reached'),
      text('服务端拒绝了当前请求频率或并发。', 'The server rejected the current request rate or concurrency.'),
      'high',
      [
        text('降低并发并等待 Retry-After。', 'Reduce concurrency and honor Retry-After.'),
        text('检查 Key、分组和渠道限流配置。', 'Check key, group, and channel rate limits.'),
      ],
      ['errors-rate-limits'],
    )
  }

  if (status === 401) {
    return diagnosis(
      'authentication',
      text('认证失败', 'Authentication failed'),
      text('API Key 缺失、失效或格式不正确。', 'The API key is missing, invalid, or malformed.'),
      'high',
      [
        text('确认 Authorization 使用 Bearer 方案。', 'Confirm the Authorization header uses Bearer.'),
        text('确认 Key 没有过期或撤销。', 'Confirm the key has not expired or been revoked.'),
      ],
      ['authentication', 'errors-rate-limits'],
    )
  }

  if (status === 403) {
    return diagnosis(
      'permission',
      text('请求没有权限', 'Permission denied'),
      text('Key 所属分组或账号可能不允许当前模型或路径。', 'The key group or account may not permit this model or path.'),
      'high',
      [
        text('检查模型白名单和分组权限。', 'Check model allowlist and group permissions.'),
        text('确认账号与渠道状态可用。', 'Confirm account and channel state.'),
      ],
      ['errors-rate-limits', 'configuration'],
    )
  }

  if (status === 404) {
    return diagnosis(
      'not_found',
      text('路径或资源不存在', 'Path or resource not found'),
      text('请求路径、版本或模型名称可能不可用。', 'The path, version, or model name may be unavailable.'),
      'high',
      [
        text('对照 API Reference 检查路径和版本。', 'Compare the path and version with API Reference.'),
        text('确认模型在当前 Key 的分组中可用。', 'Confirm the model is available to the current key group.'),
      ],
      ['errors-rate-limits', 'configuration'],
    )
  }

  if (status === 400) {
    return diagnosis(
      'invalid_request',
      text('请求格式无效', 'Invalid request'),
      text('请求体、参数或内容类型未通过校验。', 'The request body, parameters, or content type failed validation.'),
      'high',
      [
        text('验证 JSON 和必填字段。', 'Validate JSON and required fields.'),
        text('检查 Content-Type 是否匹配请求体。', 'Check that Content-Type matches the request body.'),
      ],
      ['errors-rate-limits', 'quick-start'],
    )
  }

  if (typeof status === 'number' && status >= 500) {
    return diagnosis(
      'upstream_error',
      text('上游或网关异常', 'Upstream or gateway failure'),
      text('服务端暂时无法完成请求；具体原因需要结合 Request ID 判断。', 'The server could not complete the request; use the Request ID to investigate.'),
      'medium',
      [
        text('保留 Request ID 和脱敏响应。', 'Keep the Request ID and redacted response.'),
        text('使用退避策略稍后重试。', 'Retry later with backoff.'),
      ],
      ['errors-rate-limits', 'troubleshooting'],
    )
  }

  const error = input.error
  const errorName = error && typeof error === 'object' && 'name' in error
    ? String((error as { name?: unknown }).name)
    : ''
  const message = error instanceof Error ? error.message : String(error ?? '')

  if (errorName === 'AbortError' || /timeout|timed out/i.test(message)) {
    return diagnosis(
      'timeout',
      text('请求超时或已取消', 'Request timed out or was cancelled'),
      text('请求在完成前被中止。', 'The request was aborted before it completed.'),
      'high',
      [
        text('检查网络和上游耗时。', 'Check network and upstream latency.'),
        text('确认是否是用户主动停止。', 'Confirm whether the request was stopped by the user.'),
      ],
      ['troubleshooting', 'streaming'],
    )
  }

  if (error instanceof TypeError || /failed to fetch|networkerror|cors/i.test(message)) {
    return diagnosis(
      'network_or_cors',
      text('网络连接或 CORS 失败', 'Network or CORS failure'),
      text('浏览器无法读取响应；可能是网络、TLS 或 CORS 限制。', 'The browser could not read the response; network, TLS, or CORS may be involved.'),
      'medium',
      [
        text('查看浏览器 Network 面板。', 'Inspect the browser Network panel.'),
        text('确认目标地址允许当前来源跨域。', 'Confirm the target permits the current origin.'),
        text('确认 HTTPS、证书和网络连通性。', 'Confirm HTTPS, certificates, and connectivity.'),
      ],
      ['troubleshooting', 'authentication'],
    )
  }

  return diagnosis(
    'unknown',
    text('尚未识别的错误', 'Unclassified error'),
    text('当前信息不足以确定原因。', 'The available information is insufficient to determine a cause.'),
    'low',
    [
      text('查看脱敏调试报告。', 'Inspect the redacted debug report.'),
      text('补充 Request ID 和响应状态。', 'Add the Request ID and response status.'),
    ],
    ['troubleshooting'],
  )
}

export interface DebugReportInput {
  requestId: string
  method: string
  url: string
  status?: number
  durationMs?: number
  requestHeaders?: Record<string, string>
  requestBody?: string
  responseHeaders?: Record<string, string>
  responseBody?: string
}

export function createRedactedDebugReport(input: DebugReportInput): string {
  const diagnosisResult = classifyRequestFailure({ status: input.status })
  const report = {
    requestId: input.requestId,
    method: input.method,
    url: input.url,
    status: input.status ?? null,
    durationMs: input.durationMs ?? null,
    diagnosis: {
      code: diagnosisResult.code,
      confidence: diagnosisResult.confidence,
      relatedDocIds: diagnosisResult.relatedDocIds,
    },
    requestHeaders: sanitizeHeaders(input.requestHeaders ?? {}),
    requestBody: sanitizeBody(input.requestBody ?? ''),
    responseHeaders: sanitizeHeaders(input.responseHeaders ?? {}),
    responseBody: sanitizeBody(input.responseBody ?? ''),
  }

  return [
    '# API Debug Report',
    '',
    '~~~json',
    JSON.stringify(report, null, 2),
    '~~~',
  ].join('\n')
}
