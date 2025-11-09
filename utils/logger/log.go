package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("02-01/15:04:05.000"))
}

func NewLogger() (*zap.Logger, error) {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",

		EncodeTime:   customTimeEncoder,
		EncodeLevel:  zapcore.CapitalColorLevelEncoder,
		EncodeCaller: zapcore.ShortCallerEncoder,
	}

	consoleEncoder := zapcore.NewConsoleEncoder(encoderCfg)
	writer := zapcore.Lock(os.Stdout)
	level := zap.NewAtomicLevelAt(zap.DebugLevel)

	core := zapcore.NewCore(consoleEncoder, writer, level)

	baseLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.PanicLevel),
	)

	return baseLogger, nil
}
