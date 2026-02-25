# SSH 配置管理

> **类型**: 功能设计
> **状态**: ✅ 完成
> **最后更新**: 2026-02-04
> **关联文档**: [workflow-schema.md](./workflow-schema.md)

---

## 配置文件结构

```
~/.projecttemplate/
├── config/
│   ├── ssh_targets.yaml      # SSH 目标配置
│   └── credentials.yaml       # 加密存储的凭据
└── ssh/
    ├── id_rsa                 # 默认 SSH 密钥
    └── known_hosts           # 已知主机密钥
```

## 配置文件格式

### ssh_targets.yaml

```yaml
# SSH 目标配置
ssh_targets:
  # NAS 设备
  nas:
    display_name: "NAS 存储"
    host: "192.168.1.100"
    port: 22
    user: "admin"
    auth_type: "key"               # key | password
    key_path: "~/.ssh/id_rsa_nas"  # 可选，默认使用默认密钥

  # 家庭服务器
  home-server:
    display_name: "家庭服务器"
    host: "home.example.com"
    port: 2222
    user: "dylan"
    auth_type: "key"
    key_path: "~/.ssh/id_rsa_home"

  # 工作服务器 (密码认证)
  work-server:
    display_name: "工作服务器"
    host: "10.0.1.50"
    port: 22
    user: "deploy"
    auth_type: "password"
    # 密码存储在 credentials.yaml 中，引用:
    password_ref: "work-server"

  # 本地 Docker 容器
  docker-local:
    display_name: "本地 Docker"
    host: "localhost"
    port: 2222
    user: "root"
    auth_type: "key"
    key_path: "~/.ssh/id_rsa"

# 全局默认值
defaults:
  port: 22
  user: "root"
  auth_type: "key"
  key_path: "~/.ssh/id_rsa"
  timeout: 30000                # 连接超时 (毫秒)
  max_retries: 3                 # 连接重试次数
```

### credentials.yaml (加密)

```yaml
# 凭据配置 (加密存储)
credentials:
  # 工作服务器密码
  work-server:
    password: "encrypted:..."    # 使用加密密码

  # 其他凭据...
```

## Go 代码实现

### 配置结构定义

