import { type FormEvent, useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { CheckCircle2, KeyRound, Save, Settings } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { fetchSettings, updateSettings } from "@/lib/api"

type Provider = "" | "openai" | "anthropic"

export function SettingsPage() {
  const queryClient = useQueryClient()
  const [provider, setProvider] = useState<Provider>("")
  const [automationTimezone, setAutomationTimezone] = useState("UTC")
  const [openaiApiKey, setOpenaiApiKey] = useState("")
  const [anthropicApiKey, setAnthropicApiKey] = useState("")
  const [telegramToken, setTelegramToken] = useState("")
  const [discordToken, setDiscordToken] = useState("")
  const [clearOpenAI, setClearOpenAI] = useState(false)
  const [clearAnthropic, setClearAnthropic] = useState(false)
  const [clearTelegram, setClearTelegram] = useState(false)
  const [clearDiscord, setClearDiscord] = useState(false)
  const [statusMessage, setStatusMessage] = useState("")

  const { data, isLoading, error } = useQuery({
    queryKey: ["settings"],
    queryFn: fetchSettings,
  })

  useEffect(() => {
    if (!data) return
    setProvider((data.ai_provider as Provider) || "")
    setAutomationTimezone(data.automation_timezone || "UTC")
  }, [data])

  const saveMutation = useMutation({
    mutationFn: updateSettings,
    onSuccess: (nextData) => {
      queryClient.setQueryData(["settings"], nextData)
      setOpenaiApiKey("")
      setAnthropicApiKey("")
      setTelegramToken("")
      setDiscordToken("")
      setClearOpenAI(false)
      setClearAnthropic(false)
      setClearTelegram(false)
      setClearDiscord(false)
      setStatusMessage("Settings saved. Restart background services to apply bot changes.")
    },
    onError: (err) => {
      setStatusMessage(`Failed to save settings: ${String(err)}`)
    },
  })

  const handleSubmit = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    setStatusMessage("")
    saveMutation.mutate({
      ai_provider: provider,
      automation_timezone: automationTimezone.trim() || "UTC",
      openai_api_key: openaiApiKey.trim() || undefined,
      anthropic_api_key: anthropicApiKey.trim() || undefined,
      telegram_token: telegramToken.trim() || undefined,
      discord_token: discordToken.trim() || undefined,
      clear_openai_api_key: clearOpenAI || undefined,
      clear_anthropic_api_key: clearAnthropic || undefined,
      clear_telegram_token: clearTelegram || undefined,
      clear_discord_token: clearDiscord || undefined,
    })
  }

  return (
    <div className="space-y-6 p-6">
      <h1 className="flex items-center gap-2 text-2xl font-semibold">
        <Settings className="size-6" />
        Settings
      </h1>

      {isLoading ? (
        <p className="text-muted-foreground">Loading settings...</p>
      ) : error ? (
        <p className="text-destructive">Failed to load settings: {String(error)}</p>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-6 max-w-3xl">
          <section className="space-y-3 rounded-lg border bg-card p-4">
            <h2 className="font-semibold">AI Provider</h2>
            <p className="text-sm text-muted-foreground">
              Choose one provider and save at least one API key for agent actions.
            </p>
            <label className="text-sm font-medium">Provider</label>
            <select
              className="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
              value={provider}
              onChange={(e) => setProvider(e.target.value as Provider)}
            >
              <option value="">Disabled</option>
              <option value="openai">OpenAI</option>
              <option value="anthropic">Anthropic</option>
            </select>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium">OpenAI API key</label>
                  <SecretStatus configured={Boolean(data?.openai_api_key_configured)} />
                </div>
                <Input
                  type="password"
                  placeholder="Paste a new OpenAI key to replace"
                  value={openaiApiKey}
                  onChange={(e) => setOpenaiApiKey(e.target.value)}
                />
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={clearOpenAI}
                    onChange={(e) => setClearOpenAI(e.target.checked)}
                  />
                  Clear saved key
                </label>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium">Anthropic API key</label>
                  <SecretStatus configured={Boolean(data?.anthropic_api_key_configured)} />
                </div>
                <Input
                  type="password"
                  placeholder="Paste a new Anthropic key to replace"
                  value={anthropicApiKey}
                  onChange={(e) => setAnthropicApiKey(e.target.value)}
                />
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={clearAnthropic}
                    onChange={(e) => setClearAnthropic(e.target.checked)}
                  />
                  Clear saved key
                </label>
              </div>
            </div>
          </section>

          <section className="space-y-3 rounded-lg border bg-card p-4">
            <h2 className="font-semibold">Chat Integrations</h2>
            <p className="text-sm text-muted-foreground">
              Configure tokens so Telegram and Discord capture can run in the background app.
            </p>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium">Telegram bot token</label>
                  <SecretStatus configured={Boolean(data?.telegram_token_configured)} />
                </div>
                <Input
                  type="password"
                  placeholder="Paste a new Telegram token"
                  value={telegramToken}
                  onChange={(e) => setTelegramToken(e.target.value)}
                />
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={clearTelegram}
                    onChange={(e) => setClearTelegram(e.target.checked)}
                  />
                  Clear saved token
                </label>
              </div>
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-sm font-medium">Discord bot token</label>
                  <SecretStatus configured={Boolean(data?.discord_token_configured)} />
                </div>
                <Input
                  type="password"
                  placeholder="Paste a new Discord token"
                  value={discordToken}
                  onChange={(e) => setDiscordToken(e.target.value)}
                />
                <label className="flex items-center gap-2 text-xs text-muted-foreground">
                  <input
                    type="checkbox"
                    checked={clearDiscord}
                    onChange={(e) => setClearDiscord(e.target.checked)}
                  />
                  Clear saved token
                </label>
              </div>
            </div>
          </section>

          <section className="space-y-3 rounded-lg border bg-card p-4">
            <h2 className="font-semibold">Runtime</h2>
            <label className="text-sm font-medium">Automation timezone</label>
            <Input
              placeholder="UTC or America/Los_Angeles"
              value={automationTimezone}
              onChange={(e) => setAutomationTimezone(e.target.value)}
            />
          </section>

          <div className="flex items-center gap-3">
            <Button type="submit" disabled={saveMutation.isPending}>
              <Save className="size-4" />
              Save Settings
            </Button>
            {statusMessage ? <p className="text-sm">{statusMessage}</p> : null}
          </div>
        </form>
      )}
    </div>
  )
}

function SecretStatus({ configured }: { configured: boolean }) {
  if (configured) {
    return (
      <Badge variant="secondary" className="gap-1">
        <CheckCircle2 className="size-3" />
        Configured
      </Badge>
    )
  }
  return (
    <Badge variant="outline" className="gap-1">
      <KeyRound className="size-3" />
      Missing
    </Badge>
  )
}
