package apierr

import "github.com/gin-gonic/gin"

const reportContextKey = "api_error_report"

type Report struct {
	Cause error
	Op    string
	Stack string
}

func Record(c *gin.Context, cause error, op string) {
	report, _ := GetReport(c)
	if cause != nil {
		report.Cause = cause
	}
	if op != "" {
		report.Op = op
	}
	c.Set(reportContextKey, report)
}

func RecordWithStack(c *gin.Context, cause error, op, stack string) {
	Record(c, cause, op)
	report, _ := GetReport(c)
	report.Stack = stack
	c.Set(reportContextKey, report)
}

func GetReport(c *gin.Context) (Report, bool) {
	v, ok := c.Get(reportContextKey)
	if !ok {
		return Report{}, false
	}
	report, ok := v.(Report)
	if !ok {
		return Report{}, false
	}
	return report, true
}

func Internal(c *gin.Context, cause error, message, op string) {
	Record(c, cause, op)
	InternalError(message).Respond(c)
}
