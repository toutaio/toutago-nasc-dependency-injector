# GitHub Branch Protection Configuration

This document provides instructions for configuring branch protection rules to enforce CI checks before merging.

## Prerequisites

- Repository admin access
- CI workflow merged to main branch
- At least one successful CI run

## Configuration Steps

### 1. Navigate to Branch Protection Settings

1. Go to your repository on GitHub
2. Click **Settings** (top menu)
3. Click **Branches** (left sidebar)
4. Under "Branch protection rules", click **Add rule**

### 2. Configure Protection Rule

**Branch name pattern:**
```
main
```

### 3. Enable Required Settings

Check the following options:

#### ✅ Require a pull request before merging
- ✅ Require approvals (optional, set to 1 if using)
- ✅ Dismiss stale pull request approvals when new commits are pushed
- ✅ Require review from Code Owners (if applicable)

#### ✅ Require status checks to pass before merging
- ✅ Require branches to be up to date before merging

**Select required status checks:**
- ✅ `test` (all matrix combinations will be automatically included)
- ✅ `coverage`
- ✅ `security`
- ✅ `quality`
- ✅ `build` (all matrix combinations)
- ℹ️ `benchmarks` (optional - can be informational only)
- ℹ️ `licenses` (optional - can be informational only)

#### ⚠️ Other Recommended Settings

- ✅ Require conversation resolution before merging
- ✅ Require signed commits (optional, for security)
- ✅ Require linear history (optional, for clean history)
- ⚠️ Include administrators (optional - applies rules to admins too)

#### 🚫 Do Not Enable (for this project)
- ❌ Require deployments to succeed (not applicable)
- ❌ Lock branch (prevents all changes)

### 4. Save Changes

Click **Create** or **Save changes** at the bottom of the page.

## Status Check Names

The CI workflow creates the following status checks:

| Check Name | Description | Matrix Jobs | Required? |
|------------|-------------|-------------|-----------|
| `test` | Run tests with race detector | 9 jobs (3 Go × 3 OS) | ✅ Yes |
| `coverage` | Code coverage measurement | 1 job | ✅ Yes |
| `security` | Vulnerability scanning | 1 job | ✅ Yes |
| `quality` | Code quality (fmt, vet, staticcheck) | 1 job | ✅ Yes |
| `build` | Build verification | 3 jobs (3 OS) | ✅ Yes |
| `benchmarks` | Performance benchmarks | 1 job | ℹ️ Optional |
| `licenses` | License compliance | 1 job | ℹ️ Optional |

**Total required checks:** ~17 status checks (with matrix expansions)

## Verification

After configuring:

1. Create a test PR with a small change
2. Verify status checks appear on the PR
3. Confirm merge button is blocked until checks pass
4. Verify checks must pass before merging is allowed

## Troubleshooting

### Status checks don't appear in the list

**Solution:** The checks must run at least once on a PR before they appear in the list.
1. Create a draft PR with any change
2. Let CI run completely
3. Go back to branch protection settings
4. The checks should now be available to select

### Can't find specific matrix job names

**Solution:** GitHub automatically handles matrix jobs. When you select `test`, it applies to all matrix combinations.

### Accidentally blocked yourself

**Solution:** 
- Temporarily disable "Include administrators"
- Make necessary changes
- Re-enable the protection

### Want to bypass checks in emergency

**Solution:**
- Only admins can bypass (if enabled)
- Temporarily disable specific checks
- Make changes
- Re-enable protections immediately

## Advanced Configuration

### Require specific number of approvals

Set **Required number of approvals before merging** to `1` or `2` depending on team size.

### Auto-delete head branches

Enable **Automatically delete head branches** to keep repository clean.

### Restrict who can push

Under **Restrict who can push to matching branches**, add specific teams or users.

## Rollback Instructions

To disable branch protection:

1. Go to Settings → Branches
2. Find the rule for `main`
3. Click **Delete** on the far right
4. Confirm deletion

## Monitoring

After enabling, monitor:

- PR merge times (should increase slightly)
- CI failure rate (track common failures)
- Developer feedback (ensure not too restrictive)
- Time to fix broken builds

## Support

If you encounter issues:

1. Check [GitHub Actions documentation](https://docs.github.com/en/actions)
2. Review workflow logs in Actions tab
3. Open an issue in the repository
4. Check CI troubleshooting section in CONTRIBUTING.md

---

**Last Updated:** 2024-12-28  
**Applies to:** CI workflow version 1.0
