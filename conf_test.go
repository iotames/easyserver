package easyconf

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestConfLine 测试 GetConfStrByLine 行解析函数对各种 KEY=VALUE 格式的解析正确性，
// 包括带引号、含 # 注释、等号在值内部等边界情况。
func TestConfLine(t *testing.T) {
	lines := []string{
		`NAME0 =VALUE0`,
		`NAME1=VALUE1`,
		`NAME2= VALUE2`,
		`NAME3  = 'VALUE3'`,
		`NAME4 =  "VALUE4"`,
		`NAME5= 'VALUE5'`,
		`NAME6 ="VALUE6"`,
		`NAME7 ="VALUE7" # 备注1`,
		`NAME8 = 'VALUE8' # 备注2`,
		`NAME9 = "NAME9=VALUE9" # 备注3`,
		`NAME10='NAME10=VALUE10' # 备注4`,
		`NAME11 =NAME11=VALUE11 # 备注5`,
		`NAME12 ="NAME12#VALUE12" # 备注6`,
		`NAME13="NAME13#VALUE13" # remark="a=b"`,
	}
	for i, line := range lines {
		okval := ""
		k, v := GetConfStrByLine(line)
		if i < 9 {
			okval = fmt.Sprintf(`VALUE%d`, i)
		}
		if i >= 9 && i <= 11 {
			okval = fmt.Sprintf("NAME%d=VALUE%d", i, i)
		}
		if i >= 12 {
			okval = fmt.Sprintf("NAME%d#VALUE%d", i, i)
		}
		if k != fmt.Sprintf(`NAME%d`, i) || v != okval {
			t.Fatal(fmt.Errorf("value(%s) err for %s", v, okval))
		}
	}
}

