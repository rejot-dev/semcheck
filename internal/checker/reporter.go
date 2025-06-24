package checker

// Reporter defines the interface for reporting semantic analysis results
type Reporter interface {
	Report(result *CheckResult)
}
