import { apiFetch } from "./client"
import {
  settingsResponseSchema,
  updateSettingsRequestSchema,
  type SettingsResponse,
  type UpdateSettingsRequest,
} from "./schemas"

export async function fetchSettings(): Promise<SettingsResponse> {
  const res = await apiFetch<unknown>("/settings")
  return settingsResponseSchema.parse(res)
}

export async function updateSettings(
  payload: UpdateSettingsRequest
): Promise<SettingsResponse> {
  const req = updateSettingsRequestSchema.parse(payload)
  const res = await apiFetch<unknown>("/settings", {
    method: "PUT",
    body: JSON.stringify(req),
  })
  return settingsResponseSchema.parse(res)
}
