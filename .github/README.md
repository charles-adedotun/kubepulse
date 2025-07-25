# KubePulse AI-Powered GitHub Workflows

This directory contains advanced GitHub Actions workflows powered by Claude AI to automate code review, merge decisions, and provide interactive assistance for the KubePulse project.

## 🤖 Available Workflows

### 1. Claude Code Review (`claude-code-review.yml`)
**Triggers**: Pull request events (opened, synchronized, reopened)

**Features**:
- Comprehensive AI-powered code reviews using Claude Sonnet 4
- KubePulse-specific analysis focusing on:
  - Go backend best practices and Kubernetes integration
  - React/TypeScript frontend code quality
  - AI/ML algorithm correctness and performance
  - Security and performance considerations
  - Testing coverage and quality

**Output**: Detailed review comments and analysis artifacts

### 2. AI Merge Decision (`merge-decision.yml`)
**Triggers**: Pull request events and workflow completions

**Features**:
- Multi-stage decision process:
  1. **wait-for-checks**: Monitors CI/CD pipeline status
  2. **merge-decision**: AI analyzes PR quality and makes merge recommendation
  3. **auto-merge**: Automatically merges approved PRs

**Decision Outcomes**:
- ✅ **APPROVE**: Auto-merge eligible PRs
- 👀 **MANUAL_APPROVAL**: Requires human review
- ❌ **REJECT**: Generates TODO list for improvements

### 3. Interactive Claude Assistance (`claude.yml`)
**Triggers**: `@claude` mentions in issues, PRs, and comments

**Features**:
- Specialized KubePulse expertise in:
  - Go development and Kubernetes integration
  - React/TypeScript frontend development
  - AI/ML anomaly detection algorithms
  - Testing strategies and coverage improvement
  - Kubernetes monitoring best practices

**Usage**: Mention `@claude` in any issue or PR comment to get AI assistance

### 4. Enhanced CI Pipeline (`ci.yml`)
**Triggers**: Push to main branch and pull requests

**Features**:
- Comprehensive testing pipeline:
  - Backend Go tests with race detection
  - Frontend React/TypeScript tests
  - Security scanning (Gosec, Nancy, npm audit)
  - Linting and formatting checks
  - Docker build and integration tests
  - Performance benchmarks
  - AI quality assessment

## 🛠️ Supporting Scripts

### `generate-merge-prompt.sh`
Creates structured prompts for AI merge decisions including:
- PR context and metadata
- CI/CD status summary
- Review status and commit history
- KubePulse-specific evaluation criteria

### `generate-todo-list.sh`
Generates actionable TODO lists for rejected PRs with:
- Issue-specific action items
- KubePulse development guidelines
- Testing and quality requirements
- Links to relevant documentation

## 🔧 Setup and Configuration

### Required Secrets
Add these secrets to your GitHub repository:

```
ANTHROPIC_API_KEY          # Your Anthropic Claude API key
CLAUDE_CODE_OAUTH_TOKEN    # Claude Code OAuth token for interactive features
```

### Permissions
The workflows require these GitHub permissions:
- `contents: read/write` - Repository access
- `pull-requests: write` - PR comments and management
- `issues: write` - Issue comments
- `actions: read` - CI status checking
- `checks: read` - Check run status

## 📋 Workflow Integration

### AI Review Process
1. **Code Review**: Claude analyzes every PR for quality, security, and KubePulse-specific concerns
2. **CI Integration**: Workflows wait for all CI checks to complete
3. **Merge Decision**: AI makes informed merge recommendations based on:
   - Code quality and test coverage
   - Security scan results
   - KubePulse architectural compliance
   - Performance implications

### Auto-Merge Criteria
For a PR to be auto-merged, it must:
- ✅ Pass all CI/CD checks
- ✅ Receive AI approval from Claude review
- ✅ Meet KubePulse quality standards
- ✅ Have adequate test coverage
- ✅ Pass security scans

### Manual Review Triggers
PRs require manual review when:
- Complex architectural changes
- New AI/ML algorithm implementations
- Security-sensitive modifications
- Breaking changes to APIs
- Kubernetes configuration changes

## 🎯 KubePulse-Specific Features

### Monitoring Focus
- Health check accuracy and reliability
- Kubernetes API integration best practices
- Resource monitoring performance
- Alert threshold validation

### AI/ML Validation
- Algorithm correctness verification
- Statistical calculation validation
- Performance optimization suggestions
- Model testing adequacy

### Security Considerations
- Kubernetes RBAC validation
- Secret management review
- Input sanitization checks
- Container security best practices

## 📊 Metrics and Reporting

### Coverage Reports
- Go test coverage with HTML reports
- Frontend test coverage
- Integration test results
- Performance benchmarks

### Quality Metrics
- Code quality scores from Claude AI
- Security vulnerability counts
- Performance regression detection
- Test coverage trends

## 🔄 Continuous Improvement

The AI workflows learn from:
- Successful merge patterns
- Common issue types
- KubePulse-specific challenges
- Team feedback and preferences

### Customization
- Adjust AI prompts for project evolution
- Update quality thresholds
- Modify auto-merge criteria
- Add project-specific checks

## 📚 Usage Examples

### Getting AI Help
```
@claude How can I improve the performance of the anomaly detection algorithm?
```

### Triggering Workflows
- Open/update a PR → Automatic code review
- CI completes → Merge decision analysis
- Mention @claude → Interactive assistance

### Interpreting Results
- Green checkmarks ✅ → Ready to merge
- Yellow warnings ⚠️ → Manual review needed  
- Red X ❌ → Issues need fixing

## 🤝 Contributing

When contributing to KubePulse:
1. Ensure all CI checks pass
2. Address Claude AI feedback
3. Maintain test coverage standards
4. Follow Kubernetes best practices
5. Document significant changes

The AI workflows will guide you through the process and provide specific feedback for improvements.

---

*These workflows represent the cutting edge of AI-assisted software development, specifically tailored for KubePulse's intelligent Kubernetes monitoring capabilities.*