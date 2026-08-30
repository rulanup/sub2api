import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('risk control locale copy', () => {
  it('describes worker runtime as audit and pre-block record processing', () => {
    expect(zh.admin.riskControl.workerStatusHint).toContain('前置拦截记录任务')
    expect(zh.admin.riskControl.workerStatusHint).not.toContain('异步观察任务')
    expect(en.admin.riskControl.workerStatusHint).toContain('pre-block record tasks')
    expect(en.admin.riskControl.workerStatusHint).not.toContain('observation tasks')
  })

  it('keeps pre-block audit key summary aware of async worker load', () => {
    expect(zh.admin.riskControl.preBlockAPIKeyLoadSummary).toContain('worker：{workerActive} / {workerTotal}')
    expect(en.admin.riskControl.preBlockAPIKeyLoadSummary).toContain('worker: {workerActive} / {workerTotal}')
  })

  it('does not describe pre-block audit key polling as bypassing the worker pool', () => {
    expect(zh.admin.riskControl.preBlockAPIKeyLoadHint).toBe('同步前置拦截直接轮询可用审核 Key。')
    expect(zh.admin.riskControl.preBlockAPIKeyLoadHint).not.toContain('Worker 池')
    expect(en.admin.riskControl.preBlockAPIKeyLoadHint).not.toContain('worker pool')
  })

  it('keeps Aliyun Guardrails configuration copy complete in both locales', () => {
    for (const key of [
      'protocolAliyunGuardrails',
      'aliyunProtocolHint',
      'aliyunRegion',
      'aliyunService',
      'aliyunEndpoint',
      'aliyunCredentials',
      'aliyunAccessKeyId',
      'aliyunAccessKeySecret',
      'clearAliyunCredentials',
      'aliyunTextOnlyTitle',
      'aliyunTextOnlyHint',
      'testAliyunConnection',
    ] as const) {
      expect(zh.admin.riskControl[key]).toBeTruthy()
      expect(en.admin.riskControl[key]).toBeTruthy()
    }
    expect(zh.admin.riskControl.aliyunTextOnlyHint).toContain('图片')
    expect(zh.admin.riskControl.aliyunTextOnlyHint).toContain('错误')
    expect(en.admin.riskControl.aliyunTextOnlyHint).toContain('Images')
    expect(en.admin.riskControl.aliyunTextOnlyHint).toContain('error')
  })

  it('keeps Qwen3Guard protocol and endpoint diagnostics localized', () => {
    for (const key of [
      'protocol',
      'protocolQwen3Guard',
      'controversialAction',
      'qwenImageUnsupportedHint',
      'auditEndpointUnsupported',
    ] as const) {
      expect(zh.admin.riskControl[key]).toBeTruthy()
      expect(en.admin.riskControl[key]).toBeTruthy()
    }
  })
})
