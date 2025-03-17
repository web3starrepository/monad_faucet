package config

import (
	"github.com/spf13/viper"
)

// Config 应用程序全局配置结构
// 字段说明:
//
//	Dynamic: 动态配置参数
//	NocaptchaApi: 验证码API密钥
//	WalletFile: 钱包地址文件路径
//	MaxRetries: 最大重试次数
//	Threads: 并发线程数
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

// GetConfig 获取全局配置实例
// 返回: 当前生效的配置对象
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

// Init 配置初始化函数
// 功能:
// 1. 加载config.yaml配置文件
// 2. 解析CONFIG配置节到全局变量
// 3. 初始化各配置参数
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
