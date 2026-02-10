import { CheckSquare } from "lucide-react"

export function TasksPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <CheckSquare className="size-6" />
        Tasks
      </h1>
      <p className="text-muted-foreground">
        Inbox triage and next actions coming soon.
      </p>
    </div>
  )
}
