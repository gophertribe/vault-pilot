import { Calendar } from "lucide-react"

export function CalendarPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <Calendar className="size-6" />
        Calendar
      </h1>
      <p className="text-muted-foreground">
        Week and month views coming soon.
      </p>
    </div>
  )
}
