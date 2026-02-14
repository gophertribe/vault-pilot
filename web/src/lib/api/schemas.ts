import { z } from "zod"

export const automationDefinitionSchema = z.object({
  id: z.number(),
  name: z.string(),
  action_type: z.string(),
  schedule_kind: z.string(),
  schedule_expr: z.string(),
  timezone: z.string(),
  payload_json: z.string(),
  enabled: z.boolean(),
  next_run_at: z.string().nullable().optional(),
  last_run_at: z.string().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})

export type AutomationDefinition = z.infer<typeof automationDefinitionSchema>

export const automationsResponseSchema = z.object({
  automations: z.array(automationDefinitionSchema),
})

export type AutomationsResponse = z.infer<typeof automationsResponseSchema>

export const projectsResponseSchema = z.object({
  projects: z.array(z.string()),
})

export type ProjectsResponse = z.infer<typeof projectsResponseSchema>

export const settingsResponseSchema = z.object({
  ai_provider: z.string(),
  automation_timezone: z.string(),
  openai_api_key_configured: z.boolean(),
  anthropic_api_key_configured: z.boolean(),
  telegram_token_configured: z.boolean(),
  discord_token_configured: z.boolean(),
})

export type SettingsResponse = z.infer<typeof settingsResponseSchema>

export const updateSettingsRequestSchema = z.object({
  ai_provider: z.string(),
  automation_timezone: z.string(),
  openai_api_key: z.string().optional(),
  anthropic_api_key: z.string().optional(),
  telegram_token: z.string().optional(),
  discord_token: z.string().optional(),
  clear_openai_api_key: z.boolean().optional(),
  clear_anthropic_api_key: z.boolean().optional(),
  clear_telegram_token: z.boolean().optional(),
  clear_discord_token: z.boolean().optional(),
})

export type UpdateSettingsRequest = z.infer<typeof updateSettingsRequestSchema>
