package singbox

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Installer sing-box安装器
type Installer struct {
	installDir string
	binaryPath string
}

// NewInstaller 创建安装器
func NewInstaller(installDir string) *Installer {
	binaryName := "sing-box"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	
	return &Installer{
		installDir: installDir,
		binaryPath: filepath.Join(installDir, binaryName),
	}
}

// Check 检查sing-box是否已安装
func (i *Installer) Check() (bool, string, error) {
	// 检查指定路径的二进制文件
	if stat, err := os.Stat(i.binaryPath); err == nil && !stat.IsDir() {
		version, err := i.getVersion(i.binaryPath)
		if err != nil {
			log.Printf("获取sing-box版本失败: %v", err)
			return true, "unknown", nil
		}
		return true, version, nil
	}
	
	// 检查系统PATH中的sing-box
	if path, err := exec.LookPath("sing-box"); err == nil {
		version, err := i.getVersion(path)
		if err != nil {
			log.Printf("获取系统sing-box版本失败: %v", err)
			return true, "unknown", nil
		}
		// 如果系统中存在sing-box，更新二进制路径
		i.binaryPath = path
		return true, version, nil
	}
	
	return false, "", nil
}

// Install 安装sing-box
func (i *Installer) Install() error {
	log.Println("开始安装sing-box...")
	
	// 创建安装目录
	if err := os.MkdirAll(i.installDir, 0755); err != nil {
		return fmt.Errorf("创建安装目录失败: %v", err)
	}
	
	// 使用官方安装脚本
	if err := i.installWithScript(); err != nil {
		log.Printf("使用安装脚本失败: %v", err)
		// 尝试手动下载安装
		return i.installManually()
	}
	
	return nil
}

// GetBinaryPath 获取二进制文件路径
func (i *Installer) GetBinaryPath() string {
	return i.binaryPath
}

// installWithScript 使用官方安装脚本安装
func (i *Installer) installWithScript() error {
	log.Println("使用官方安装脚本安装sing-box...")
	
	// 下载安装脚本
	resp, err := http.Get("https://sing-box.app/install.sh")
	if err != nil {
		return fmt.Errorf("下载安装脚本失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载安装脚本失败: HTTP %d", resp.StatusCode)
	}
	
	// 创建临时脚本文件
	tmpScript := filepath.Join(os.TempDir(), "install-sing-box.sh")
	file, err := os.Create(tmpScript)
	if err != nil {
		return fmt.Errorf("创建临时脚本失败: %v", err)
	}
	defer os.Remove(tmpScript)
	
	// 写入脚本内容
	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		return fmt.Errorf("写入脚本内容失败: %v", err)
	}
	file.Close()
	
	// 设置执行权限
	if err := os.Chmod(tmpScript, 0755); err != nil {
		return fmt.Errorf("设置脚本权限失败: %v", err)
	}
	
	// 执行安装脚本
	cmd := exec.Command("/bin/sh", tmpScript)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PREFIX=%s", i.installDir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("执行安装脚本失败: %v", err)
	}
	
	log.Println("sing-box安装完成")
	return nil
}

// installManually 手动下载安装
func (i *Installer) installManually() error {
	log.Println("尝试手动下载安装sing-box...")
	
	// 获取系统架构信息
	arch := i.getArch()
	if arch == "" {
		return fmt.Errorf("不支持的系统架构: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	
	// 构建下载URL（这里使用GitHub releases作为备选）
	downloadURL := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/latest/download/sing-box-%s-%s.tar.gz", runtime.GOOS, arch)
	
	log.Printf("从 %s 下载sing-box...", downloadURL)
	
	// 下载文件
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}
	
	// 创建临时文件
	tmpFile := filepath.Join(os.TempDir(), "sing-box.tar.gz")
	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile)
	defer file.Close()
	
	// 保存下载内容
	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}
	file.Close()
	
	// 解压文件
	if err := i.extractTarGz(tmpFile, i.installDir); err != nil {
		return fmt.Errorf("解压文件失败: %v", err)
	}
	
	log.Println("sing-box手动安装完成")
	return nil
}

// getVersion 获取sing-box版本
func (i *Installer) getVersion(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	// 解析版本信息，格式通常是: sing-box version 1.x.x
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "sing-box version") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				return parts[2], nil
			}
		}
	}
	
	return strings.TrimSpace(string(output)), nil
}

// getArch 获取系统架构
func (i *Installer) getArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "386":
		return "386"
	case "arm":
		return "armv7"
	default:
		return ""
	}
}

// extractTarGz 解压tar.gz文件
func (i *Installer) extractTarGz(src, dest string) error {
	// 使用系统命令解压（简化实现）
	cmd := exec.Command("tar", "-xzf", src, "-C", dest, "--strip-components=1")
	return cmd.Run()
}