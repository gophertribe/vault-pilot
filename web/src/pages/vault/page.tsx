import { FolderOpen } from "lucide-react"

export function VaultPage() {
  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <FolderOpen className="size-6" />
        Vault
      </h1>
      <p className="text-muted-foreground">
        File tree and markdown preview coming soon.
      </p>
    </div>
  )
}
