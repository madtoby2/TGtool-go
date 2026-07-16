package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/gotd/td/telegram"

	"github.com/madtoby2/tgtool/internal/clone"
	"github.com/madtoby2/tgtool/internal/collector"
	"github.com/madtoby2/tgtool/internal/config"
	"github.com/madtoby2/tgtool/internal/farming"
	"github.com/madtoby2/tgtool/internal/filter"
	"github.com/madtoby2/tgtool/internal/sender"
	"github.com/madtoby2/tgtool/internal/session"
)

var ctx = context.Background()

// ─── palette ────────────────────────────────────────────────────────
var (
	pink   = lipgloss.Color("#FF6B9D")
	cyan   = lipgloss.Color("#00D9FF")
	purple = lipgloss.Color("#A855F7")
	green  = lipgloss.Color("#22C55E")
	amber  = lipgloss.Color("#F59E0B")
	red    = lipgloss.Color("#EF4444")
	gray   = lipgloss.Color("#6B7280")
)

var (
	svc = session.NewManager()
	col = collector.New()
	snd = sender.New()
)

func main() {
	if len(os.Args) < 2 { showMenu(); return }
	switch os.Args[1] {
	case "setup": runSetup()
	case "login": runLogin()
	case "accounts": runAccounts()
	case "config": runConfig()
	case "collect-query": runCollectQuery(os.Args[2:])
	case "collect-members": runCollectMembers(os.Args[2:])
	case "join": runJoin(os.Args[2:])
	case "send": runSend(os.Args[2:])
	case "dm": runDM(os.Args[2:])
	case "invite": runInvite(os.Args[2:])
	case "farm": runFarm(os.Args[2:])
	case "filter": runFilter(os.Args[2:])
	case "clone": runClone(os.Args[2:])
	default: showMenu()
	}
}

// ─── helpers ────────────────────────────────────────────────────────
func header() string {
	name := lipgloss.NewStyle().Bold(true).Foreground(cyan).Render("▎ MADTOBY")
	sub  := lipgloss.NewStyle().Foreground(gray).Render("telegram automation toolkit")
	b := lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true, false).
		BorderForeground(pink).Padding(0, 1)
	return b.Render(name + "\n" + sub)
}
func ok(s string)   { fmt.Println(lipgloss.NewStyle().Foreground(green).Render("✓ " + s)) }
func fail(s string) { fmt.Println(lipgloss.NewStyle().Foreground(red).Render("✗ " + s)) }
func info(s string) { fmt.Println(lipgloss.NewStyle().Foreground(amber).Render("→ " + s)) }
func ask(label string) string {
	var v string
	huh.NewInput().Title(label).Value(&v).Run()
	return strings.TrimSpace(v)
}
func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil { fail("file not found: " + path); return nil }
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" { out = append(out, l) }
	}
	return out
}
func getClient() *telegram.Client {
	accs := svc.List()
	if len(accs) == 0 { fail("no accounts — run login first"); return nil }
	c, err := session.Login(ctx, accs[0].Phone, nil, nil)
	if err != nil { fail("connect failed: " + err.Error()); return nil }
	return c
}
func sanitize(s string) string {
	r := strings.NewReplacer("@","","/","_","\\","_",":","_"," ","_","?","")
	s = r.Replace(s)
	if len(s) > 30 { s = s[:30] }
	return s
}

// ─── menu ───────────────────────────────────────────────────────────
func showMenu() {
	fmt.Println(header())
	sections := []struct{ title string; items []string }{
		{"Account", []string{
			"  setup     configure api credentials",
			"  login     login telegram account",
			"  accounts  list all accounts",
		}},
		{"Collection", []string{
			"  collect-query      search groups by keyword",
			"  collect-members    scrape group members",
		}},
		{"Messaging", []string{
			"  join    batch join / leave groups",
			"  send    mass message to groups",
			"  dm      mass private messages",
		}},
		{"Tools", []string{
			"  invite  invite users to group",
			"  farm    simulate group activity",
			"  filter  phone number screening",
			"  clone   mirror channel content",
		}},
	}
	for _, s := range sections {
		title := lipgloss.NewStyle().Bold(true).Foreground(cyan).Render(s.title)
		body  := strings.Join(s.items, "\n")
		box   := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(purple).Padding(1, 2).Width(50).
			Render(title + "\n\n" + body)
		fmt.Println(box)
		fmt.Println()
	}
	fmt.Println(lipgloss.NewStyle().Foreground(gray).Render("tgtool v2.0  •  madtoby"))
}

// ─── commands ───────────────────────────────────────────────────────
func runSetup() {
	fmt.Println(header())
	cfg := config.Load()
	if s := ask("API ID"); s != "" { cfg.APIID, _ = strconv.Atoi(s) }
	if s := ask("API Hash"); s != "" { cfg.APIHash = s }
	if strings.ToLower(ask("Enable proxy? (y/n)")) == "y" {
		cfg.Proxy.Enabled = true
		cfg.Proxy.Type = ask("Proxy type (socks5/http)")
		if cfg.Proxy.Type == "" { cfg.Proxy.Type = "socks5" }
		cfg.Proxy.Host = ask("Proxy host")
		cfg.Proxy.Port, _ = strconv.Atoi(ask("Proxy port"))
	}
	if err := config.Save(cfg); err != nil { fail("save failed"); return }
	ok("configuration saved")
}

