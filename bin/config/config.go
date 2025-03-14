package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Dynamic      string `mapstructure:"Dynamic"`
	NocaptchaApi string `mapstructure:"nocaptchaApi"`
	WalletFile   string `mapstructure:"WalletFile"`
	MaxRetries   int    `mapstructure:"MaxRetries"`
	Threads      int    `mapstructure:"Threads"`
}

var GlobalConfig Config

// 全局配置变量
var (
	Dynamic      string
	NocaptchaApi string
	WalletFile   string
	MaxRetries   int
	Threads      int
)

func GetConfig() Config {
	return GlobalConfig
}

// 初始化全局变量
func initGlobalVars() {
	Dynamic = GlobalConfig.Dynamic
	NocaptchaApi = GlobalConfig.NocaptchaApi
	WalletFile = GlobalConfig.WalletFile
	MaxRetries = GlobalConfig.MaxRetries
	Threads = GlobalConfig.Threads
}

// 修改 Init 函数
func Init() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("CONFIG", &GlobalConfig); err != nil {
		return err
	}

	// 初始化全局变量
	initGlobalVars()

	return nil
}
