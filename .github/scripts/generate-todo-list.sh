#!/bin/bash

# generate-todo-list.sh - Generate TODO list for rejected PRs in KubePulse
# Usage: generate-todo-list.sh <REJECTION_REASON> <CRITICAL_ISSUES_JSON>

set -e

REJECTION_REASON="$1"
CRITICAL_ISSUES="$2"

if [ -z "$REJECTION_REASON" ]; then
    echo "Error: Rejection reason is required"
    exit 1
fi

echo "Generating TODO list for rejected PR..."

# Convert reason to lowercase for pattern matching
reason_lower=$(echo "$REJECTION_REASON" | tr '[:upper:]' '[:lower:]')

# Initialize TODO list
cat > /tmp/todo-items.md << 'EOF'
## 📋 Action Items

The following tasks need to be completed before this PR can be merged:

EOF

# Track added items to prevent duplicates
declare -A added_items

# Function to add TODO item if not already added
add_todo_item() {
    local item="$1"
    if [ -z "${added_items[$item]}" ]; then
        echo "- [ ] $item" >> /tmp/todo-items.md
        added_items["$item"]=1
    fi
}

# Process critical issues from JSON array
if [ "$CRITICAL_ISSUES" != "[]" ] && [ "$CRITICAL_ISSUES" != "null" ]; then
    echo "$CRITICAL_ISSUES" | jq -r '.[] // empty' | while read -r issue; do
        if [ -n "$issue" ]; then
            add_todo_item "$issue"
        fi
    done
fi

# Add specific TODO items based on rejection reason patterns
case "$reason_lower" in
    *test*fail*|*failing*test*)
        add_todo_item "Fix failing unit tests"
        add_todo_item "Ensure all Go tests pass with \`go test -v -race ./...\`"
        add_todo_item "Fix frontend test failures with \`npm test\`"
        add_todo_item "Update test coverage to meet project standards"
        ;;
    *security*|*vulnerabilit*)
        add_todo_item "Address security vulnerabilities identified in scan"
        add_todo_item "Remove any hardcoded secrets or credentials"
        add_todo_item "Validate input sanitization and error handling"
        add_todo_item "Review Kubernetes RBAC configurations"
        ;;
    *code*review*|*quality*)
        add_todo_item "Address code review feedback from Claude AI"
        add_todo_item "Improve code documentation and comments"
        add_todo_item "Follow Go best practices for error handling"
        add_todo_item "Ensure TypeScript types are properly defined"
        ;;
    *build*fail*|*compilation*)
        add_todo_item "Fix build compilation errors"
        add_todo_item "Resolve Go module dependency issues"
        add_todo_item "Fix frontend build errors and type checking"
        add_todo_item "Ensure Docker build completes successfully"
        ;;
    *lint*|*style*|*format*)
        add_todo_item "Fix code linting issues with \`golangci-lint run\`"
        add_todo_item "Run \`go fmt\` on all Go files"
        add_todo_item "Fix ESLint errors in frontend code"
        add_todo_item "Follow project coding standards and conventions"
        ;;
    *performance*|*efficiency*)
        add_todo_item "Optimize algorithm performance and complexity"
        add_todo_item "Reduce memory allocation and garbage collection pressure"
        add_todo_item "Improve database query efficiency"
        add_todo_item "Optimize frontend bundle size and loading performance"
        ;;
    *documentation*|*docs*)
        add_todo_item "Add comprehensive code documentation"
        add_todo_item "Update README with new functionality"
        add_todo_item "Add API documentation for new endpoints"
        add_todo_item "Include usage examples and configuration details"
        ;;
    *kubernetes*|*k8s*)
        add_todo_item "Fix Kubernetes resource definitions and manifests"
        add_todo_item "Validate RBAC permissions and security policies"
        add_todo_item "Test Kubernetes API interactions thoroughly"
        add_todo_item "Ensure monitoring accuracy for Kubernetes resources"
        ;;
    *ai*|*ml*|*algorithm*)
        add_todo_item "Validate AI/ML algorithm correctness and accuracy"
        add_todo_item "Add comprehensive tests for anomaly detection logic"
        add_todo_item "Optimize machine learning model performance"
        add_todo_item "Ensure statistical calculations are mathematically sound"
        ;;
    *frontend*|*ui*|*react*)
        add_todo_item "Fix React component rendering issues"
        add_todo_item "Improve TypeScript type safety and definitions"
        add_todo_item "Add proper error boundaries and loading states"
        add_todo_item "Ensure responsive design and accessibility"
        ;;
    *api*|*endpoint*)
        add_todo_item "Fix API endpoint implementation and error handling"
        add_todo_item "Add proper request validation and sanitization"
        add_todo_item "Implement authentication and authorization checks"
        add_todo_item "Add comprehensive API testing coverage"
        ;;
    *monitoring*|*health*check*)
        add_todo_item "Validate health check logic and accuracy"
        add_todo_item "Test monitoring reliability under various conditions"
        add_todo_item "Ensure alerting thresholds are properly configured"
        add_todo_item "Verify monitoring performance and resource usage"
        ;;
esac

# Add general improvement items if no specific items were detected
todo_count=$(grep -c "^- \[ \]" /tmp/todo-items.md || echo "0")

if [ "$todo_count" -eq 0 ]; then
    add_todo_item "Address the issues identified in the rejection reason"
    add_todo_item "Ensure all CI/CD checks pass successfully"
    add_todo_item "Request a manual code review if needed"
    add_todo_item "Update tests to cover new or modified functionality"
fi

# Add helpful footer
cat >> /tmp/todo-items.md << 'EOF'

### 🛠️ KubePulse Development Guidelines

**Testing:**
- Run `make test` for full test suite
- Use `go test -v -race ./...` for Go tests
- Execute `npm test` for frontend tests
- Ensure coverage meets project standards

**Code Quality:**
- Use `golangci-lint run` for Go linting
- Run `npm run lint` for frontend linting
- Follow project coding conventions
- Add comprehensive documentation

**Kubernetes Integration:**
- Test with real Kubernetes clusters when possible
- Validate RBAC permissions and security
- Ensure monitoring accuracy and performance
- Test edge cases and error conditions

**AI/ML Components:**
- Validate algorithm correctness with test data
- Ensure statistical accuracy and performance
- Add comprehensive unit tests for ML logic
- Document algorithm behavior and limitations

Once all items are completed, request a new review. The AI system will re-evaluate the changes automatically.
EOF

final_count=$(grep -c "^- \[ \]" /tmp/todo-items.md || echo "0")
echo "Generated TODO list with $final_count items at /tmp/todo-items.md"

echo "TODO list generation completed successfully!"