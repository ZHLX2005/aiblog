# 贡献指南

感谢您对AI博客生成器项目的关注！我们欢迎所有形式的贡献，包括但不限于：

- 🐛 报告Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 提交代码修复
- ✨ 开发新功能

## 开始之前

在开始贡献之前，请：

1. 阅读我们的 [行为准则](CODE_OF_CONDUCT.md)
2. 查看已有的 [Issues](https://github.com/yourusername/aiblog/issues) 和 [Pull Requests](https://github.com/yourusername/aiblog/pulls)
3. 确保您的贡献符合项目目标

## 开发环境设置

### 前置要求

- Go 1.19 或更高版本
- Git
- FFmpeg（用于测试）
- 代码编辑器（推荐 VSCode 或 GoLand）

### 设置步骤

1. **Fork 项目**
   ```bash
   # 在 GitHub 上点击 Fork 按钮，然后克隆您的 fork
   git clone https://github.com/YOUR_USERNAME/aiblog.git
   cd aiblog
   ```

2. **添加上游仓库**
   ```bash
   git remote add upstream https://github.com/yourusername/aiblog.git
   ```

3. **创建开发分支**
   ```bash
   git checkout -b feature/your-feature-name
   # 或
   git checkout -b fix/your-bug-fix
   ```

4. **安装依赖**
   ```bash
   go mod tidy
   ```

5. **配置文件**
   ```bash
   cp config.example.yaml config.yaml
   # 编辑 config.yaml，添加必要的配置
   ```

## 开发指南

### 代码风格

我们遵循 [Go 官方代码规范](https://golang.org/doc/effective_go.html)：

- 使用 `gofmt` 格式化代码
- 使用 `golint` 检查代码风格
- 变量和函数使用驼峰命名法
- 导出的大写首字母，私有的小写首字母

### 提交规范

使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

类型说明：
- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整（不影响功能）
- `refactor`: 代码重构
- `test`: 添加或修改测试
- `chore`: 构建过程或辅助工具的变动

示例：
```
feat(tts): add support for new voice model

- Implement new TTS provider integration
- Add configuration options for voice selection
- Update documentation

Closes #123
```

### 测试

在提交代码前，请确保：

1. **运行测试**
   ```bash
   go test ./...
   ```

2. **测试覆盖率**
   ```bash
   go test -cover ./...
   ```

3. **集成测试**
   ```bash
   # 构建项目
   go build -o blog.exe

   # 测试基本功能
   ./blog.exe --help
   ./blog.exe generate -t "测试主题"
   ```

## 贡献类型

### 报告 Bug

使用 [Bug Report 模板](https://github.com/yourusername/aiblog/issues/new?template=bug_report.md) 创建 Issue，包含：

- 清晰的标题
- 详细的问题描述
- 复现步骤
- 期望行为
- 实际行为
- 环境信息（OS、Go版本等）
- 相关日志

### 提出新功能

使用 [Feature Request 模板](https://github.com/yourusername/aiblog/issues/new?template=feature_request.md) 创建 Issue，包含：

- 功能描述
- 使用场景
- 预期收益
- 可能的实现方案

### 提交代码

1. **创建分支**
   ```bash
   git checkout -b feature/your-feature
   ```

2. **开发和测试**
   ```bash
   # 编写代码
   # 运行测试
   go test ./...
   ```

3. **提交更改**
   ```bash
   git add .
   git commit -m "feat: add your feature description"
   ```

4. **推送分支**
   ```bash
   git push origin feature/your-feature
   ```

5. **创建 Pull Request**
   - 使用清晰的标题
   - 填写详细的描述
   - 引用相关的 Issue
   - 添加必要的标签

## Pull Request 指南

### PR 检查清单

在提交 PR 前，请确保：

- [ ] 代码通过所有测试
- [ ] 新功能有相应的测试
- [ ] 文档已更新
- [ ] 代码风格符合规范
- [ ] 提交信息符合规范
- [ ] 没有引入不必要的依赖

### PR 模板

```markdown
## 描述
简要描述此 PR 的目的和实现的功能。

## 变更类型
- [ ] Bug 修复
- [ ] 新功能
- [ ] 代码重构
- [ ] 文档更新
- [ ] 其他

## 测试
描述如何测试这些更改。

## 检查清单
- [ ] 我的代码遵循项目的代码规范
- [ ] 我已经进行了自我审查
- [ ] 我已经添加了必要的注释
- [ ] 我的更改生成了新的警告
- [ ] 我已经添加了测试来证明我的修复是有效的或我的功能可以工作
- [ ] 新的和现有的单元测试都通过了我的更改
- [ ] 任何依赖的更改都已经合并并发布

## 相关 Issue
Closes #(issue number)
```

## 特定贡献指南

### 添加新的 TTS 提供商

1. 在 `pkg/tts/` 目录下创建新的 provider 文件
2. 实现 `TTSProvider` 接口
3. 添加配置选项
4. 编写测试
5. 更新文档

### 添加新的视频格式支持

1. 在 `pkg/ffmpeg/` 中添加新格式支持
2. 更新配置文件选项
3. 添加相应的测试
4. 更新 README

### 改进提示词

1. 修改 `pkg/client/prompts.go`
2. 添加多语言支持（如需要）
3. 测试不同主题的生成效果
4. 更新配置说明

## 发布流程

项目维护者负责发布：

1. 更新版本号
2. 更新 CHANGELOG
3. 创建 Git 标签
4. 构建 release 二进制文件
5. 发布到 GitHub Releases

## 获取帮助

如果您需要帮助：

- 查看 [FAQ](https://github.com/yourusername/aiblog/discussions/categories/q-a)
- 在 [Discussions](https://github.com/yourusername/aiblog/discussions) 中提问
- 联系维护者

## 许可证

通过贡献代码，您同意您的贡献将在 [MIT 许可证](LICENSE) 下发布。

---

再次感谢您的贡献！🎉