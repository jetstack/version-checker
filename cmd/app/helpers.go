package app

import (
	"os"

	"github.com/sirupsen/logrus"
)

func newLogger(logLevel logrus.Level) *logrus.Entry {
	nlog := logrus.New()
	nlog.SetOutput(os.Stdout)
	nlog.SetLevel(logLevel)
	log := logrus.NewEntry(nlog)
	return log
}
