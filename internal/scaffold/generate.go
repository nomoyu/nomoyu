package scaffold

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// 目录结构：internal/scaffold/templates/ddd/**/*
//
//go:embed templates/ddd/**/*
var dddFS embed.FS

// SplitCSV 转 slice + 归一化小写
func SplitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	return out
}

// pascalCase user_profile -> UserProfile
func pascalCase(s string) string {
	seps := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	for i, seg := range seps {
		if seg == "" {
			continue
		}
		seps[i] = strings.ToUpper(seg[:1]) + strings.ToLower(seg[1:])
	}
	return strings.Join(seps, "")
}

type tmplData struct {
	Project  string
	Module   string
	Context  string // 小写
	ContextP string // Pascal
}

// GenerateDDDProject 创建 DDD 骨架
func GenerateDDDProject(project, module string, ctxs []string, frameworkPath string) error {
	dest := filepath.Join(".", project)
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// 1) 复制基础骨架
	if err := copyDDDTplDir("templates/ddd/skeleton", dest, tmplData{
		Project: project,
		Module:  moduleOrDefault(project, module),
	}); err != nil {
		return err
	}

	// 2) go mod init + tidy
	if err := runGoModInit(dest, moduleOrDefault(project, module)); err != nil {
		return fmt.Errorf("go mod init 失败: %w", err)
	}

	// 写 go-gin-framework 的本地 replace（优先参数，其次环境变量）
	if err := ensureFrameworkReplace(dest, frameworkPathFromCLIorEnv(frameworkPath)); err != nil {
		return err
	}

	if err := runGoModTidy(dest); err != nil {
		return fmt.Errorf("go mod tidy 失败: %w", err)
	}

	// 3) 追加上下文（可选）
	if len(ctxs) > 0 {
		if err := AddDDDContexts(dest, ctxs); err != nil {
			return err
		}
	}

	// 生成 contexts
	if len(ctxs) > 0 {
		if err := AddDDDContexts(dest, ctxs); err != nil {
			return err
		}
		// 追加完再 tidy
		if err := runGoModTidy(dest); err != nil {
			return err
		}
	}
	return nil
}

// AddDDDContexts 在已有项目中追加领域上下文
func AddDDDContexts(projectRoot string, ctxs []string) error {
	for _, ctx := range ctxs {
		data := tmplData{
			Project:  filepath.Base(projectRoot),
			Module:   detectModule(projectRoot),
			Context:  ctx,
			ContextP: pascalCase(ctx),
		}
		// 写入四层：domain/application/infrastructure/interfaces
		if err := copyDDDTplDir("templates/ddd/context", projectRoot, data); err != nil {
			return err
		}
	}
	// 追加完跑一次 tidy
	if err := runGoModTidy(projectRoot); err != nil {
		return err
	}
	return nil
}

func moduleOrDefault(project, module string) string {
	if strings.TrimSpace(module) != "" {
		return module
	}
	return project // 允许本地相对 module
}

func detectModule(root string) string {
	// 读取 go.mod 第一行
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(b), "\n")
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(l, "module "))
		}
	}
	return ""
}

func runGoModInit(dir, module string) error {
	if module == "" {
		return nil
	}
	cmd := exec.Command("go", "mod", "init", module)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

func runGoModTidy(dir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}

func copyDDDTplDir(src, dest string, data tmplData) error {
	entries, err := dddFS.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dest, e.Name())

		// 处理占位目录名：CTX -> 实际上下文名
		to = strings.ReplaceAll(to, "CTX", data.Context)

		if e.IsDir() {
			if err := os.MkdirAll(to, 0755); err != nil {
				return err
			}
			if err := copyDDDTplDir(from, to, data); err != nil {
				return err
			}
			continue
		}

		// 文件：.tmpl 需要渲染；普通文件原样拷贝
		b, err := dddFS.ReadFile(from)
		if err != nil {
			return err
		}
		if strings.HasSuffix(from, ".tmpl") {
			// 渲染
			t, err := template.New(filepath.Base(from)).Funcs(template.FuncMap{
				"Pascal": pascalCase,
			}).Parse(string(b))
			if err != nil {
				return err
			}
			// 去掉 .tmpl
			to = strings.TrimSuffix(to, ".tmpl")
			f, err := os.Create(to)
			if err != nil {
				return err
			}
			if err := t.Execute(f, data); err != nil {
				f.Close()
				return err
			}
			f.Close()
		} else {
			if err := os.WriteFile(to, b, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func frameworkPathFromCLIorEnv(cli string) string {
	if strings.TrimSpace(cli) != "" {
		return cli
	}
	if p := os.Getenv("NOMOYU_FRAMEWORK_PATH"); strings.TrimSpace(p) != "" {
		return p
	}
	// 常见相对路径
	if _, err := os.Stat("../go-gin-framework"); err == nil {
		return "../go-gin-framework"
	}
	return ""
}

func ensureFrameworkReplace(dest, fwPath string) error {
	if fwPath == "" {
		// 不设置也能拉到远程时就用远程；本地开发建议传 --framework
		return nil
	}
	// 先 require 一个占位版本
	if err := runCmd(dest, "go", "mod", "edit", "-require=github.com/nomoyu/go-gin-framework@v0.0.0"); err != nil {
		return err
	}
	// 再 replace 到本地
	return runCmd(dest, "go", "mod", "edit", fmt.Sprintf("-replace=github.com/nomoyu/go-gin-framework=%s", fwPath))
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(out))
	}
	return nil
}
