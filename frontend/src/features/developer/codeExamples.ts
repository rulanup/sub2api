import type { DeveloperEndpoint } from './types'

export type CodeLanguage = 'curl' | 'python' | 'javascript' | 'typescript' | 'go'

const BASE_URL = 'https://your-api-host'

function requestBody(endpoint: DeveloperEndpoint): string {
  if (endpoint.requestExample === null) return ''
  if (typeof endpoint.requestExample === 'string') return endpoint.requestExample
  return JSON.stringify(endpoint.requestExample, null, 2)
}

export function generateCodeExample(endpoint: DeveloperEndpoint, language: CodeLanguage): string {
  const url = BASE_URL + endpoint.path
  const body = requestBody(endpoint)
  const method = endpoint.method

  if (language === 'curl') {
    const parts = [
      'curl --request ' + method + ' \\',
      '  "' + url + '" \\',
      '  --header "Authorization: Bearer $API_KEY" \\',
      '  --header "Content-Type: application/json"',
    ]
    if (body) {
      parts[parts.length - 1] += ' \\'
      parts.push("  --data '" + body + "'")
    }
    return parts.join('\n')
  }

  if (language === 'python') {
    return [
      'import os',
      'import requests',
      '',
      'response = requests.request(',
      "    '" + method + "',",
      "    '" + url + "',",
      '    headers={',
      "        'Authorization': f\"Bearer {os.environ['API_KEY']}\",",
      "        'Content-Type': 'application/json',",
      '    },',
      body ? '    json=' + body.replace(/\n/g, '\n    ') + ',' : '',
      '    timeout=60,',
      ')',
      'response.raise_for_status()',
      'print(response.json())',
    ]
      .filter((line) => line !== '')
      .join('\n')
  }

  if (language === 'javascript' || language === 'typescript') {
    const declaration = language === 'typescript' ? 'const response: Response' : 'const response'
    return [
      declaration + ' = await fetch(' + JSON.stringify(url) + ', {',
      '  method: ' + JSON.stringify(method) + ',',
      '  headers: {',
      "    Authorization: 'Bearer ' + process.env.API_KEY,",
      "    'Content-Type': 'application/json',",
      '  },',
      body ? '  body: JSON.stringify(' + body.replace(/\n/g, '\n  ') + '),' : '',
      '});',
      '',
      'if (!response.ok) throw new Error(await response.text());',
      'console.log(await response.json());',
    ]
      .filter((line) => line !== '')
      .join('\n')
  }

  const goMethod = 'http.Method' + method.charAt(0) + method.slice(1).toLowerCase()
  return [
    'package main',
    '',
    'import (',
    '    "bytes"',
    '    "fmt"',
    '    "io"',
    '    "net/http"',
    '    "os"',
    ')',
    '',
    'func main() {',
    body ? '    body := bytes.NewBufferString(' + JSON.stringify(body) + ')' : '    body := bytes.NewBuffer(nil)',
    '    req, err := http.NewRequest(' + goMethod + ', ' + JSON.stringify(url) + ', body)',
    '    if err != nil { panic(err) }',
    '    req.Header.Set("Authorization", "Bearer "+os.Getenv("API_KEY"))',
    '    req.Header.Set("Content-Type", "application/json")',
    '    resp, err := http.DefaultClient.Do(req)',
    '    if err != nil { panic(err) }',
    '    defer resp.Body.Close()',
    '    payload, err := io.ReadAll(resp.Body)',
    '    if err != nil { panic(err) }',
    '    fmt.Println(string(payload))',
    '}',
  ].join('\n')
}

export const CODE_LANGUAGES: Array<{ id: CodeLanguage; label: string }> = [
  { id: 'curl', label: 'cURL' },
  { id: 'python', label: 'Python' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'typescript', label: 'TypeScript' },
  { id: 'go', label: 'Go' },
]
