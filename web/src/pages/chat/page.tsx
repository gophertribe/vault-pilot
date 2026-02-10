import { MessageSquare } from "lucide-react"

export function ChatPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <MessageSquare className="size-6" />
        Chat
      </h1>
      <p className="text-muted-foreground">
        AI chat workspace coming soon.
      </p>
    </div>
  )
}
