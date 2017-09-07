package logger

import (
	"fmt"
	"github.com/evalphobia/logrus_sentry"
	"github.com/getsentry/raven-go"
	"github.com/sirupsen/logrus"
	"github.com/stgleb/logrus-logstash-hook"
	"github.com/x-cray/logrus-prefixed-formatter"
	"net"
	"time"
)

var instance *logrus.Logger

func Set(logger *logrus.Logger) {
	instance = logger
}

func Get() *logrus.Logger {
	if instance == nil {
		return logrus.New()
	}
	return instance
}

func InitLogger(level int) {
	instance = logrus.New()
	logLevel := logrus.AllLevels[level]
	instance.Level = logLevel

	instance.Infof("Logger - Logging established with level %q on stderr", logLevel)
}

func AddLogstashHook(host string, port int, protocol string, level int) {
	hostPort := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.Dial(protocol, hostPort)

	if err != nil {
		Get().Errorf("Logger - Error dialing logstash (%s): %s", hostPort, err.Error())
	} else {
		formatter := new(prefixed.TextFormatter)
		logstashLevel := logrus.AllLevels[level]
		Get().Infof("Logger - Establish %s connection on %s", protocol, hostPort)
		hook := logrustash.New(conn, formatter)

		if err := hook.Fire(&logrus.Entry{}); err != nil {
			Get().Errorf("Logger - Error firing logstash hook: %s", err.Error())
		} else {
			Get().Infof("Logger - Add hook for logstash with level %q", logstashLevel)
			hook.SetLevel(logstashLevel)
			Get().Hooks.Add(hook)
		}
	}
}

func AddSentryHook(apiKey, secret, host, projectId, release, env string) {
	dsn := fmt.Sprintf("https://%s:%s@%s/%s",
		apiKey,
		secret,
		host,
		projectId)

	Get().Infof("Sentry - Adding hook to logger to url %q", dsn)

	// ---configure default client
	if err := raven.SetDSN(dsn); err != nil {
		Get().Errorf("Sentry - Error setting DSN to default client %q", err.Error())
	}
	raven.SetRelease(release)
	raven.SetEnvironment(env)
	// ---

	client, err := raven.New(dsn)
	// Set basic information
	client.SetRelease(release)
	client.SetEnvironment(env)

	if err != nil {
		Get().Errorf("Sentry - Error getting new Sentry client instance %q", err.Error())
	}

	hook, err := logrus_sentry.NewWithClientSentryHook(client, []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	})

	// Add hook for collecting stack traces
	hook.StacktraceConfiguration.Enable = true
	hook.Timeout = time.Second * 5

	if err != nil {
		Get().Errorf("Sentry - Error creating a hook using an initialized client %q", err.Error())
	} else {
		Get().Hooks.Add(hook)
	}
}
