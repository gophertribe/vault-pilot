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