// TestConf 测试配置的完整生命周期：默认值创建 → Parse 解析 → UpdateFile 更新写入 → 重新 Parse 验证持久化。
func TestConf(t *testing.T) {
	cf := NewConf()
	var version string
	var isbool1, isbool2 bool
	var webport int
	var domains []string
	var intlist []int
	var costDays float64
	domainUsage := []string{
		"1. 允许直连的域名放在 routing.rules 数组中",
		"2. 当路由匹配到一个规则时就会跳出匹配而不会对之后的规则进行匹配；",
	}
	version = "v1.0.1"
	const DEFAULT_VERSION = "v1.0.0"
	const DEFAULT_WEBPORT = 8080
	var DEFAULT_DOMAINS = []string{"baidu.com", "taobao.com"}
	var DEFAULT_INTLIST = []int{2, 4, 6, 8}
	var DEFAULT_COST_DAYS = 3.25
	cf.StringVar(&version, "VERSION", DEFAULT_VERSION, "版本号")
	cf.BoolVar(&isbool1, "IS_BOOL1", false, "默认关闭")
	cf.BoolVar(&isbool2, "IS_BOOL2", true, "默认开启")
	cf.IntVar(&webport, "WEB_PORT", DEFAULT_WEBPORT, "web服务器端口")
	cf.AddComment("这个是注释说明", "继续添加注释说明A")
	cf.StringListVar(&domains, "DOMAINS", DEFAULT_DOMAINS, "允许的域名列表", domainUsage...)
	cf.IntListVar(&intlist, "INTLIST", DEFAULT_INTLIST, "整数列表")
	cf.Float64Var(&costDays, "COST_DAYS", DEFAULT_COST_DAYS, "耗时天数", "完成工期花费的时间/天")
	webport = 8888
	err := cf.Parse(false)
	if err != nil {
		t.Fatal(err)
	}

	// 验证默认值
	if version != DEFAULT_VERSION || isbool1 || !isbool2 || webport != DEFAULT_WEBPORT {
		t.Fatal(fmt.Errorf(`默认值设置错误isbool1(%t)--isbool2(%t)`, isbool1, isbool2))
	}
	if costDays != DEFAULT_COST_DAYS {
		t.Fatal(fmt.Errorf("COST_DAYS err--val(%v)---default(%v)", costDays, DEFAULT_COST_DAYS))
	}

	for i, d := range DEFAULT_DOMAINS {
		if d != domains[i] {
			t.Fatal("[]string 默认值设置错误")
		}
	}

	for i, d := range DEFAULT_INTLIST {
		if d != intlist[i] {
			t.Fatal("[]int 默认值设置错误")
		}
	}

	// t.Logf("---111--VERSION(%s)--IS_BOOL1(%t)--WEB_PORT(%d)--DOMAINS(%v)---\n", version, isbool1, webport, domains)

	// 更新测试
	webport = 8899
	version = "v1.99.9"
	isbool1 = true
	isbool2 = false
	domains = []string{"amazon.com", "google.com"}
	intlist = []int{1, 3, 7}
	costDays = 6.79
	err = cf.UpdateFile("")
	if err != nil {
		t.Fatal(err)
	}
	err = cf.UpdateFile("update.env")
	if err != nil {
		t.Fatal(err)
	}

	// 验证更新
	updatedWebport := webport
	updatedVersion := version
	updatedDomains := domains
	updatedIntlist := intlist
	updateCostDays := costDays

	cf = NewConf()
	cf.StringVar(&version, "VERSION", DEFAULT_VERSION, "版本号")
	cf.BoolVar(&isbool1, "IS_BOOL1", false, "默认关闭")
	cf.BoolVar(&isbool2, "IS_BOOL2", true, "默认开启")
	cf.IntVar(&webport, "WEB_PORT", DEFAULT_WEBPORT, "web服务器端口")
	cf.StringListVar(&domains, "DOMAINS", DEFAULT_DOMAINS, "允许的域名列表", domainUsage...)
	cf.IntListVar(&intlist, "INTLIST", DEFAULT_INTLIST, "整数列表")
	cf.Float64Var(&costDays, "COST_DAYS", DEFAULT_COST_DAYS, "耗时天数", "完成工期花费的时间/天")
	webport = 8888
	err = cf.Parse(false)
	if err != nil {
		t.Fatal(err)
	}
	if version != updatedVersion || !isbool1 || isbool2 || webport != updatedWebport {
		t.Fatal(fmt.Errorf(`配置更新验证失败isbool1(%t)--isbool2(%t)`, isbool1, isbool2))
	}
	if costDays != updateCostDays {
		t.Fatal(fmt.Errorf("COST_DAYS err--costDays-val(%v)---expected(%v)", costDays, updateCostDays))
	}
	for i, d := range updatedDomains {
		if d != domains[i] {
			t.Fatal("[]string 更新值设置错误")
		}
	}
	for i, d := range updatedIntlist {
		if d != intlist[i] {
			t.Fatal("[]int 更新值设置错误")
		}
	}

	// 验证默认值

	// t.Logf("--222--VERSION(%s)--IS_BOOL1(%t)--WEB_PORT(%d)--DOMAINS(%v)---\n", version, isbool1, webport, domains)
	f, err := os.OpenFile(".env", os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("")
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}

	cf = NewConf()
	cf.StringVar(&version, "VERSION", DEFAULT_VERSION, "版本号")
	cf.BoolVar(&isbool1, "IS_BOOL1", false, "默认关闭")
	cf.BoolVar(&isbool2, "IS_BOOL2", true, "默认开启")
	cf.IntVar(&webport, "WEB_PORT", DEFAULT_WEBPORT, "web服务器端口")
	cf.StringListVar(&domains, "DOMAINS", DEFAULT_DOMAINS, "允许的域名列表", domainUsage...)
	cf.IntListVar(&intlist, "INTLIST", DEFAULT_INTLIST, "整数列表")
	err = cf.Parse(false)
	if err != nil {
		t.Fatal(err)
	}
	// 验证默认值
	if version != DEFAULT_VERSION || isbool1 || !isbool2 || webport != DEFAULT_WEBPORT {
		t.Fatal(fmt.Errorf(`默认值设置错误isbool1(%t)--isbool2(%t)`, isbool1, isbool2))
	}
	for i, d := range DEFAULT_DOMAINS {
		if d != domains[i] {
			t.Fatal("[]string 默认值设置错误")
		}
	}
	for i, d := range DEFAULT_INTLIST {
		if d != intlist[i] {
			t.Fatal("[]int 默认值设置错误")
		}
	}
	os.Remove(".env")
	os.Remove("default.env")
}

