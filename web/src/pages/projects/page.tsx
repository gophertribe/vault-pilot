import { useQuery } from "@tanstack/react-query"
import { FolderKanban } from "lucide-react"
import { apiFetch } from "@/lib/api"
import { projectsResponseSchema } from "@/lib/api/schemas"
import { Skeleton } from "@/components/ui/skeleton"

async function fetchProjects() {
  const data = await apiFetch<unknown>("/projects")
  const parsed = projectsResponseSchema.parse(data)
  return parsed.projects
}

export function ProjectsPage() {
  const { data: projects, isLoading, error } = useQuery({
    queryKey: ["projects"],
    queryFn: fetchProjects,
  })

  if (isLoading) {
    return (
      <div className="space-y-6 p-6">
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <FolderKanban className="size-6" />
          Projects
        </h1>
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6 p-6">
        <h1 className="flex items-center gap-2 text-2xl font-semibold">
          <FolderKanban className="size-6" />
          Projects
        </h1>
        <p className="text-destructive">Failed to load projects: {String(error)}</p>
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <FolderKanban className="size-6" />
        Projects
      </h1>
      <ul className="list-disc space-y-1 pl-6">
        {projects?.length ? (
          projects.map((name) => (
            <li key={name}>{name}</li>
          ))
        ) : (
          <li className="text-muted-foreground">No active projects</li>
        )}
      </ul>
    </div>
  )
}
