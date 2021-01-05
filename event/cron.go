package event

const (
	// ETCronJobScheduled fires when a job is registered for updating, or when
	// a scheduled job changes
	// payload is a Job
	// subscriptions do not block the publisher
	ETCronJobScheduled = Type("cron:JobScheduled")
	// ETCronJobUnscheduled fires when a job is removed from the update schedule
	// payload is a Job
	// subscriptions do not block the publisher
	ETCronJobUnscheduled = Type("cron:JobUnscheduled")
	// ETCronJobStarted fires when a job has started running
	// payload is a Job
	// subscriptions do not block the publisher
	ETCronJobStarted = Type("cron:JobStarted")
	// ETCronJobCompleted fires when a job has finished running
	// payload is a Job
	// subscriptions do not block the publisher
	ETCronJobCompleted = Type("cron:JobCompleted")
)
