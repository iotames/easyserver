package easyconf

// StringVar 注册一个字符串类型的配置项。
// pval 为字符串指针，用于接收配置值；name 为配置项键名；
// defval 为默认值；title 为注释标题；usage 为多行使用说明。
func (cf *Conf) StringVar(pval *string, name string, defval, title string, usage ...string) {
	*pval = defval
	cf.addItem(pval, name, defval, title, usage...)
}

// BoolVar 注册一个布尔类型的配置项。
// pval 为布尔指针，用于接收配置值；name 为配置项键名；
// defval 为默认值；title 为注释标题；usage 为多行使用说明。
func (cf *Conf) BoolVar(pval *bool, name string, defval bool, title string, usage ...string) {
	*pval = defval
	cf.addItem(pval, name, defval, title, usage...)
}

// IntVar 注册一个整数类型的配置项。
// pval 为整数指针，用于接收配置值；name 为配置项键名；
// defval 为默认值；title 为注释标题；usage 为多行使用说明。
func (cf *Conf) IntVar(pval *int, name string, defval int, title string, usage ...string) {
	*pval = defval
	cf.addItem(pval, name, defval, title, usage...)
}

// StringListVar 注册一个字符串列表类型的配置项。
// pval 为 []string 指针；name 为配置项键名；
// defval 为默认值；title 为注释标题；usage 为多行使用说明。
// 配置文件中多个值用逗号分隔，如：ALLOW_IP = 10.0.0.1,10.0.0.2
func (cf *Conf) StringListVar(pval *[]string, name string, defval []string, title string, usage ...string) {
	*pval = defval
	cf.addItem(pval, name, defval, title, usage...)
}

// IntListVar 注册一个整数列表类型的配置项。
// pval 为 []int 指针；name 为配置项键名；
// defval 为默认值；title 为注释标题；usage 为多行使用说明。
// 配置文件中多个值用逗号分隔，如：AGE_RANGE = 3,6,9
func (cf *Conf) IntListVar(pval *[]int, name string, defval []int, title string, usage ...string) {
	*pval = defval
	cf.addItem(pval, name, defval, title, usage...)
}

// Float64Var 注册一个浮点数类型的配置项。
// pval 为 float64 指针；name 为配置项键名；
// defval 为默认值；title 为注释标题；usage 为多行使用说明。
func (cf *Conf) Float64Var(pval *float64, name string, defval float64, title string, usage ...string) {
	*pval = defval
	cf.addItem(pval, name, defval, title, usage...)
}
