package event

const (
	// ETLogbookWriteCommit occurs when the logbook writes an op of model
	// `CommitModel`, indicating that a new dataset version has been saved
	// payload is a dsref.VersionInfo
	ETLogbookWriteCommit = Type("logbook:WriteCommit")
	// ETLogbookWriteRun occurs when the logbook writes an op of model
	// `RunModel`, indicating that a new run of a dataset has occured
	// payload is a dsref.VersionInfo
	ETLogbookWriteRun = Type("logbook:WriteRun")
)