// TestUpdateFilePreserveComments 验证 UpdateFile 更新配置值时，不会丢失原文件中的各类注释：
// 用户手写注释、ConfItem 自动生成的标题/默认值/扩展注释、独立注释块，以及所有类型的值是否正确更新。
func TestUpdateFilePreserveComments(t *testing.T) {
	const testFile = "test_update_preserve.env"
	originalContent := `# 这是用户手写注释，应该在更新后保留
# 数据库主机地址
# The default value is: "127.0.0.1"
DB_HOST = "127.0.0.1"

# 数据库地址端口号
# The default value is: 3306
DB_PORT = 3306

# 是否启用数据库
# The default value is: false
DB_ENABLE = false

# 这是紧邻键值对的注释行
# 允许访问的IP列表
# The default value is: 192.168.1.6,192.168.2.8
ALLOW_IP_LIST = 192.168.1.6,192.168.2.8

# 这是一段独立的注释，不关联任何 key
# 第二行独立注释，也应保留

# 年龄范围
# The default value is: 3,6
# 填写2个正整数,中间用逗号,隔开
AGE_RANGE = 3,6
`

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	var dbPort int
	var dbEnable bool
	var allowIPs []string
	var ageRange []int

	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "数据库主机地址")
	cf.IntVar(&dbPort, "DB_PORT", 3306, "数据库地址端口号")
	cf.BoolVar(&dbEnable, "DB_ENABLE", false, "是否启用数据库")
	cf.StringListVar(&allowIPs, "ALLOW_IP_LIST", []string{"192.168.1.6", "192.168.2.8"}, "允许访问的IP列表")
	cf.IntListVar(&ageRange, "AGE_RANGE", []int{3, 6}, "年龄范围", "填写2个正整数,中间用逗号,隔开")

	err = cf.Parse(false)
	if err != nil {
		t.Fatal(err)
	}

	dbHost = "192.168.1.99"
	dbPort = 5432
	dbEnable = true
	allowIPs = []string{"10.0.0.1", "10.0.0.2"}
	ageRange = []int{18, 60}

	err = cf.UpdateFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	updatedContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	content := string(updatedContent)

	assertContains := func(haystack, needle, desc string) {
		if !strings.Contains(haystack, needle) {
			t.Fatalf("%s: 应包含 %q，实际未找到", desc, needle)
		}
	}

	assertContains(content, "# 这是用户手写注释，应该在更新后保留", "用户手写注释（文件顶部）")
	assertContains(content, "# 这是一段独立的注释，不关联任何 key", "独立注释块（首行）")
	assertContains(content, "# 第二行独立注释，也应保留", "独立注释块（第二行）")
	assertContains(content, "# 这是紧邻键值对的注释行", "键值对前的手写注释")

	assertContains(content, "# 数据库主机地址", "DB_HOST 标题注释")
	assertContains(content, `# The default value is: "127.0.0.1"`, "DB_HOST 默认值注释")
	assertContains(content, `DB_HOST = 192.168.1.99`, "DB_HOST 更新后的值")

	assertContains(content, "# 数据库地址端口号", "DB_PORT 标题注释")
	assertContains(content, "# The default value is: 3306", "DB_PORT 默认值注释")
	assertContains(content, "DB_PORT = 5432", "DB_PORT 更新后的值")

	assertContains(content, "# 是否启用数据库", "DB_ENABLE 标题注释")
	assertContains(content, "# The default value is: false", "DB_ENABLE 默认值注释")
	assertContains(content, "DB_ENABLE = true", "DB_ENABLE 更新后的值")

	assertContains(content, "# 允许访问的IP列表", "ALLOW_IP_LIST 标题注释")
	assertContains(content, "# The default value is: 192.168.1.6,192.168.2.8", "ALLOW_IP_LIST 默认值注释")
	assertContains(content, "ALLOW_IP_LIST = 10.0.0.1,10.0.0.2", "ALLOW_IP_LIST 更新后的值")

	assertContains(content, "# 年龄范围", "AGE_RANGE 标题注释")
	assertContains(content, "# The default value is: 3,6", "AGE_RANGE 默认值注释")
	assertContains(content, "# 填写2个正整数,中间用逗号,隔开", "AGE_RANGE 扩展注释")
	assertContains(content, "AGE_RANGE = 18,60", "AGE_RANGE 更新后的值")

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if line == "" {
			continue
		}
		key, _ := GetConfStrByLine(line)
		if key == "" {
			continue
		}
		switch key {
		case "DB_HOST":
			if !strings.Contains(line, "192.168.1.99") {
				t.Fatalf("DB_HOST 值不正确: %s", line)
			}
		case "DB_PORT":
			if !strings.Contains(line, "5432") {
				t.Fatalf("DB_PORT 值不正确: %s", line)
			}
		case "DB_ENABLE":
			if !strings.Contains(line, "true") {
				t.Fatalf("DB_ENABLE 值不正确: %s", line)
			}
		case "ALLOW_IP_LIST":
			if !strings.Contains(line, "10.0.0.1,10.0.0.2") {
				t.Fatalf("ALLOW_IP_LIST 值不正确: %s", line)
			}
		case "AGE_RANGE":
			if !strings.Contains(line, "18,60") {
				t.Fatalf("AGE_RANGE 值不正确: %s", line)
			}
		default:
			t.Fatalf("文件中包含未预期的配置 key: %s", key)
		}
	}
}

