package easyconf

import (
	"os"
	"strings"
	"testing"
)

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
	assertContains(content, `DB_HOST = "192.168.1.99"`, "DB_HOST 更新后的值")

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

	if !strings.Contains(contentStr, `DB_HOST = "10.0.0.1"`) {
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

// TestUpdateFilePreserveInlineComments 验证 UpdateFile 不会丢弃键值对后面的行内注释（inline comment）。
func TestUpdateFilePreserveInlineComments(t *testing.T) {
	const testFile = "test_inline_comment.env"
	originalContent := `# 数据库配置
DB_HOST = 127.0.0.1 # 生产环境主机
DB_PORT = 3306 # 默认 MySQL 端口
`

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	var dbPort int
	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "")
	cf.IntVar(&dbPort, "DB_PORT", 3306, "")

	if err = cf.Parse(false); err != nil {
		t.Fatal(err)
	}

	dbHost = "10.0.0.1"
	dbPort = 5432
	if err = cf.UpdateFile(testFile); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	if !strings.Contains(contentStr, "# 生产环境主机") {
		t.Fatalf("行内注释应保留，实际内容:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "# 默认 MySQL 端口") {
		t.Fatalf("行内注释应保留，实际内容:\n%s", contentStr)
	}
	// 值已更新，且行内注释仍在同一行
	if !strings.Contains(contentStr, "10.0.0.1 #") {
		t.Fatalf("值更新后行内注释应在同一行，实际内容:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "5432 #") {
		t.Fatalf("值更新后行内注释应在同一行，实际内容:\n%s", contentStr)
	}
}

// TestUpdateFilePreserveStringQuotes 验证 UpdateFile 前后，字符串类型的值保持引号风格一致。
func TestUpdateFilePreserveStringQuotes(t *testing.T) {
	const testFile = "test_string_quotes.env"
	// 原始文件：部分带引号，部分不带
	originalContent := `HOST_WITH_QUOTES = "127.0.0.1"
HOST_WITHOUT_QUOTES = 192.168.1.1
`

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var hostQuoted string
	var hostBare string
	cf.StringVar(&hostQuoted, "HOST_WITH_QUOTES", "127.0.0.1", "")
	cf.StringVar(&hostBare, "HOST_WITHOUT_QUOTES", "192.168.1.1", "")

	if err = cf.Parse(false); err != nil {
		t.Fatal(err)
	}

	hostQuoted = "10.0.0.1"
	hostBare = "172.16.0.1"
	if err = cf.UpdateFile(testFile); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// 原来带引号的应保持引号
	if !strings.Contains(contentStr, `HOST_WITH_QUOTES = "10.0.0.1"`) {
		t.Fatalf("带引号的原值应保持引号，实际内容:\n%s", contentStr)
	}
	// 原来不带引号的应保持无引号
	if !strings.Contains(contentStr, "HOST_WITHOUT_QUOTES = 172.16.0.1") {
		t.Fatalf("不带引号的原值应保持无引号，实际内容:\n%s", contentStr)
	}
}

// TestUpdateFileFileNotExist 验证 UpdateFile 在文件不存在时自动创建文件并写入配置。
func TestUpdateFileFileNotExist(t *testing.T) {
	const testFile = "test_not_exist_create.env"
	// 确保文件不存在
	os.Remove(testFile)
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "数据库主机地址")

	// Parse 会创建文件，跳过它以模拟文件不存在的场景
	// 直接调用 UpdateFile 期望创建新文件（而非报错）
	err := cf.UpdateFile(testFile)
	if err != nil {
		t.Fatalf("UpdateFile 应自动创建不存在的文件，但返回错误: %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("文件应已被创建，但读取失败: %v", err)
	}
	contentStr := string(content)
	if !strings.Contains(contentStr, `DB_HOST = "127.0.0.1"`) {
		t.Fatalf("新创建的文件应包含配置项，实际内容:\n%s", contentStr)
	}
}

// TestUpdateFilePreserveFloatFormat 验证 UpdateFile 更新浮点数时，不引入多余的尾部零。
func TestUpdateFilePreserveFloatFormat(t *testing.T) {
	const testFile = "test_float_format.env"
	originalContent := `PI = 3.14
`

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var pi float64
	cf.Float64Var(&pi, "PI", 3.14, "圆周率")

	if err = cf.Parse(false); err != nil {
		t.Fatal(err)
	}

	pi = 3.14159
	if err = cf.UpdateFile(testFile); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// 期望保持原始精度，而不是变为 3.141590
	// 检查 PI 所在行是否包含尾部多余零
	for _, line := range strings.Split(contentStr, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "PI") {
			if strings.Contains(line, "3.141590") {
				t.Fatalf("浮点数不应有尾部多余零，实际行: %q", line)
			}
			if !strings.Contains(line, "3.14159") {
				t.Fatalf("浮点数应包含 3.14159，实际行: %q", line)
			}
			return
		}
	}
	t.Fatal("未找到 PI 配置行")
}

// TestUpdateFilePreserveBlankLines 验证 UpdateFile 前后，配置项之间的空行被保留。
func TestUpdateFilePreserveBlankLines(t *testing.T) {
	const testFile = "test_blank_lines.env"
	originalContent := `DB_HOST = 127.0.0.1


DB_PORT = 3306
`

	err := os.WriteFile(testFile, []byte(originalContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	var dbPort int
	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "")
	cf.IntVar(&dbPort, "DB_PORT", 3306, "")

	if err = cf.Parse(false); err != nil {
		t.Fatal(err)
	}

	dbHost = "10.0.0.1"
	dbPort = 5432
	if err = cf.UpdateFile(testFile); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	contentStr := string(content)

	// 应该有至少 2 个空行分隔两个配置项（原文件有 2 个空行）
	// 验证 DB_HOST 和 DB_PORT 之间有 2 个连续换行
	if !strings.Contains(contentStr, "10.0.0.1\n\n\nDB_PORT") &&
		!strings.Contains(contentStr, "10.0.0.1\r\n\r\n\r\nDB_PORT") {
		t.Fatalf("两个配置项之间的空行数目应保留，实际内容:\n%q", contentStr)
	}
}

// TestSetItemValue 验证 SetItemValue 能正确更新配置项的值。
func TestSetItemValue(t *testing.T) {
	const testFile = "test_setitem.env"
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	var dbPort int
	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "数据库主机地址")
	cf.IntVar(&dbPort, "DB_PORT", 3306, "数据库地址端口号")

	// 正常更新
	if err := cf.SetItemValue("DB_HOST", "10.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if dbHost != "10.0.0.1" {
		t.Fatalf("DB_HOST 应为 10.0.0.1，实际为 %s", dbHost)
	}

	// 空值允许
	if err := cf.SetItemValue("DB_HOST", ""); err != nil {
		t.Fatal(err)
	}
	if dbHost != "" {
		t.Fatalf("DB_HOST 应为空，实际为 %s", dbHost)
	}

	// 空键名应报错
	if err := cf.SetItemValue("", "foo"); err == nil {
		t.Fatal("空键名应返回错误")
	}

	// 不存在的键名不报错（静默跳过）
	if err := cf.SetItemValue("NOT_EXIST", "bar"); err != nil {
		t.Fatalf("不存在的键名应静默跳过，但返回错误: %v", err)
	}
}

// TestUpdateByMap 验证 UpdateByMap 批量更新并持久化到文件。
func TestUpdateByMap(t *testing.T) {
	const testFile = "test_updatemap.env"
	originalContent := `DB_HOST = 127.0.0.1
DB_PORT = 3306
`
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	var dbPort int
	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "")
	cf.IntVar(&dbPort, "DB_PORT", 3306, "")

	if err := cf.Parse(false); err != nil {
		t.Fatal(err)
	}

	// 批量更新
	if err := cf.UpdateByMap(map[string]string{
		"DB_HOST": "10.0.0.1",
		"DB_PORT": "5432",
	}, testFile); err != nil {
		t.Fatal(err)
	}

	// 验证内存值已更新
	if dbHost != "10.0.0.1" {
		t.Fatalf("DB_HOST 应为 10.0.0.1，实际为 %s", dbHost)
	}
	if dbPort != 5432 {
		t.Fatalf("DB_PORT 应为 5432，实际为 %d", dbPort)
	}

	// 验证文件已持久化
	content, _ := os.ReadFile(testFile)
	contentStr := string(content)
	if !strings.Contains(contentStr, "DB_HOST = 10.0.0.1") {
		t.Fatalf("文件应包含 DB_HOST 新值，实际内容:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "DB_PORT = 5432") {
		t.Fatalf("文件应包含 DB_PORT 新值，实际内容:\n%s", contentStr)
	}
}

// TestUpdateByMapEdgeCases 验证 UpdateByMap 的边界情况：nil map 和空 map。
func TestUpdateByMapEdgeCases(t *testing.T) {
	const testFile = "test_updatemap_edge.env"
	originalContent := `DB_HOST = 127.0.0.1
`
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	cf := NewConf(testFile)
	var dbHost string
	cf.StringVar(&dbHost, "DB_HOST", "127.0.0.1", "")

	if err := cf.Parse(false); err != nil {
		t.Fatal(err)
	}

	// nil map 应无操作、不报错
	if err := cf.UpdateByMap(nil, testFile); err != nil {
		t.Fatalf("nil map 不应返回错误: %v", err)
	}
	if dbHost != "127.0.0.1" {
		t.Fatalf("nil map 后 DB_HOST 不应变化，实际为 %s", dbHost)
	}

	// 空 map 应无操作、不报错
	if err := cf.UpdateByMap(map[string]string{}, testFile); err != nil {
		t.Fatalf("空 map 不应返回错误: %v", err)
	}
	if dbHost != "127.0.0.1" {
		t.Fatalf("空 map 后 DB_HOST 不应变化，实际为 %s", dbHost)
	}
}

