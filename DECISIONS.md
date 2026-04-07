# Design Decisions

---

## 1. `AWS_PROFILE` only vs full credential export

**Status:** Decided — **Option A: `AWS_PROFILE` only**

**Decision:** `awss` is a profile switcher, not a credential resolver. It sets `AWS_PROFILE` (and optionally `AWS_REGION`) and lets the SDK handle everything else — SSO token exchange, assume-role chains, MFA prompts, credential refresh.

**Rationale:**
- `AWS_PROFILE` is respected by the AWS CLI, all official SDKs, Terraform, Pulumi, CDK, SAM, and Serverless Framework — covers the target use cases
- No risk of stale/expired credentials in env (the SDK resolves fresh on each call)
- No credential leakage into environment variables
- Keeps the tool focused and simple

**Note:** If full credential export (`--export`) is ever added, the shell wrapper contract doesn't change (still eval-ing export statements). The flag should unset `AWS_PROFILE` to avoid conflicts with the SDK credential chain precedence.

---

## 2. SSO login flow on expired tokens

**Status:** Open — decide during Phase 4 or 5

**Context:** When a user selects an SSO profile whose cached token has expired, the next AWS API call will fail. The tool could detect this proactively and help.

**Option A: Don't handle it — out of scope**
- `awss` is a profile switcher, not a credential manager
- Users already know to run `aws sso login --profile <name>`
- Keeps the tool simple and focused

**Option B: Detect and warn**
- After switching, check if the SSO token cache file exists and is expired
- Print a warning to stderr: `SSO token expired. Run: aws sso login --profile <name>`
- No automatic action, just a helpful nudge

**Option C: Detect and offer to login**
- Same detection as B, but prompt the user to run `aws sso login`
- Or run it automatically (risky — opens a browser)
- Granted does this automatically

**Leaning toward:** B — low cost, high value, no surprising side effects.

### Considerations
- Token cache lives at `~/.aws/sso/cache/` — is it feasible to check expiry without the SDK doing it?
- If using `aws-sdk-go-v2/config` to load profiles, does the SDK surface token expiry status?

---

## 3. MFA handling

**Status:** Decided — **Out of scope**

**Rationale:** Since Decision 1 chose `AWS_PROFILE` only, MFA is entirely the SDK's problem. When a tool uses a profile with `mfa_serial`, the SDK prompts for the TOTP code at credential resolution time. `awss` never resolves credentials, so there's nothing to handle.