```go
package ssh

import (
    "time"
    "golang.org/x/crypto/ssh"
)

// SSHTarget SSH 目标配置
type SSHTarget struct {
    DisplayName string   `yaml:"display_name"`
    Host       string   `yaml:"host"`
    Port       int      `yaml:"port"`
    User       string   `yaml:"user"`
    AuthType   AuthType `yaml:"auth_type"`
    KeyPath    string   `yaml:"key_path"`
    Password   string   `yaml:"password,omitempty"`   // 加密存储
    PasswordRef string   `yaml:"password_ref,omitempty"` // 引用凭据

    // 连接配置
    Timeout    time.Duration `yaml:"timeout"`
    MaxRetries int           `yaml:"max_retries"`

    // 运行时状态
    client     *ssh.Client `yaml:"-"`
}

type AuthType string

const (
    AuthTypeKey      AuthType = "key"
    AuthTypePassword AuthType = "password"
)

// SSHTargetManager SSH 目标管理器
type SSHTargetManager struct {
    targets    map[string]*SSHTarget
    credentials map[string]string
    crypto     Crypto
    configPath string
}

func NewSSHTargetManager(configDir string) *SSHTargetManager {
    return &SSHTargetManager{
        targets:    make(map[string]*SSHTarget),
        credentials: make(map[string]string),
        crypto:     NewAESCrypto(),
        configPath: configDir,
    }
}

// Load 加载配置
func (m *SSHTargetManager) Load() error {
    // 加载 SSH 目标配置
    targetsFile := filepath.Join(m.configPath, "ssh_targets.yaml")
    data, err := os.ReadFile(targetsFile)
    if err != nil {
        return fmt.Errorf("failed to load ssh_targets.yaml: %w", err)
    }

    var config struct {
        Targets map[string]SSHTarget `yaml:"ssh_targets"`
        Defaults map[string]any        `yaml:"defaults,omitempty"`
    }

    if err := yaml.Unmarshal(data, &config); err != nil {
        return fmt.Errorf("failed to parse ssh_targets.yaml: %w", err)
    }

    // 应用默认值
    for name, target := range config.Targets {
        if target.Port == 0 {
            target.Port = m.getDefaultInt(config.Defaults, "port", 22)
        }
        if target.User == "" {
            target.User = m.getDefaultString(config.Defaults, "user", "root")
        }
        if target.AuthType == "" {
            target.AuthType = AuthType(m.getDefaultString(config.Defaults, "auth_type", "key"))
        }
        if target.KeyPath == "" && target.AuthType == AuthTypeKey {
            target.KeyPath = m.getDefaultString(config.Defaults, "key_path", "~/.ssh/id_rsa")
        }
        if target.Timeout == 0 {
            target.Timeout = 30 * time.Second
        }

        m.targets[name] = &target
    }

    // 加载凭据
    return m.loadCredentials()
}

// Get 获取 SSH 目标
func (m *SSHTargetManager) Get(name string) (*SSHTarget, error) {
    target, ok := m.targets[name]
    if !ok {
        return nil, fmt.Errorf("SSH target not found: %s", name)
    }

    // 解析密码引用
    if target.PasswordRef != "" {
        if password, ok := m.credentials[target.PasswordRef]; ok {
            target.Password = password
        }
    } else if target.Password != "" {
        // 解密密码
        decrypted, err := m.crypto.Decrypt(target.Password)
        if err != nil {
            return nil, fmt.Errorf("failed to decrypt password: %w", err)
        }
        target.Password = decrypted
    }

    return target, nil
}

// GetClient 获取 SSH 客户端
func (m *SSHTargetManager) GetClient(ctx context.Context, name string) (*ssh.Client, error) {
    target, err := m.Get(name)
    if err != nil {
        return nil, err
    }

    // 如果已有缓存的客户端，验证后返回
    if target.client != nil {
        // 发送 keepalive 测试连接
        _, _, err = target.client.SendRequest("keepalive-alive", []byte{}, false)
        if err == nil {
            return target.client, nil
        }
        // 连接已断开，重新连接
        target.client.Close()
    }

    // 建立新连接
    client, err := m.connect(ctx, target)
    if err != nil {
        return nil, err
    }

    target.client = client
    return client, nil
}

// connect 建立 SSH 连接
func (m *SSHTargetManager) connect(ctx context.Context, target *SSHTarget) (*ssh.Client, error) {
    var auth ssh.AuthMethod

    switch target.AuthType {
    case AuthTypeKey:
        // 密钥认证
        keyPath := expandPath(target.KeyPath)
        signer, err := m.getSigner(keyPath)
        if err != nil {
            return nil, fmt.Errorf("failed to load SSH key: %w", err)
        }
        auth = ssh.PublicKeys(signer)

    case AuthTypePassword:
        // 密码认证
        auth = ssh.Password(target.Password)
    }

    config := &ssh.ClientConfig{
        User:            target.User,
        Auth:            auth,
        HostKeyCallback: m.hostKeyCallback,
        Timeout:         target.Timeout,
    }

    // 建立连接
    conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", target.Host, target.Port), config)
    if err != nil {
        return nil, fmt.Errorf("failed to dial: %w", err)
    }

    client := ssh.NewClientConn(conn, config)
    return client, nil
}

// getSigner 加载私钥签名器
func (m *SSHTargetManager) getSigner(keyPath string) (ssh.Signer, error) {
    key, err := os.ReadFile(keyPath)
    if err != nil {
        return nil, err
    }

    signer, err := ssh.ParsePrivateKey(key, "")
    if err != nil {
        return nil, err
    }

    return signer, nil
}

// hostKeyCallback 主机密钥回调
func (m *SSHTargetManager) hostKeyCallback(hostname string, remote net.Addr, key ssh.PublicKey) error {
    // TODO: 实现已知主机密钥验证
    // 1. 检查 ~/.projecttemplate/ssh/known_hosts
    // 2. 首次连接时询问用户是否信任
    // 3. 记录已验证的主机密钥
    return nil
}

// List 列出所有目标
func (m *SSHTargetManager) List() []*SSHTarget {
    targets := make([]*SSHTarget, 0, len(m.targets))
    for _, t := range m.targets {
        targets = append(targets, t)
    }
    return targets
}

// Add 动态添加目标
func (m *SSHTargetManager) Add(name string, target *SSHTarget) error {
    if _, exists := m.targets[name]; exists {
        return fmt.Errorf("SSH target already exists: %s", name)
    }
    m.targets[name] = target

    // 保存到配置文件
    return m.save()
}

// Remove 删除目标
func (m *SSHTargetManager) Remove(name string) error {
    if _, exists := m.targets[name]; !exists {
        return fmt.Errorf("SSH target not found: %s", name)
    }
    delete(m.targets, name)

    return m.save()
}

// save 保存配置
func (m *SSHTargetManager) save() error {
    config := struct {
        Targets map[string]SSHTarget `yaml:"ssh_targets"`
    }{
        Targets: make(map[string]SSHTarget),
    }

    for name, target := range m.targets {
        config.Targets[name] = *target
    }

    data, err := yaml.Marshal(config)
    if err != nil {
        return err
    }

    targetsFile := filepath.Join(m.configPath, "ssh_targets.yaml")
    return os.WriteFile(targetsFile, data, 0644)
}
```

