import { createBrowserRouter, Navigate } from "react-router-dom"
import { Suspense, lazy } from "react"
import { AppLayout } from "@/app/providers"
import { PageLoading } from "@/components/shared/page-loading"
import { NotFound } from "@/components/shared/not-found"

const DashboardPage = lazy(() =>
  import("@/pages/dashboard/page").then((m) => ({ default: m.DashboardPage }))
)
const CalendarPage = lazy(() =>
  import("@/pages/calendar/page").then((m) => ({ default: m.CalendarPage }))
)
const VaultPage = lazy(() =>
  import("@/pages/vault/page").then((m) => ({ default: m.VaultPage }))
)
const TasksPage = lazy(() =>
  import("@/pages/tasks/page").then((m) => ({ default: m.TasksPage }))
)
const ProjectsPage = lazy(() =>
  import("@/pages/projects/page").then((m) => ({ default: m.ProjectsPage }))
)
const AutomationsPage = lazy(() =>
  import("@/pages/automations/page").then((m) => ({ default: m.AutomationsPage }))
)
const ChatPage = lazy(() =>
  import("@/pages/chat/page").then((m) => ({ default: m.ChatPage }))
)
const SettingsPage = lazy(() =>
  import("@/pages/settings/page").then((m) => ({ default: m.SettingsPage }))
)

export const router = createBrowserRouter(
  [
    {
      path: "/",
      element: <AppLayout />,
      children: [
        {
          index: true,
          element: <Navigate to="/dashboard" replace />,
        },
        {
          path: "dashboard",
          element: (
            <Suspense fallback={<PageLoading />}>
              <DashboardPage />
            </Suspense>
          ),
        },
        {
          path: "calendar",
          element: (
            <Suspense fallback={<PageLoading />}>
              <CalendarPage />
            </Suspense>
          ),
        },
        {
          path: "vault",
          element: (
            <Suspense fallback={<PageLoading />}>
              <VaultPage />
            </Suspense>
          ),
        },
        {
          path: "tasks",
          element: (
            <Suspense fallback={<PageLoading />}>
              <TasksPage />
            </Suspense>
          ),
        },
        {
          path: "projects",
          element: (
            <Suspense fallback={<PageLoading />}>
              <ProjectsPage />
            </Suspense>
          ),
        },
        {
          path: "automations",
          element: (
            <Suspense fallback={<PageLoading />}>
              <AutomationsPage />
            </Suspense>
          ),
        },
        {
          path: "chat",
          element: (
            <Suspense fallback={<PageLoading />}>
              <ChatPage />
            </Suspense>
          ),
        },
        {
          path: "settings",
          element: (
            <Suspense fallback={<PageLoading />}>
              <SettingsPage />
            </Suspense>
          ),
        },
        {
          path: "*",
          element: <NotFound />,
        },
      ],
    },
  ],
  { basename: "/app" }
)
