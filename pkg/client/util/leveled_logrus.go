package util

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func NewLeveledLogger(entry *logrus.Entry) *LeveledLogrus {
	return &LeveledLogrus{Entry: entry}
}

type LeveledLogrus struct {
	*logrus.Entry
}

func (l *LeveledLogrus) Error(msg string, keysAndValues ...interface{}) {
	l.WithFields(fields(keysAndValues)).Error(msg)
}

func (l *LeveledLogrus) Info(msg string, keysAndValues ...interface{}) {
	l.WithFields(fields(keysAndValues)).Info(msg)
}
func (l *LeveledLogrus) Debug(msg string, keysAndValues ...interface{}) {
	l.WithFields(fields(keysAndValues)).Debug(msg)
}

func (l *LeveledLogrus) Warn(msg string, keysAndValues ...interface{}) {
	l.WithFields(fields(keysAndValues)).Warn(msg)
}

func fields(keysAndValues []interface{}) map[string]interface{} {
	fields := make(map[string]interface{})

	for i := 0; i < len(keysAndValues)-1; i += 2 {
		// turn *any* key into its string representation
		keyStr := fmt.Sprint(keysAndValues[i])
		fields[keyStr] = keysAndValues[i+1]
	}

	return fields
}
