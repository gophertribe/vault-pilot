import { useQuery } from "@tanstack/react-query"
import { Zap } from "lucide-react"
import { fetchAutomations } from "@/lib/api"
import { Skeleton } from "@/components/ui/skeleton"
import { Badge } from "@/components/ui/badge"

export function AutomationsPage() {
  const { data: automations, isLoading, error } = useQuery({
    queryKey: ["automations"],
    queryFn: fetchAutomations,
  })

  if (isLoading) {
    return (
      <div className="space-y-6 p-6">
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <Zap className="size-6" />
          Automations
        </h1>
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 w-full" />
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6 p-6">
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <Zap className="size-6" />
          Automations
        </h1>
        <p className="text-destructive">Failed to load automations: {String(error)}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <Zap className="size-6" />
        Automations
      </h1>
      <div className="space-y-2">
        {automations?.length ? (
          automations.map((a) => (
            <div
              key={a.id}
              className="flex items-center justify-between rounded-lg border bg-card p-4"
            >
              <div>
                <p className="font-medium">{a.name}</p>
                <p className="text-sm text-muted-foreground">
                  {a.action_type} • {a.schedule_kind}: {a.schedule_expr}
                </p>
              </div>
              <Badge variant={a.enabled ? "default" : "secondary"}>
                {a.enabled ? "Enabled" : "Paused"}
              </Badge>
            </div>
          ))
        ) : (
          <p className="text-muted-foreground">No automations configured</p>
        )}
      </div>
    </div>
  )
}
