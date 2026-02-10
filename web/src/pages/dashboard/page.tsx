import { LayoutDashboard } from "lucide-react"

export function DashboardPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <LayoutDashboard className="size-6" />
        Dashboard
      </h1>
      <p className="text-muted-foreground">
        Daily overview coming soon.
      </p>
    </div>
  )
}