// TestUpdateFileAppendNewKey 验证 UpdateFile 对原文件中不存在的配置项，会追加到末尾并携带完整的注释（标题+默认值）。
func TestUpdateFileAppendNewKey(t *testing.T) {
	const testFile = "test_update_newkey.env"
	originalContent := `# 数据库主机地址
# The default value is: "127.0.0.1"
DB_HOST = "127.0.0.1"
`

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	var dbPort int

	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "数据库主机地址")
	cf.IntVar(&dbPort, "DB_PORT", 3306, "数据库地址端口号")

	err = cf.Parse(false)
	if err != nil {
		t.Fatal(err)
	}

	dbHost = "10.0.0.1"
	dbPort = 9999

	err = cf.UpdateFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "DB_HOST = 10.0.0.1") {
		t.Fatalf("DB_HOST 应更新为新值，实际内容:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "# 数据库主机地址") {
		t.Fatal("DB_HOST 原有注释应保留")
	}
	if !strings.Contains(contentStr, "# 数据库地址端口号") {
		t.Fatal("新增 key DB_PORT 应包含标题注释")
	}
	if !strings.Contains(contentStr, "# The default value is: 3306") {
		t.Fatal("新增 key DB_PORT 应包含默认值注释")
	}
	if !strings.Contains(contentStr, "DB_PORT = 9999") {
		t.Fatal("新增 key DB_PORT 值不正确")
	}
}

// TestUpdateFileEmptyFile 验证 UpdateFile 对空文件也能正常写入配置项及其注释。
func TestUpdateFileEmptyFile(t *testing.T) {
	const testFile = "test_update_empty.env"

	f, err := os.Create(testFile)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbPort int
	cf.IntVar(&dbPort, "DB_PORT", 3306, "数据库地址端口号")

	err = cf.Parse(false)
	if err != nil {
		t.Fatal(err)
	}

	dbPort = 8888
	err = cf.UpdateFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "DB_PORT = 8888") {
		t.Fatal("空文件更新后应包含 DB_PORT 配置项")
	}
	if !strings.Contains(contentStr, "# 数据库地址端口号") {
		t.Fatal("空文件更新后应包含 DB_PORT 标题注释")
	}
}
