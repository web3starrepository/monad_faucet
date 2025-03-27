package utils

import (
	"log"
	"os"
	"time"

	"github.com/gookit/color"
	"github.com/gookit/slog"
	"github.com/gookit/slog/handler"
	"github.com/gookit/slog/rotatefile"
)

var (
	Logs *slog.Logger
	// 设置颜色化输出
	LightBlue   = color.New(color.LightBlue).Sprint   // 任务编号为蓝色
	BgLightCyan = color.New(color.BgLightCyan).Sprint // 错误信息为红色
)

func clearLogs() {

	Logs.Info("🔄 定时清理日志功能开启")
	// 设置清理间隔为1天
	interval := 24 * time.Hour

	go func() {
		for {

			time.Sleep(interval)

			// 删除日志文件
			err := os.Remove("logs/")
			if err != nil && !os.IsNotExist(err) {
				Logs.Error("清理日志文件失败:", err)
			}

			// 等待下一次清理

		}
	}()
}

func InitLogger(clearLogsBool bool) {
	// 定义日志文件路径和轮转设置
	logFilePath := "logs/app.log"

	writer, err := rotatefile.NewConfig(logFilePath).
		Create()
	if err != nil {
		panic(err)
	}

	log.SetOutput(writer)

	myTemplate := "[{{datetime}}] [{{level}}] {{message}} {{data}} {{extra}}\n"
	h := handler.NewConsoleHandler(slog.AllLevels)
	h.Formatter().(*slog.TextFormatter).SetTemplate(myTemplate)
	h.TextFormatter().EnableColor = true

	// 设置颜色
	h.TextFormatter().ColorTheme = map[slog.Level]color.Color{
		slog.PanicLevel:  color.FgRed,
		slog.FatalLevel:  color.FgRed,
		slog.ErrorLevel:  color.FgMagenta,
		slog.WarnLevel:   color.FgYellow,
		slog.NoticeLevel: color.FgBlue,
		slog.InfoLevel:   color.FgGreen,
		slog.DebugLevel:  color.FgCyan,
		slog.TraceLevel:  color.FgLightGreen,
	}

	Logs = slog.NewWithHandlers(h)
	// Set the custom logger to use the rotatefile writer for file output
	Logs.AddHandler(handler.NewSimple(writer, slog.Level(slog.InfoLevel|slog.DebugLevel|slog.WarnLevel|slog.ErrorLevel|slog.FatalLevel)))

	// 如果需要清理日志，则开启清理功能
	if clearLogsBool {
		clearLogs()
	}
}
