package main

import (
	"fmt"
	"monad/config"
	"os"
	"regexp"

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

// BypassCloudflare 绕过Cloudflare验证获取访问令牌
// 返回: 成功返回验证令牌字符串，失败返回空字符串
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

func (c *goTask) GetverificationToken() (map[string]string, error) {
	resp, err := c.requ.R().
		SetHeaders(map[string]string{
			"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		}).
		Get("https://testnet.monad.xyz/")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}

	// 使用正则表达式匹配
	re := regexp.MustCompile(`requestVerification.*`)
	matches := re.FindAllString(resp.String(), -1)

	if len(matches) == 0 {
		return nil, fmt.Errorf("未找到验证信息")
	}

	// 提取 token 和 timestamp
	token := matches[0][35:99]
	timestamp := matches[0][118:131]

	// 返回包含token和timestamp的map
	return map[string]string{
		"x-request-timestamp":          timestamp,
		"x-request-verification-token": token,
	}, nil
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
		// 初始化请求客户端（使用动态代理配置）
		c := requests.NewClient(cfg.Dynamic, false)
		client := goTask{
			requ:   c,
			apiKey: cfg.NocaptchaApi,
		}
		logger.Logs.Info("[" + wallet + "] 正在处理钱包")
		maxRetries := cfg.MaxRetries
		success := false

		// 重试机制循环
		for i := 0; i < maxRetries; i++ {
			// 执行水龙头领取操作
			err := client.ClaimFaucet(wallet)
			if err != nil {
				logger.Logs.Warnf("[%s] 第 %d 次尝试失败: %v，正在重试...", wallet, i+1, err)
				continue
			}
			success = true
			break
		}

		// 处理最终结果
		if success {
			// 记录成功地址到good.txt
			if err := appendToFile("good.txt", wallet); err != nil {
				logger.Logs.Errorf("[%s] 记录成功地址失败: %v", wallet, err)
			}
			logger.Logs.Infof("[%s] 处理成功", wallet)
		} else {
			// 记录失败地址到fail.txt
			if err := appendToFile("fail.txt", wallet); err != nil {
				logger.Logs.Errorf("[%s] 记录失败地址失败: %v", wallet, err)
			}
			logger.Logs.Errorf("[%s] 重试 %d 次后仍然失败", wallet, maxRetries)
		}
	}
}

// 同时修改 ClaimFaucet 函数中的日志输出
func (c *goTask) ClaimFaucet(address string) error {
	// 获取Cloudflare验证令牌
	cftoken := c.BypassCloudflare()
	if cftoken == "" {
		return fmt.Errorf("BypassCloudflare failed")
	}

	// 生成唯一访问者ID
	visitorId := strings.ReplaceAll(uuid.New().String(), "-", "")

	// 构建请求参数
	postData := map[string]string{
		"address":                 address,
		"visitorId":               visitorId,
		"cloudFlareResponseToken": cftoken,
	}

	headers, err := c.GetverificationToken()
	if err != nil {
		return fmt.Errorf("GetverificationToken failed %v", err)
	}
	// 设置请求头
	c.requ.SetCommonHeaders(headers)

	// 发送POST请求到水龙头接口
	resp, err := c.requ.R().
		SetBody(postData).
		Post("https://faucet-claim.monadinfra.com/")

	// 处理请求错误
	if err != nil {
		return fmt.Errorf("领取失败: %v", err)
	}

	// 处理405状态码（方法不允许）
	if resp.GetStatusCode() == 405 {
		return fmt.Errorf("请求返回 405")
	}

	// 解析响应JSON
	var result struct {
		Message string `json:"message"`
	}
	if err := resp.UnmarshalJson(&result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if result.Message != "Success" {
		return fmt.Errorf("领取失败: %s", result.Message)
	}

	// 记录成功日志
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

	// title
	logger.Logs.Info("Monad Faucet")
	logger.Logs.Info("Author: Web3StarRepository")
	logger.Logs.Info("TG: @Web3um")
	logger.Logs.Info("开源脚本，如果你是买的，那么你就是大呆比")

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
