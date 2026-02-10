const API_BASE = "/api"

export class ApiError extends Error {
  status: number
  body?: unknown
  constructor(message: string, status: number, body?: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.body = body
  }
}

export async function apiFetch<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const url = path.startsWith("http") ? path : `${API_BASE}${path}`
  const res = await fetch(url, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  })

  if (!res.ok) {
    let body: unknown
    try {
      body = await res.json().catch(() => res.text())
    } catch {
      body = undefined
    }
    throw new ApiError(
      `API error: ${res.status} ${res.statusText}`,
      res.status,
      body
    )
  }

  const contentType = res.headers.get("content-type")
  if (contentType?.includes("application/json")) {
    return res.json() as Promise<T>
  }
  return res.text() as unknown as Promise<T>
}
