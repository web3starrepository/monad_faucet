package main

import (
	"fmt"
	"monad/config"
	"os"

	"strings"

	"sync"

	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	"github.com/panjf2000/ants/v2"
	"github.com/web3starrepository/web3utils/utils/logger"
	"github.com/web3starrepository/web3utils/utils/nocaptcha"
	"github.com/web3starrepository/web3utils/utils/requests"
)

type goTask struct {
	requ   *req.Client
	apiKey string
}

func (c *goTask) BypassCloudflare() string {

	token, err := nocaptcha.BypassCloudflare(nocaptcha.NocaptchaType{
		User_Token_KEY:     c.apiKey,
		Cloudflare_URL:     "https://testnet.monad.xyz/",
		Cloudflare_SITEKEY: "0x4AAAAAAA-3X4Nd7hf3mNGx",
		Request:            c.requ,
	},
	)

	if err != nil {
		return ""
	}

	return token

}

// 新版本不使用
func (c *goTask) Bypassvercel(proxy string) (map[string]string, error) {
	user_agent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	href := "https://testnet.monad.xyz/"

	headers := map[string]string{
		"User-Token": c.apiKey,
	}

	jsonData := map[string]interface{}{
		"href":       href,
		"user_agent": user_agent,
		"proxy":      proxy,
		"timeout":    90,
	}

	resp, err := c.requ.R().SetHeaders(headers).SetBody(jsonData).Post("http://api.nocaptcha.cn/api/wanda/vercel/universal")
	if err != nil {
		return nil, err
	}

	var result struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Data   struct {
			Vcrcs string `json:"_vcrcs"`
		} `json:"data"`
		Extra map[string]string `json:"extra"`
	}

	if err := resp.UnmarshalJson(&result); err != nil {
		return nil, err
	}

	if result.Status != 1 {
		if result.Msg == "未触发 vercel 盾" {
			logger.Logs.Error("Bypass失败，错误信息：", result.Msg) // 添加错误信息打印
			return nil, nil
		}
		logger.Logs.Error("Bypass失败，错误信息：", result.Msg) // 添加错误信息打印
		return nil, fmt.Errorf("bypass failed: %s", result.Msg)
	}

	// 构建完整的 headers
	requestHeaders := map[string]string{
		"sec-ch-ua":                 result.Extra["sec-ch-ua"],
		"sec-ch-ua-mobile":          "?0",
		"sec-ch-ua-platform":        result.Extra["sec-ch-ua-platform"],
		"upgrade-insecure-requests": "1",
		"user-agent":                result.Extra["user-agent"],
		"accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"sec-fetch-site":            "same-origin",
		"sec-fetch-mode":            "navigate",
		"sec-fetch-dest":            "document",
		"referer":                   href,
		"accept-encoding":           "gzip, deflate, br, zstd",
		"accept-language":           result.Extra["accept-language"],
		"cookie":                    "_vcrcs=" + result.Data.Vcrcs,
		"priority":                  "u=0, i",
	}

	return requestHeaders, nil
}

// 添加文件写入函数
func appendToFile(filename string, content string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(content + "\n"); err != nil {
		return err
	}
	return nil
}

// 修改处理函数
func processWallet(wallet string, cfg config.Config) func() {
	return func() {
		c := requests.NewClient(cfg.Dynamic, false)
		client := goTask{
			requ:   c,
			apiKey: cfg.NocaptchaApi,
		}
		logger.Logs.Info("[" + wallet + "] 正在处理钱包")
		maxRetries := cfg.MaxRetries
		success := false

		for i := 0; i < maxRetries; i++ {
			err := client.ClaimFaucet(wallet)
			if err != nil {
				logger.Logs.Warnf("[%s] 第 %d 次尝试失败: %v，正在重试...", wallet, i+1, err)
				continue
			}
			success = true
			break
		}

		if success {
			if err := appendToFile("good.txt", wallet); err != nil {
				logger.Logs.Errorf("[%s] 记录成功地址失败: %v", wallet, err)
			}
			logger.Logs.Infof("[%s] 处理成功", wallet)
		} else {
			if err := appendToFile("fail.txt", wallet); err != nil {
				logger.Logs.Errorf("[%s] 记录失败地址失败: %v", wallet, err)
			}
			logger.Logs.Errorf("[%s] 重试 %d 次后仍然失败", wallet, maxRetries)
		}
	}
}

// 同时修改 ClaimFaucet 函数中的日志输出
func (c *goTask) ClaimFaucet(address string) error {
	cftoken := c.BypassCloudflare()
	if cftoken == "" {
		return fmt.Errorf("BypassCloudflare failed")
	}

	// Generate visitor ID using UUID
	visitorId := strings.ReplaceAll(uuid.New().String(), "-", "")

	postData := map[string]string{
		"address":                 address,
		"visitorId":               visitorId,
		"cloudFlareResponseToken": cftoken,
	}

	resp, err := c.requ.R().
		SetBody(postData).
		Post("https://testnet.monad.xyz/api/faucet/claim")

	if err != nil {
		return fmt.Errorf("领取失败: %v", err)
	}

	if resp.GetStatusCode() == 405 {
		return fmt.Errorf("请求返回 405")
	}

	logger.Logs.Info("[" + address + "] 领取成功")
	logger.Logs.Info("[" + address + "] " + resp.String())
	return nil
}

func readWallets(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var wallets []string
	for _, line := range lines {
		if addr := strings.TrimSpace(line); addr != "" {
			wallets = append(wallets, addr)
		}
	}
	return wallets, nil
}

func main() {
	logger.InitLogger(true)

	// 初始化配置
	if err := config.Init(); err != nil {
		logger.Logs.Error("配置初始化失败:", err)
		return
	}

	cfg := config.GetConfig()

	// 读取钱包地址
	wallets, err := readWallets(cfg.WalletFile)
	if err != nil {
		logger.Logs.Error("读取钱包文件失败:", err)
		return
	}

	// 创建协程池
	p, err := ants.NewPool(cfg.Threads) // 设置最大并发数为10
	if err != nil {
		logger.Logs.Error("创建协程池失败:", err)
		return
	}
	defer p.Release()

	var wg sync.WaitGroup
	// 循环处理每个钱包地址
	for _, wallet := range wallets {
		wg.Add(1)
		wallet := wallet // 创建副本避免闭包问题

		err := p.Submit(func() {
			defer wg.Done()
			processWallet(wallet, cfg)() // 使用保留的第一个版本
		})

		if err != nil {
			logger.Logs.Error("提交任务失败:", err)
			wg.Done()
			continue
		}
	}

	wg.Wait()
	logger.Logs.Info("所有任务处理完成")
}
