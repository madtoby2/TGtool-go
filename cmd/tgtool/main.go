package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/madtoby2/tgtool/internal/clone"
	"github.com/madtoby2/tgtool/internal/collector"
	"github.com/madtoby2/tgtool/internal/config"
	"github.com/madtoby2/tgtool/internal/farming"
	"github.com/madtoby2/tgtool/internal/filter"
	"github.com/madtoby2/tgtool/internal/sender"
	"github.com/madtoby2/tgtool/internal/session"
)

var (
	svc   = session.NewManager()
	col   = collector.New()
	snd   = sender.New()
	ctx   = context.Background()
)

func banner() {
	fmt.Println(`
╔══════════════════════════════════════════════╗
║          ⚡ 疾风TG营销助手 v2.0 (Go)         ║
║          Telegram Marketing Tool             ║
╚══════════════════════════════════════════════╝`)
}

func readInput(prompt string) string {
	fmt.Print(prompt)
	s, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(s)
}

func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("文件不存在: %s\n", path)
		return nil
	}
	var lines []string
	for _, l := range strings.Split(string(data), "\n") {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func listAccounts() {
	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有已登录账号")
		return
	}
	for i, a := range accs {
		fmt.Printf("  %d. %s  @%s  %s %s\n", i+1, a.Phone, a.Username, a.FirstName, a.LastName)
	}
}

func cmdSetup() {
	banner()
	fmt.Println("=== 首次配置 ===")
	cfg := config.Load()

	id := readInput(fmt.Sprintf("api_id [%d]: ", cfg.APIID))
	if id != "" {
		cfg.APIID, _ = strconv.Atoi(id)
	}

	hash := readInput(fmt.Sprintf("api_hash [%s...]: ", cfg.APIHash[:min(8, len(cfg.APIHash))]))
	if hash != "" {
		cfg.APIHash = hash
	}

	if strings.ToLower(readInput("启用代理? (y/n): ")) == "y" {
		cfg.Proxy.Enabled = true
		cfg.Proxy.Type = readInput("代理类型 (socks5/http) [socks5]: ")
		if cfg.Proxy.Type == "" {
			cfg.Proxy.Type = "socks5"
		}
		cfg.Proxy.Host = readInput("代理地址 [127.0.0.1]: ")
		if cfg.Proxy.Host == "" {
			cfg.Proxy.Host = "127.0.0.1"
		}
		p := readInput("端口 [1080]: ")
		if p != "" {
			cfg.Proxy.Port, _ = strconv.Atoi(p)
		} else {
			cfg.Proxy.Port = 1080
		}
		cfg.Proxy.Username = readInput("用户名 (可选): ")
		if cfg.Proxy.Username != "" {
			cfg.Proxy.Password = readInput("密码: ")
		}
	}

	config.Save(cfg)
	fmt.Println("配置已保存!")
}

func cmdLogin() {
	phone := readInput("手机号 (+86xxx): ")
	if phone == "" {
		return
	}
	ctx := context.Background()

	codeFunc := func() string {
		return readInput("验证码: ")
	}
	passFunc := func() string {
		return readInput("二步验证密码: ")
	}

	client, err := session.Login(ctx, phone, codeFunc, passFunc)
	if err != nil {
		fmt.Printf("登录失败: %v\n", err)
		return
	}
	_ = client
	fmt.Printf("账号 %s 登录成功!\n", phone)
}

