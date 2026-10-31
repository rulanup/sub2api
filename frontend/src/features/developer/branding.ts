const PRODUCT_NAME = 'Sub2API'

export function brandText(value: string, siteName?: string): string {
  const brand = siteName?.trim()
  if (!brand || brand === PRODUCT_NAME) return value
  return value.replace(/Sub2API/g, brand)
}

export function brandMarkdown(markdown: string, siteName?: string): string {
  const brand = siteName?.trim()
  if (!brand || brand === PRODUCT_NAME) return markdown

  let inCodeFence = false
  return markdown
    .split('\n')
    .map((line) => {
      if (line.trimStart().startsWith('~~~')) {
        inCodeFence = !inCodeFence
        return line
      }
      return inCodeFence ? line : brandText(line, brand)
    })
    .join('\n')
}
