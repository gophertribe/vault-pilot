package events

const (
	TypeMailReceived             = "mail.received"
	TypeMailSynced               = "mail.synced"
	TypeGTDTaskCreated           = "gtd.task.created"
	TypeGTDInboxCaptured         = "gtd.inbox.captured"
	TypeGTDInsightCreated        = "gtd.insight.created"
	TypeBusinessInvoiceDetected  = "business.invoice.detected"
	TypeBusinessInvoiceProcessed = "business.invoice.processed"
	TypeSoftwareJobStarted       = "software.job.started"
	TypeSoftwareJobProgress      = "software.job.progress"
	TypeSoftwareJobCompleted     = "software.job.completed"
	TypeJobQueued                = "job.queued"
	TypeJobStarted               = "job.started"
	TypeJobCompleted             = "job.completed"
	TypeJobFailed                = "job.failed"
)
