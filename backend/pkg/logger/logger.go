package logger

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Log is a global logger
	Log *zap.Logger

	// customTimeFormat is a string
	// representing a custom time format
	customTimeFormat string

	// onceInit makes sure logger is
	// initialized only one time
	onceInit sync.Once
)

// customTimeEncoder encode Time to our custom format
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(customTimeFormat))
}

// Init initializes log by input parameters
// lvl - global log level: Debug(-1), Info(0), Warn(1), Error(2), DPanic(3), Panic(4), Fatal(5)
// timeFormat - custom time format for logger of empty string to use default
func Init(lvl int, timeFormat string) error {
	var err error

	onceInit.Do(func() {
		// Define log level
		globalLevel := zapcore.Level(lvl)

		// High-priority output should go to standard error,
		// and low-priority output should go to standard out.
		// It is useful for Kubernetes deployment.
		// Kubernetes interprets os.Stdout log items as INFO
		// and os.Stderr log items as ERROR by default.
		highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		})
		lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= globalLevel && lvl < zapcore.ErrorLevel
		})
		consoleInfos := zapcore.Lock(os.Stdout)
		consoleErrors := zapcore.Lock(os.Stderr)

		// Configure console output
		var useCustomTimeFormat bool

		ecfg := zap.NewProductionEncoderConfig()

		if len(timeFormat) > 0 {
			customTimeFormat = timeFormat
			ecfg.EncodeTime = customTimeEncoder
			useCustomTimeFormat = true
		}
		consoleEncoder := zapcore.NewJSONEncoder(ecfg)

		// Join the outputs, encoders, and level-handling functions into zapcore
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleInfos, lowPriority),
		)

		Log = zap.New(core)
		zap.RedirectStdLog(Log)

		if !useCustomTimeFormat {
			Log.Warn("time format for logger is not provided - use zap default")
		}
	})

	return err
}
