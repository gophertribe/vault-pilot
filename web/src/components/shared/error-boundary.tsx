import { Component, type ReactNode } from "react"

interface Props {
  children: ReactNode
}

interface State {
  hasError: boolean
  error?: Error
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error("App error:", error, errorInfo)
  }

  render() {
    if (this.state.hasError && this.state.error) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-4 p-8">
          <h1 className="text-xl font-semibold text-destructive">
            Something went wrong
          </h1>
          <pre className="max-w-2xl overflow-auto rounded bg-muted p-4 text-sm">
            {this.state.error.message}
          </pre>
          <button
            type="button"
            onClick={() => this.setState({ hasError: false, error: undefined })}
            className="rounded bg-primary px-4 py-2 text-primary-foreground"
          >
            Try again
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