## SSH Tool 实现

```go
package tools

import (
    "context"
    "fmt"
    "io"
    "os"
    "time"

    "golang.org/x/crypto/ssh"
    "github.com/pkg/sftp"
)

type SSHTool struct {
    targetManager *ssh.SSHTargetManager
}

// Exec 执行远程命令
func (t *SSHTool) Exec(ctx context.Context, req *SSHExecRequest) (*SSHExecResult, error) {
    // 获取 SSH 客户端
    client, err := t.targetManager.GetClient(ctx, req.Target)
    if err != nil {
        return nil, err
    }

    // 创建会话
    session, err := client.NewSession()
    if err != nil {
        return nil, fmt.Errorf("failed to create session: %w", err)
    }
    defer session.Close()

    // 执行命令
    output, err := session.CombinedOutput(req.Command)
    if err != nil {
        return nil, fmt.Errorf("command failed: %w", err)
    }

    return &SSHExecResult{
        Output:    string(output),
        ExitCode:  0, // TODO: 解析退出码
    }, nil
}

// Download 下载文件
func (t *SSHTool) Download(ctx context.Context, req *SSHDownloadRequest) error {
    client, err := t.targetManager.GetClient(ctx, req.Target)
    if err != nil {
        return err
    }

    // 创建 SFTP 客户端
    sftpClient, err := sftp.NewClient(client)
    if err != nil {
        return fmt.Errorf("failed to create SFTP client: %w", err)
    }
    defer sftpClient.Close()

    // 打开远程文件
    srcFile, err := sftpClient.Open(req.RemotePath)
    if err != nil {
        return fmt.Errorf("failed to open remote file: %w", err)
    }
    defer srcFile.Close()

    // 创建本地文件
    dstFile, err := os.Create(req.LocalPath)
    if err != nil {
        return fmt.Errorf("failed to create local file: %w", err)
    }
    defer dstFile.Close()

    // 复制文件
    _, err = io.Copy(dstFile, srcFile)
    if err != nil {
        return fmt.Errorf("failed to download file: %w", err)
    }

    return nil
}

// Upload 上传文件
func (t *SSHTool) Upload(ctx context.Context, req *SSHUploadRequest) error {
    client, err := t.targetManager.GetClient(ctx, req.Target)
    if err != nil {
        return err
    }

    sftpClient, err := sftp.NewClient(client)
    if err != nil {
        return fmt.Errorf("failed to create SFTP client: %w", err)
    }
    defer sftpClient.Close()

    // 打开本地文件
    srcFile, err := os.Open(req.LocalPath)
    if err != nil {
        return fmt.Errorf("failed to open local file: %w", err)
    }
    defer srcFile.Close()

    // 创建远程文件
    dstFile, err := sftpClient.Create(req.RemotePath)
    if err != nil {
        return fmt.Errorf("failed to create remote file: %w", err)
    }
    defer dstFile.Close()

    // 复制文件
    _, err = io.Copy(dstFile, srcFile)
    if err != nil {
        return fmt.Errorf("failed to upload file: %w", err)
    }

    return nil
}

// DockerRun 在远程执行 Docker 命令
func (t *SSHTool) DockerRun(ctx context.Context, req *SSHDockerRequest) (*SSHExecResult, error) {
    // 构建 docker run 命令
    cmd := t.buildDockerCommand(req)

    return t.Exec(ctx, &SSHExecRequest{
        Target:  req.Target,
        Command: cmd,
    })
}

func (t *SSHTool) buildDockerCommand(req *SSHDockerRequest) string {
    parts := []string{"docker", "run", "--rm"}

    // 添加卷挂载
    for _, vol := range req.Volumes {
        parts = append(parts, "-v", vol)
    }

    // 添加镜像
    parts = append(parts, req.Image)

    // 添加命令
    if req.Command != "" {
        parts = append(parts, req.Command)
    }

    // 转义为单行命令
    return strings.Join(parts, " ")
}
```

