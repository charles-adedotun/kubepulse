#!/bin/bash

# generate-merge-prompt.sh - Generate structured merge decision prompt for KubePulse
# Usage: generate-merge-prompt.sh <PR_NUMBER>

set -e

PR_NUMBER="$1"
CHECKS_PASSED="${CHECKS_PASSED:-false}"

if [ -z "$PR_NUMBER" ]; then
    echo "Error: PR number is required"
    exit 1
fi

if [ -z "$GH_TOKEN" ]; then
    echo "Error: GH_TOKEN environment variable is required"
    exit 1
fi

echo "Generating merge decision prompt for PR #${PR_NUMBER}..."

# Get PR information
PR_INFO=$(gh pr view $PR_NUMBER --json title,author,headRefName,baseRefName,mergeable,isDraft,body)
PR_TITLE=$(echo "$PR_INFO" | jq -r '.title')
PR_AUTHOR=$(echo "$PR_INFO" | jq -r '.author.login')
PR_BRANCH=$(echo "$PR_INFO" | jq -r '.headRefName')
BASE_BRANCH=$(echo "$PR_INFO" | jq -r '.baseRefName')
IS_MERGEABLE=$(echo "$PR_INFO" | jq -r '.mergeable')
IS_DRAFT=$(echo "$PR_INFO" | jq -r '.isDraft')
PR_BODY=$(echo "$PR_INFO" | jq -r '.body // ""')

# Get CI status
CI_STATUS=$(gh pr view $PR_NUMBER --json statusCheckRollup | jq -r '.statusCheckRollup[] | "\(.name): \(.conclusion // .status)"' | sort)

# Get review status
REVIEWS=$(gh pr view $PR_NUMBER --json reviews | jq -r '.reviews[] | "\(.author.login): \(.state)"' | sort | uniq)

# Check if there's a Claude review
CLAUDE_REVIEW_EXISTS="false"
if echo "$CI_STATUS" | grep -q "Claude Code Review"; then
    CLAUDE_REVIEW_EXISTS="true"
fi

# Get recent commits
COMMITS=$(gh pr view $PR_NUMBER --json commits | jq -r '.commits[-3:] | .[] | "\(.messageHeadline) (\(.abbreviatedOid))"')

# Generate the structured prompt
cat > /tmp/merge-decision-prompt.md << EOF
# KubePulse AI Merge Decision Request

You are an AI assistant helping make merge decisions for KubePulse, an intelligent Kubernetes health monitoring tool with ML-powered anomaly detection.

## Pull Request Context

**PR #${PR_NUMBER}**: ${PR_TITLE}
**Author**: ${PR_AUTHOR}
**Branch**: ${PR_BRANCH} → ${BASE_BRANCH}
**Mergeable**: ${IS_MERGEABLE}
**Draft Status**: ${IS_DRAFT}

### PR Description
${PR_BODY}

## CI/CD Status Summary

**Overall Checks Passed**: ${CHECKS_PASSED}

### Individual Check Status
\`\`\`
${CI_STATUS}
\`\`\`

## Review Status

### Human Reviews
\`\`\`
${REVIEWS}
\`\`\`

### Claude AI Review
**Available**: ${CLAUDE_REVIEW_EXISTS}

## Recent Commits
\`\`\`
${COMMITS}
\`\`\`

## KubePulse-Specific Evaluation Criteria

### 1. Code Quality Standards
- Go code follows project conventions and best practices
- React/TypeScript components are properly typed and structured
- Error handling is comprehensive and appropriate
- Code is maintainable and well-documented

### 2. Testing Requirements
- Unit tests cover new functionality
- Integration tests validate Kubernetes interactions
- Frontend tests cover UI components and hooks
- AI/ML algorithms have proper validation tests
- Test coverage maintains or improves existing levels

### 3. Security & Safety
- No hardcoded secrets or credentials
- Proper input validation and sanitization
- Kubernetes configurations follow security best practices
- API endpoints have appropriate authentication/authorization

### 4. Performance & Reliability
- Code is efficient and doesn't introduce performance regressions
- Memory usage is reasonable for monitoring workloads
- Kubernetes resource consumption is optimized
- AI/ML algorithms are computationally efficient

### 5. Architecture & Integration
- Changes align with KubePulse's monitoring architecture
- Kubernetes API usage follows best practices
- Frontend changes maintain design consistency
- Database operations are optimized and safe

## Decision Request

Based on the above context and KubePulse-specific criteria, make a merge decision.

**Required Response Format (JSON):**
\`\`\`json
{
  "decision": "APPROVE|MANUAL_APPROVAL|REJECT",
  "reason": "Clear explanation of the decision",
  "critical_issues": ["list", "of", "critical", "issues", "if", "any"],
  "recommended_action": "Specific next steps or improvements needed"
}
\`\`\`

**Decision Guidelines:**
- **APPROVE**: All checks pass, code quality is excellent, no security concerns, comprehensive tests
- **MANUAL_APPROVAL**: Good code quality but requires human review for complex changes or edge cases
- **REJECT**: Critical issues, security problems, failing tests, or significant quality concerns

**Focus Areas for KubePulse:**
- Kubernetes monitoring accuracy and reliability
- AI/ML algorithm correctness and performance
- Health check logic validity
- Dashboard usability and responsiveness
- System scalability and resource efficiency

Please provide your decision with specific, actionable reasoning based on the PR content and KubePulse's requirements.
EOF

echo "Generated merge decision prompt at /tmp/merge-decision-prompt.md"

# Create decision output file path for later use
echo "/tmp/ai_decision_${GITHUB_RUN_ID}_${GITHUB_RUN_ATTEMPT}.json" > /tmp/decision_file_path.txt

echo "Merge decision prompt generated successfully!"