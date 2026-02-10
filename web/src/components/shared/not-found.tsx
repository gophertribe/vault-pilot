import { Link } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { FileQuestion } from "lucide-react"

export function NotFound() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 p-6">
      <FileQuestion className="size-16 text-muted-foreground" />
      <h1 className="text-2xl font-semibold">Page not found</h1>
      <p className="text-muted-foreground">
        The page you're looking for doesn't exist or has been moved.
      </p>
      <Button asChild>
        <Link to="/dashboard">Go to Dashboard</Link>
      </Button>
    </div>
  )
}
