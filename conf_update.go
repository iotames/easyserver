package easyconf

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// AddComment 添加纯注释行到配置输出中。
// title 为注释标题（生成为 # title），comment 为多行注释内容。
// 这些注释行无关联的配置键名，在解析时会被跳过。
func (cf *Conf) AddComment(title string, comment ...string) {
	cf.addItem(nil, "", nil, title, comment...)
}

func (cf *Conf) addItem(pval any, name string, defval any, title string, usage ...string) {
	cf.items = append(cf.items, newConfItem(pval, name, defval, title, usage...))
}

// setItemVar 设置配置项的值。
// 允许设置值为空字符串。
func (cf *Conf) setItemVar(k, v string) error {
	var err error
	if k == "" {
		return fmt.Errorf("配置项的键不能为空")
	}
	for _, item := range cf.items {
		if item.Name == k {
			err1 := item.setValueVar(v)
			if err1 != nil {
				err = err1
			} else {
				item.ValueStr = v
			}
		}
	}
	return err
}

// SetItemValue 设置指定配置项的值，并更新指针指向的变量。键名 k 不可为空，值 v 允许为空字符串。
func (cf *Conf) SetItemValue(k, v string) error {
	return cf.setItemVar(k, v)
}

// SetValuesByCmdArgs 从命令行参数获取配置。优先级高
// 允许设置值为空字符串。
func (cf *Conf) SetValuesByCmdArgs() []error {
	var errs []error
	for _, item := range cf.items {
		// 注释语句 Name 为空字符
		if item.Name != "" {
			vstr := item.ValueStr
			v := item.Value
			switch val := v.(type) {
			case *int:
				flag.IntVar(val, item.Name, *val, strings.Join(item.Usage, ";"))
			case *float64:
				flag.Float64Var(val, item.Name, *val, strings.Join(item.Usage, ";"))
			case *bool:
				flag.BoolVar(val, item.Name, *val, strings.Join(item.Usage, ";"))
			case *string:
				flag.StringVar(val, item.Name, *val, strings.Join(item.Usage, ";"))
			case *[]string:
				flag.StringVar(&item.ValueStr, item.Name, vstr, strings.Join(item.Usage, ";"))
			case *[]int:
				flag.StringVar(&item.ValueStr, item.Name, vstr, strings.Join(item.Usage, ";"))
			default:
				errs = append(errs, fmt.Errorf("设置项%s配置值%s不支持变量类型(%t)", item.Name, vstr, v))
			}
		}
	}
	flag.Parse()
	for _, item := range cf.items {
		// 注释语句 Name 为空字符
		if item.Name != "" {
			v := item.Value
			switch val := v.(type) {
			case *[]string:
				parseStringList(val, item.ValueStr)
			case *[]int:
				parseIntList(val, item.ValueStr, *val)
				fmt.Printf("--k(%s)--vstr(%s)-------\n", item.Name, item.ValueStr)
			}
		}
	}
	return errs
}

// SetValuesByEnv 从操作系统的环境变量获取配置。优先级中
// 配置值为空字符串会被忽略
func (cf *Conf) SetValuesByEnv() error {
	var err error
	for _, item := range cf.items {
		if item.Name == "" {
			// 注释语句 Name 为空字符
			continue
		}
		v := os.Getenv(item.Name)
		if v != "" {
			err1 := item.setValueVar(v)
			if err1 != nil {
				err = err1
			} else {
				item.ValueStr = v
			}
		}
	}
	return err
}

// SetValuesByEnvFile 从env配置文件更新配置项。优先级低。
// 配置值为空字符串会被忽略
func (cf *Conf) SetValuesByEnvFile(envfile string) error {
	content, err := os.ReadFile(envfile)
	if err != nil {
		return err
	}
	contstr := string(content)
	lines := strings.Split(contstr, "\n")
	// 解析env文件的每一行
	errStrList := []string{}
	for _, line := range lines {
		itemk, itemv := GetConfStrByLine(line)
		if itemk == "" || itemv == "" {
			continue
		}
		// fmt.Printf("-----ReadFile(%s)-----k(%s)=v(%s)--------\n", envfile, itemk, itemv)
		err = cf.setItemVar(itemk, itemv)
		if err != nil {
			errStrList = append(errStrList, err.Error())
		}
	}
	if len(errStrList) > 0 {
		return fmt.Errorf("env文件(%s)解析错误(SetValuesByEnvFile error):%s", envfile, strings.Join(errStrList, ";"))
	}
	return nil
}

