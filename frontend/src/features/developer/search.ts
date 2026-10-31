import { developerDocuments, developerEndpoints } from './content'
import { CODE_LANGUAGES, generateCodeExample } from './codeExamples'
import { brandMarkdown, brandText } from './branding'
import type {
  DeveloperSearchOptions,
  DeveloperSearchResult,
  PortalLocale,
} from './types'

function normalized(value: string): string {
  return value.trim().toLocaleLowerCase()
}

function textScore(haystack: string, needle: string, exactScore: number, includesScore: number): number {
  const source = normalized(haystack)
  if (!source || !needle) return 0
  if (source === needle) return exactScore
  return source.includes(needle) ? includesScore : 0
}

function matchesFilter(value: string, filter?: string): boolean {
  return !filter || filter === 'all' || normalized(value) === normalized(filter)
}

export function searchDeveloperContent(
  query: string,
  options: DeveloperSearchOptions,
): DeveloperSearchResult[] {
  const needle = normalized(query)
  if (!needle) return []

  const locale: PortalLocale = options.locale
  const results: DeveloperSearchResult[] = []

  for (const document of developerDocuments) {
    if (!matchesFilter(document.category, options.category)) continue
    if (options.version && options.version !== 'all' && document.version !== options.version) continue

    const title = brandText(document.title[locale], options.siteName)
    const description = brandText(document.description[locale], options.siteName)
    const content = brandMarkdown(document.content[locale], options.siteName)
    const titleScore = textScore(title, needle, 320, 260)
    const descriptionScore = textScore(description, needle, 180, 130)
    const contentScore = textScore(content, needle, 150, 90)
    const tagScore = document.tags.reduce(
      (best, tag) => Math.max(best, textScore(tag, needle, 220, 150)),
      0,
    )
    const score = titleScore + descriptionScore + contentScore + tagScore

    if (score > 0) {
      results.push({
        kind: 'document',
        id: document.id,
        title,
        description,
        category: document.category,
        path: '/developer/docs/' + document.slug,
        score,
      })
    }
  }

  for (const endpoint of developerEndpoints) {
    if (!matchesFilter(endpoint.category, options.category)) continue
    if (options.version && options.version !== 'all' && endpoint.version !== options.version) continue

    const title = brandText(endpoint.summary[locale], options.siteName)
    const description = brandText(endpoint.description[locale], options.siteName)
    const pathScore = textScore(endpoint.path, needle, 1000, 760)
    const methodScore = textScore(endpoint.method + ' ' + endpoint.path, needle, 980, 720)
    const parameterScore = endpoint.parameters.reduce((best, parameter) => {
      return Math.max(best, textScore(parameter.name, needle, 560, 420))
    }, 0)
    const errorScore = endpoint.errors.reduce((best, error) => {
      return Math.max(
        best,
        textScore(String(error.status), needle, 520, 410),
        textScore(error.code, needle, 520, 410),
      )
    }, 0)
    const titleScore = textScore(title, needle, 340, 280)
    const descriptionScore = textScore(description, needle, 190, 120)
    const exampleScore = CODE_LANGUAGES.reduce(
      (best, language) =>
        Math.max(best, textScore(generateCodeExample(endpoint, language.id), needle, 320, 220)),
      0,
    )
    const score =
      Math.max(pathScore, methodScore) +
      parameterScore +
      errorScore +
      titleScore +
      descriptionScore +
      exampleScore

    if (score > 0) {
      results.push({
        kind: 'endpoint',
        id: endpoint.id,
        title,
        description: endpoint.method + ' ' + endpoint.path + ' — ' + description,
        category: endpoint.category,
        path: '/developer/api?endpoint=' + endpoint.id,
        score,
      })
    }
  }

  return results
    .sort((left, right) => right.score - left.score || left.title.localeCompare(right.title))
    .slice(0, options.limit ?? 30)
}