## 请求/响应结构

```go
// SSH 执行请求
type SSHExecRequest struct {
    Target  string                 // 目标名称 (nas, home-server)
    Command string                 // 要执行的命令
    Timeout time.Duration          // 超时 (可选)
}

// SSH 执行结果
type SSHExecResult struct {
    Output   string  // 标准输出 + 标准错误
    ExitCode int     // 退出码
    Error    error   // 错误信息
}

// SSH 下载请求
type SSHDownloadRequest struct {
    Target     string  // 目标名称
    RemotePath string  // 远程文件路径
    LocalPath  string  // 本地保存路径
}

// SSH 上传请求
type SSHUploadRequest struct {
    Target     string  // 目标名称
    LocalPath  string  // 本地文件路径
    RemotePath string  // 远程保存路径
}

// SSH Docker 请求
type SSHDockerRequest struct {
    Target   string            // 目标名称
    Image    string            // Docker 镜像
    Command  string            // 容器内命令
    Volumes  []string          // 卷挂载: ["host:container", ...]
    Envs     map[string]string // 环境变量
}
```

## 使用示例

```yaml
# 工作流中使用 SSH
steps:
  # 远程执行命令
  - id: list_files
    tool: ssh.exec
    input:
      target: nas
      command: "ls /data/videos/*.mp4"

  # 远程 Docker
  - id: docker_extract
    tool: ssh.docker_run
    input:
      target: nas
      image: "ffmpeg:latest"
      volumes: ["/data:/work"]
      command: |
        ffmpeg -i /work/input.mp4 -vn -acodec pcm_s16le -ar 16000 -ac 1 /work/output.wav

  # 下载文件
  - id: download
    tool: ssh.download
    input:
      target: nas
      remote_path: "/data/output.wav"
      local_path: "/tmp/audio.wav"

  # 上传文件
  - id: upload
    tool: ssh.upload
    input:
      target: nas
      local_path: "/tmp/subtitles.srt"
      remote_path: "/data/subtitles/video.srt"
```

## CLI 工具（可选）

```bash
# 管理 SSH 目标
projecttemplate ssh list                    # 列出所有目标
projecttemplate ssh add nas 192.168.1.100     # 添加目标
projecttemplate ssh remove nas                # 删除目标
projecttemplate ssh test nas                  # 测试连接

# 管理凭据
projecttemplate ssh credentials set           # 设置凭据（会加密存储）
projecttemplate ssh credentials list           # 列出凭据
```