// UpdateFile 将当前配置值持久化到指定文件。
//
// fpath 留空时默认更新 cf.files[0]（第一个配置文件）。
// 采用增量更新策略：只替换已有配置项的值行，保留原文件中的注释、空行和格式。
// 原文件中不存在的配置项会被追加到文件末尾。
// 若文件不存在，会自动创建并用当前值全量写入。
func (cf *Conf) UpdateFile(fpath string) error {
	if fpath == "" {
		fpath = cf.files[0]
	}

	content, err := os.ReadFile(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在时，用当前值全量创建
			content = []byte(cf.String())
		} else {
			return fmt.Errorf("read file(%s) err(%v)", fpath, err)
		}
	}

	// 统一处理 CRLF 为 LF
	text := strings.ReplaceAll(string(content), "\r", "")
	lines := strings.Split(text, "\n")
	updatedKeys := make(map[string]bool)

	for i, line := range lines {
		key, _ := GetConfStrByLine(line)
		if key == "" {
			continue
		}
		for _, item := range cf.items {
			if item.Name == key {
				lines[i] = rebuildValueLine(line, key, item.GetValue())
				updatedKeys[key] = true
				break
			}
		}
	}

	// 追加原文件中不存在的新 key
	hasNew := false
	for _, item := range cf.items {
		if item.Name != "" && !updatedKeys[item.Name] {
			if !hasNew {
				// 第一个新 key 前：除非文件末尾已是空行，否则加空行分隔
				if len(lines) > 0 && lines[len(lines)-1] != "" {
					lines = append(lines, "")
				}
			} else {
				// 后续新 key 之间始终加空行分隔
				lines = append(lines, "")
			}
			for _, s := range strings.Split(item.String(), "\n") {
				lines = append(lines, s)
			}
			hasNew = true
		}
	}

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("open file(%s) err(%v)", fpath, err)
	}
	defer f.Close()

	_, err = f.WriteString(strings.Join(lines, "\n"))
	if err != nil {
		return fmt.Errorf("write file(%s) err(%v)", fpath, err)
	}
	return nil
}

// UpdateByMap 通过 map 批量更新配置项的值，并将结果持久化到文件。
//
// mp 的键为配置项名称，值为新值。空 map 或 nil map 不会产生任何变更。
// 如果某个键名在注册的配置项中不存在，setItemVar 会静默跳过该键。
// 遇到错误时立即返回，已更新成功的值不会回滚。
func (cf *Conf) UpdateByMap(mp map[string]string, fpath string) error {
	if mp == nil {
		return nil
	}
	var err error
	for k, v := range mp {
		if err = cf.setItemVar(k, v); err != nil {
			return err
		}
	}
	return cf.UpdateFile(fpath)
}

// rebuildValueLine 替换一行中的值为新值，同时保留：
// - 字符串的引号风格（有引号保留引号，无引号不加）
// - 等号后的行内注释（# comment）
// - 原始缩进/间距格式
func rebuildValueLine(line, key, newValue string) string {
	eqIndex := strings.Index(line, "=")
	if eqIndex < 0 {
		return line
	}

	afterEq := line[eqIndex+1:]
	trimmed := strings.TrimSpace(afterEq)
	if trimmed == "" {
		return fmt.Sprintf("%s = %s", key, newValue)
	}

	// 检测引号风格
	if trimmed[0] == '"' || trimmed[0] == '\'' {
		q := string(trimmed[0])
		// 找到匹配的右引号（取最后出现的同种引号）
		lastQ := strings.LastIndex(trimmed, q)
		if lastQ > 0 {
			suffix := strings.TrimSpace(trimmed[lastQ+1:])
			if suffix != "" {
				return fmt.Sprintf("%s = %s%s%s %s", key, q, newValue, q, suffix)
			}
			return fmt.Sprintf("%s = %s%s%s", key, q, newValue, q)
		}
	}

	// 无引号：保留行内 # 注释
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		suffix := trimmed[idx:]
		return fmt.Sprintf("%s = %s %s", key, newValue, suffix)
	}

	return fmt.Sprintf("%s = %s", key, newValue)
}