func cmdCollectQuery(args []string) {
	keyword := strings.Join(args, " ")
	if keyword == "" {
		keyword = readInput("关键词: ")
	}
	if keyword == "" {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	output := fmt.Sprintf("%s/groups_%s.txt", config.DataDir, keyword[:min(20, len(keyword))])
	col.CollectGroupsByKeyword(ctx, client, []string{keyword}, output, 0, true, "default")
	fmt.Printf("结果已保存到: %s\n", output)
}

func cmdCollectMembers(args []string) {
	groupLink := ""
	if len(args) > 0 {
		groupLink = args[0]
	} else {
		groupLink = readInput("群链接: ")
	}
	if groupLink == "" {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	output := fmt.Sprintf("%s/members_%s.txt", config.DataDir, strings.ReplaceAll(groupLink, "/", "_")[:30])
	col.CollectMembers(ctx, client, groupLink, output, 0, false, "default")
	fmt.Printf("结果已保存到: %s\n", output)
}

func cmdJoin(args []string) {
	filePath := ""
	if len(args) > 0 {
		filePath = args[0]
	} else {
		filePath = readInput("群链接文件: ")
	}
	links := readLines(filePath)
	if links == nil {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	mode := readInput("模式 (join/leave) [join]: ")
	if mode == "" {
		mode = "join"
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	snd.JoinGroups(ctx, client, links, mode, 30, 60, "default")
	fmt.Println("加群完成!")
}

func cmdSend(args []string) {
	filePath := ""
	if len(args) > 0 {
		filePath = args[0]
	} else {
		filePath = readInput("消息文件: ")
	}
	msgs := readLines(filePath)
	if msgs == nil {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	snd.SendToGroups(ctx, client, msgs, 10, 30, "", "default")
	fmt.Println("群发完成!")
}

func cmdDM(args []string) {
	targetFile, msgFile := "", ""
	if len(args) >= 2 {
		targetFile = args[0]
		msgFile = args[1]
	} else {
		targetFile = readInput("目标用户文件: ")
		msgFile = readInput("消息文件: ")
	}
	targets := readLines(targetFile)
	msgs := readLines(msgFile)
	if targets == nil || msgs == nil {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	maxPer := 30
	if s := readInput("单账号最大发送数 [30]: "); s != "" {
		maxPer, _ = strconv.Atoi(s)
	}

	snd.SendDM(ctx, client, targets, msgs, maxPer, 15, 30, "default")
}

func cmdInvite(args []string) {
	userFile, groupLink := "", ""
	if len(args) >= 2 {
		userFile = args[0]
		groupLink = args[1]
	} else {
		userFile = readInput("用户文件: ")
		groupLink = readInput("目标群链接: ")
	}
	users := readLines(userFile)
	if users == nil {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	snd.InviteUsers(ctx, client, users, groupLink, 50, 30, 60, "default")
}

func cmdFarm(args []string) {
	scriptFile, groupFile := "", ""
	if len(args) >= 2 {
		scriptFile = args[0]
		groupFile = args[1]
	} else {
		scriptFile = readInput("话术文件: ")
		groupFile = readInput("群链接文件: ")
	}
	scripts := readLines(scriptFile)
	groups := readLines(groupFile)
	if scripts == nil || groups == nil {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	farming.FarmGroups(ctx, client, groups, scripts, 5, 20, 0, "default")
}

func cmdFilter(args []string) {
	filePath := ""
	if len(args) > 0 {
		filePath = args[0]
	} else {
		filePath = readInput("手机号文件: ")
	}
	numbers := readLines(filePath)
	if numbers == nil {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	output := fmt.Sprintf("%s/filtered_%d.txt", config.DataDir, len(numbers))
	maxPer := 30
	if s := readInput("单账号最大筛选数 [30]: "); s != "" {
		maxPer, _ = strconv.Atoi(s)
	}

	filter.FilterPhones(ctx, client, numbers, output, maxPer, "default")
	fmt.Printf("结果已保存到: %s\n", output)
}

func cmdClone(args []string) {
	src, dst := "", ""
	if len(args) >= 2 {
		src = args[0]
		dst = args[1]
	} else {
		src = readInput("源频道 (@xxx): ")
		dst = readInput("目标频道 (@xxx): ")
	}
	if src == "" || dst == "" {
		return
	}

	accs := svc.List()
	if len(accs) == 0 {
		fmt.Println("没有可用账号")
		return
	}

	client, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil {
		fmt.Printf("连接失败: %v\n", err)
		return
	}

	limit := 0
	if s := readInput("最大消息数 (0=无限制) [0]: "); s != "" {
		limit, _ = strconv.Atoi(s)
	}

	mode := readInput("内容类型 (all/text/media) [all]: ")
	useForward := strings.ToLower(readInput("转发模式? (y/n) [y]: ")) != "n"

	cfg := clone.Config{
		Source:      src,
		Target:      dst,
		Limit:       limit,
		TextOnly:    mode == "text",
		MediaOnly:   mode == "media",
		Forward:     useForward,
		IntervalMin: 2,
		IntervalMax: 8,
	}

	clone.CloneChannel(ctx, client, cfg, "default")
}

func printHelp() {
	banner()
	fmt.Println("用法: tgtool <命令> [参数]")
	fmt.Println()
	fmt.Println("━━━ 账号管理 ━━━")
	fmt.Println("  setup          - 首次配置")
	fmt.Println("  login           - 登录账号")
	fmt.Println("  accounts        - 列出所有账号")
	fmt.Println("━━━ 数据采集 ━━━")
	fmt.Println("  collect-query <关键词>  - 关键词采集群组")
	fmt.Println("  collect-members <群链接> - 采集群成员")
	fmt.Println("━━━ 消息群发 ━━━")
	fmt.Println("  join      <群组文件>   - 批量加群")
	fmt.Println("  send      <消息文件>   - 群组群发")
	fmt.Println("  dm        <目标文件> <消息文件>  - 私信群发")
	fmt.Println("━━━ 其他功能 ━━━")
	fmt.Println("  invite    <用户文件> <群链接>  - 批量邀请")
	fmt.Println("  farm      <话术文件> <群链接>  - 炒群")
	fmt.Println("  filter    <手机号文件>         - 筛号")
	fmt.Println("  clone     <源频道> <目标频道>   - 复刻频道消息")
	fmt.Println("━━━ 系统 ━━━")
	fmt.Println("  config            - 查看配置")
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		printHelp()
	case "setup":
		cmdSetup()
	case "login":
		cmdLogin()
	case "accounts":
		listAccounts()
	case "collect-query":
		cmdCollectQuery(args)
	case "collect-members":
		cmdCollectMembers(args)
	case "join":
		cmdJoin(args)
	case "send":
		cmdSend(args)
	case "dm":
		cmdDM(args)
	case "invite":
		cmdInvite(args)
	case "farm":
		cmdFarm(args)
	case "filter":
		cmdFilter(args)
	case "clone":
		cmdClone(args)
	case "config":
		cfg := config.Load()
		cfg.APIHash = cfg.APIHash[:min(8, len(cfg.APIHash))] + "***"
		fmt.Printf("api_id: %d\napi_hash: %s\nproxy: %s://%s:%d\n",
			cfg.APIID, cfg.APIHash, cfg.Proxy.Type, cfg.Proxy.Host, cfg.Proxy.Port)
	default:
		fmt.Printf("未知命令: %s\n", cmd)
		printHelp()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
