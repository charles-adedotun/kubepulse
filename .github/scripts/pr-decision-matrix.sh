#!/bin/bash

# PR Decision Matrix Script
# Implements 4 decision outcomes based on CI and Claude Code review results

set -e

echo "🎯 PR Decision Matrix Analysis..."

# Get inputs
CI_STATUS=${1:-"unknown"}
CLAUDE_DECISION=${2:-"COMMENT"}
PR_SIZE=${3:-"medium"}

echo "📊 Input Parameters:"
echo "   CI Status: $CI_STATUS"
echo "   Claude Decision: $CLAUDE_DECISION"
echo "   PR Size: $PR_SIZE"

# Decision matrix logic
make_decision() {
    local ci_status=$1
    local claude_decision=$2
    local pr_size=$3
    
    # Outcome 1: AUTO-MERGE
    # - CI passes AND Claude approves AND small PR
    if [[ "$ci_status" == "success" && "$claude_decision" == "APPROVE" && "$pr_size" == "small" ]]; then
        echo "auto-merge"
        return
    fi
    
    # Outcome 2: PASS BUT MANUAL MERGE
    # - CI passes AND Claude approves BUT medium/large PR
    # - CI passes AND Claude comments (no blocking issues)
    if [[ "$ci_status" == "success" && "$claude_decision" == "APPROVE" && "$pr_size" != "small" ]]; then
        echo "pass-manual-merge"
        return
    fi
    
    if [[ "$ci_status" == "success" && "$claude_decision" == "COMMENT" ]]; then
        echo "pass-manual-merge"
        return
    fi
    
    # Outcome 3: WAIT FOR APPROVAL
    # - CI passes BUT Claude requests changes
    # - CI has warnings but Claude approves
    if [[ "$ci_status" == "success" && "$claude_decision" == "REQUEST_CHANGES" ]]; then
        echo "wait-for-approval"
        return
    fi
    
    if [[ "$ci_status" == "warning" && "$claude_decision" == "APPROVE" ]]; then
        echo "wait-for-approval"
        return
    fi
    
    # Outcome 4: FAIL CI
    # - CI fails (regardless of Claude decision)
    # - Any other combination that doesn't fit above
    if [[ "$ci_status" == "failure" ]]; then
        echo "fail-ci"
        return
    fi
    
    # Default to wait for approval for unclear cases
    echo "wait-for-approval"
}

# Calculate PR size based on changed files
calculate_pr_size() {
    local changed_files=$(git diff --name-only HEAD~1 2>/dev/null | wc -l || echo "1")
    local additions=$(git diff --shortstat HEAD~1 2>/dev/null | grep -o '[0-9]* insertion' | grep -o '[0-9]*' || echo "10")
    
    if [[ $changed_files -le 3 && $additions -le 50 ]]; then
        echo "small"
    elif [[ $changed_files -le 10 && $additions -le 200 ]]; then
        echo "medium"
    else
        echo "large"
    fi
}

# Auto-calculate PR size if not provided
if [[ "$PR_SIZE" == "medium" ]]; then
    PR_SIZE=$(calculate_pr_size)
    echo "📏 Calculated PR size: $PR_SIZE"
fi

# Make the decision
DECISION=$(make_decision "$CI_STATUS" "$CLAUDE_DECISION" "$PR_SIZE")

echo ""
echo "🎯 DECISION MATRIX RESULT: $DECISION"
echo ""

# Output detailed explanation and actions
case $DECISION in
    "auto-merge")
        echo "✅ AUTO-MERGE"
        echo "   ✓ CI passing"
        echo "   ✓ Claude approved" 
        echo "   ✓ Small PR size"
        echo "   → Merging automatically"
        echo "AUTO_MERGE=true" >> $GITHUB_OUTPUT
        echo "MANUAL_REVIEW=false" >> $GITHUB_OUTPUT
        echo "BLOCK_MERGE=false" >> $GITHUB_OUTPUT
        ;;
    "pass-manual-merge")
        echo "✅ PASS BUT MANUAL MERGE REQUIRED"
        echo "   ✓ CI passing"
        echo "   ✓ Code quality acceptable"
        echo "   ⚠ Requires human review (size/complexity)"
        echo "   → Ready for manual merge"
        echo "AUTO_MERGE=false" >> $GITHUB_OUTPUT
        echo "MANUAL_REVIEW=true" >> $GITHUB_OUTPUT
        echo "BLOCK_MERGE=false" >> $GITHUB_OUTPUT
        ;;
    "wait-for-approval")
        echo "⏳ WAIT FOR APPROVAL"
        echo "   ⚠ Issues detected requiring review"
        echo "   → Waiting for maintainer approval"
        echo "AUTO_MERGE=false" >> $GITHUB_OUTPUT
        echo "MANUAL_REVIEW=true" >> $GITHUB_OUTPUT
        echo "BLOCK_MERGE=true" >> $GITHUB_OUTPUT
        ;;
    "fail-ci")
        echo "❌ FAIL CI"
        echo "   ✗ CI checks failed"
        echo "   → Must fix issues before proceeding"
        echo "AUTO_MERGE=false" >> $GITHUB_OUTPUT
        echo "MANUAL_REVIEW=false" >> $GITHUB_OUTPUT
        echo "BLOCK_MERGE=true" >> $GITHUB_OUTPUT
        exit 1
        ;;
esac

echo "DECISION=$DECISION" >> $GITHUB_OUTPUT
echo ""
echo "🏁 Decision matrix analysis complete"