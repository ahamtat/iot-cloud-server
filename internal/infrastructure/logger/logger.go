package logger

import (
	"log"
	"sync"

	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"

	"gopkg.in/natefinch/lumberjack.v2"
)

var sugar *zap.SugaredLogger
var once sync.Once

// Init initializes a thread-safe singleton logger
// This would be called from a main method when the application starts up
// This function would ideally, take zap configuration, but is left out
// in favor of simplicity using the example logger.
func Init(logLevel, filePath string, logRotate bool) {
	// once ensures the singleton is initialized only once
	once.Do(func() {
		config := zap.NewProductionConfig()
		config.OutputPaths = []string{
			filePath,
			"stderr",
		}
		config.ErrorOutputPaths = []string{
			filePath,
			"stderr",
		}
		config.EncoderConfig = zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			//CallerKey:    "caller",
			//EncodeCaller: zapcore.ShortCallerEncoder,
		}

		// Set log level
		var level zapcore.Level
		err := level.UnmarshalText([]byte(logLevel))
		if err != nil {
			log.Fatalf("can't marshal level string: %v", logLevel)
		}
		config.Level = zap.NewAtomicLevelAt(level)

		// Added logrotate syncer from
		// https://github.com/uber-go/zap/issues/342
		var syncer zapcore.WriteSyncer
		if logRotate {
			syncer = zapcore.AddSync(&lumberjack.Logger{
				Filename:   filePath,
				MaxSize:    1, // megabytes
				MaxBackups: 5,
				MaxAge:     28, // days
			})
		}

		// Create logger
		logger, err := config.Build() //SetOutput(syncer, config))
		if err != nil {
			log.Fatalf("can't initialize zap logger: %v", err)
		}
		defer func() {
			if err := logger.Sync(); err != nil {
				// Do not process sync err as told in:
				// https://github.com/uber-go/zap/issues/328
				//log.Fatalf("can't sync logger: %v", err)
			}
		}()

		if logRotate {
			sugar = sugar.Desugar().WithOptions(SetOutput(syncer, config)).Sugar()
		} else {
			sugar = logger.Sugar()
		}
	})
}

// SetOutput replaces existing Core with new, that writes to passed WriteSyncer.
func SetOutput(ws zapcore.WriteSyncer, conf zap.Config) zap.Option {
	var enc zapcore.Encoder
	// Copy paste from zap.Config.buildEncoder.
	switch conf.Encoding {
	case "json":
		enc = zapcore.NewJSONEncoder(conf.EncoderConfig)
	case "console":
		enc = zapcore.NewConsoleEncoder(conf.EncoderConfig)
	default:
		panic("unknown encoding")
	}
	return zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return zapcore.NewCore(enc, ws, conf.Level)
	})
}

// Debug logs a debug message with the given fields
func Debug(message string, fields ...interface{}) {
	sugar.Debugw(message, fields...)
}

// Info logs a debug message with the given fields
func Info(message string, fields ...interface{}) {
	sugar.Infow(message, fields...)
}

// Warn logs a debug message with the given fields
func Warn(message string, fields ...interface{}) {
	sugar.Warnw(message, fields...)
}

// Error logs a debug message with the given fields
func Error(message string, fields ...interface{}) {
	sugar.Errorw(message, fields...)
}

// Fatal logs a message than calls os.Exit(1)
func Fatal(message string, fields ...interface{}) {
	sugar.Fatalw(message, fields...)
}
