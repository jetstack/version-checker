// Creates a leveled logger compatible with hashicorp/go-retryablehttp
package leveledlogger

import "github.com/sirupsen/logrus"

type Logger struct {
	*logrus.Entry
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	l.fields(keysAndValues).Error(msg)
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	l.fields(keysAndValues).Info(msg)
}
func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	l.fields(keysAndValues).Debug(msg)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	l.fields(keysAndValues).Warn(msg)
}

func (l *Logger) fields(keysAndValues []interface{}) *logrus.Entry {
	if len(keysAndValues) == 0 {
		return l.Entry
	}

	fields := make(map[string]interface{}, len(keysAndValues)/2)

	for i := 0; i < len(keysAndValues)-1; i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			// Skip invalid key-value pairs
			continue
		}
		fields[key] = keysAndValues[i+1]
	}

	// Handle odd number of elements - log the trailing key without a value
	if len(keysAndValues)%2 != 0 {
		key, ok := keysAndValues[len(keysAndValues)-1].(string)
		if ok {
			fields[key] = "(MISSING)"
		}
	}

	if len(fields) == 0 {
		return l.Entry
	}

	return l.WithFields(fields)
}
