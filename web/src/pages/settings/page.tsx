import { Settings } from "lucide-react"

export function SettingsPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <Settings className="size-6" />
        Settings
      </h1>
      <p className="text-muted-foreground">
        Integration health and preferences coming soon.
      </p>
    </div>
  )
}
