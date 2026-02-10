import { apiFetch } from "./client"
import {
  automationsResponseSchema,
  type AutomationDefinition,
} from "./schemas"

export async function fetchAutomations(): Promise<AutomationDefinition[]> {
  const data = await apiFetch<unknown>("/automations")
  const parsed = automationsResponseSchema.parse(data)
  return parsed.automations
}