func runLogin() {
	phone := ask("Phone number (+86xxx)")
	if phone == "" { fail("phone required"); return }
	ok("sending code to " + phone)
	if _, err := session.Login(ctx, phone,
		func() string { return ask("Verification code") },
		func() string { return ask("2FA password (if any)") },
	); err != nil { fail("login failed: " + err.Error()); return }
	ok("logged in: " + phone)
}

func runAccounts() {
	fmt.Println(header())
	for _, a := range svc.List() {
		dot := lipgloss.NewStyle().Foreground(green).Render("●")
		if !a.IsActive { dot = lipgloss.NewStyle().Foreground(red).Render("○") }
		fmt.Printf("  %s  %-20s  @%-15s  %s %s\n",
			dot, a.Phone, a.Username, a.FirstName, a.LastName)
	}
}

func runConfig() {
	cfg := config.Load()
	h := cfg.APIHash
	if len(h) > 8 { h = h[:8] + "***" }
	fmt.Println(header())
	fmt.Printf("  api_id:  %d\n  api_hash: %s\n  proxy:   %s://%s:%d\n",
		cfg.APIID, h, cfg.Proxy.Type, cfg.Proxy.Host, cfg.Proxy.Port)
}

func runCollectQuery(args []string) {
	kw := strings.Join(args, " ")
	if kw == "" { kw = ask("Search keyword") }
	if kw == "" { return }
	c := getClient(); if c == nil { return }
	out := config.DataDir + "/groups_" + sanitize(kw) + ".txt"
	info("searching: " + kw)
	col.CollectGroupsByKeyword(ctx, c, []string{kw}, out, 0, true, "default")
	ok("saved → " + out)
}

func runCollectMembers(args []string) {
	link := ""; if len(args) > 0 { link = args[0] } else { link = ask("Group @username") }
	if link == "" { return }
	c := getClient(); if c == nil { return }
	out := config.DataDir + "/members_" + sanitize(link) + ".txt"
	info("collecting from " + link)
	col.CollectMembers(ctx, c, link, out, 0, false, "default")
	ok("saved → " + out)
}

func runJoin(args []string) {
	path := ""; if len(args) > 0 { path = args[0] } else { path = ask("Group links file") }
	links := readLines(path); if links == nil { return }
	c := getClient(); if c == nil { return }
	mode := ask("Mode (join/leave)"); if mode == "" { mode = "join" }
	snd.JoinGroups(ctx, c, links, mode, 30, 60, "default")
	ok("done")
}

func runSend(args []string) {
	path := ""; if len(args) > 0 { path = args[0] } else { path = ask("Messages file") }
	msgs := readLines(path); if msgs == nil { return }
	c := getClient(); if c == nil { return }
	info("sending to joined groups...")
	snd.SendToGroups(ctx, c, msgs, 10, 30, "", "default")
}

func runDM(args []string) {
	tf, mf := "", ""
	if len(args) >= 2 { tf, mf = args[0], args[1] } else {
		tf = ask("Target users file"); mf = ask("Messages file")
	}
	targets, msgs := readLines(tf), readLines(mf)
	if targets == nil || msgs == nil { return }
	c := getClient(); if c == nil { return }
	limit := 30
	if s := ask("Max per account"); s != "" { limit, _ = strconv.Atoi(s) }
	snd.SendDM(ctx, c, targets, msgs, limit, 15, 30, "default")
}

func runInvite(args []string) {
	uf, gl := "", ""
	if len(args) >= 2 { uf, gl = args[0], args[1] } else {
		uf = ask("Users file"); gl = ask("Target group @username")
	}
	users := readLines(uf); if users == nil { return }
	c := getClient(); if c == nil { return }
	snd.InviteUsers(ctx, c, users, gl, 50, 30, 60, "default")
}

func runFarm(args []string) {
	sf, gf := "", ""
	if len(args) >= 2 { sf, gf = args[0], args[1] } else {
		sf = ask("Script file"); gf = ask("Group links file")
	}
	scripts, groups := readLines(sf), readLines(gf)
	if scripts == nil || groups == nil { return }
	c := getClient(); if c == nil { return }
	farming.FarmGroups(ctx, c, groups, scripts, 5, 20, 0, "default")
}

func runFilter(args []string) {
	path := ""; if len(args) > 0 { path = args[0] } else { path = ask("Phone numbers file") }
	nums := readLines(path); if nums == nil { return }
	c := getClient(); if c == nil { return }
	out := fmt.Sprintf("%s/filtered_%d.txt", config.DataDir, len(nums))
	limit := 30
	if s := ask("Max per account"); s != "" { limit, _ = strconv.Atoi(s) }
	filter.FilterPhones(ctx, c, nums, out, limit, "default")
	ok("saved → " + out)
}

func runClone(args []string) {
	src, dst := "", ""
	if len(args) >= 2 { src, dst = args[0], args[1] } else {
		src = ask("Source channel (@username)")
		dst = ask("Target channel (@username)")
	}
	if src == "" || dst == "" { return }
	c := getClient(); if c == nil { return }
	limit := 0
	if s := ask("Max messages (0=unlimited)"); s != "" { limit, _ = strconv.Atoi(s) }
	mode := ask("Content type (all/text/media)"); if mode == "" { mode = "all" }
	forward := strings.ToLower(ask("Forward mode? (y/n)")) != "n"
	cfg := clone.Config{Source: src, Target: dst, Limit: limit,
		TextOnly: mode == "text", MediaOnly: mode == "media",
		Forward: forward, IntervalMin: 2, IntervalMax: 8}
	clone.CloneChannel(ctx, c, cfg, "default")
}
