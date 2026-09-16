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

## 2. Unset `AWS_REGION` on switch when profile has no region

**Status:** Decided — **Option A: Always clean up**

**Decision:** When `awss select` switches to a profile that does not define a region, it emits `unset AWS_REGION` rather than leaving the variable untouched.

**Rationale:**
- Prevents a stale `AWS_REGION` from a previous `awss select` silently applying to the new profile
- Matches the behavior of tools like aws-vault and granted that clean up after themselves
- Makes the shell state predictable: after a switch, `AWS_REGION` always reflects the current profile

**Trade-off:** A user who sets `AWS_REGION` independently (outside of `awss`) will have it cleared on switch. This is acceptable because `awss` owns the region lifecycle once you start using it — mixing manual and tool-managed region state leads to confusion regardless.

---

## 3. SSO login flow on expired tokens

**Status:** Partly decided — `awss login` owns the login; `awss select` stays passive

**Decision:** `awss login` (Decision 5) runs the device login itself when the cached token is missing or expired, so the "how do I log in" question has an answer inside the tool. `awss select` still does not inspect the token cache. Whether `select` should warn on an expired token (Option B below) stays open.

**Context:** When a user selects an SSO profile whose cached token has expired, the next AWS API call will fail. The tool could detect this proactively and help.

**Option A: Don't handle it in `select`**
- Users can run `awss login` or `aws sso login --profile <name>`
- Keeps `select` simple and fast

**Option B: Detect and warn in `select`**
- After switching, check if the SSO token cache file exists and is expired
- Print a warning to stderr: `SSO token expired. Run: awss login`
- No automatic action, just a nudge

**Leaning toward:** B for `select`. The token cache code now lives in `internal/sso`, so the check is cheap to add.

---

## 4. MFA handling

**Status:** Decided — **Out of scope**

**Rationale:** Since Decision 1 chose `AWS_PROFILE` only, MFA is entirely the SDK's problem. When a tool uses a profile with `mfa_serial`, the SDK prompts for the TOTP code at credential resolution time. `awss` never resolves credentials, so there's nothing to handle.

---

## 5. SSO login and profile generation are in scope

**Status:** Decided

**Decision:** `awss login` discovers every account and role a user can reach through IAM Identity Center and writes one profile per role into `~/.aws/config`. With `--sts` it also writes short-lived keys into `~/.aws/credentials` for tools that cannot read SSO profiles.

**Rationale:**
- Switching profiles is only useful once profiles exist. For SSO users with many accounts, writing them by hand is the main friction.
- Decision 1 is untouched: `select` still exports only `AWS_PROFILE` and `AWS_REGION`. Credentials never enter the environment.
- The `--sts` path writes to the credentials file, not the shell, so the SDK still resolves through `AWS_PROFILE`.

**Guard rails:**
- Every section awss writes carries `awss_managed = true`. Sections without it are never modified or deleted.
- Writing over an unmanaged section is an error, not a silent overwrite.
- Stale managed sections for the same start URL are removed so `list` does not fill up with dead profiles.
- Start URL, SSO region, and name template come from flags or `~/.config/awss/config.yaml`. Nothing about any particular organization is built in.

---

## 6. Legacy vs `sso-session` profile format

**Status:** Open

**Context:** `awss login` writes the legacy per-profile format (`sso_start_url`, `sso_region`, `sso_account_id`, `sso_role_name` on each profile) and caches the token under `sha1(start_url)`, matching `aws sso login` in legacy mode. The newer `[sso-session]` format shares one block across profiles, supports token refresh, and caches under `sha1(session_name)`.

**Option A: Keep legacy format**
- Works with every SDK and CLI version
- Matches the current token cache logic

**Option B: Write `[sso-session]` blocks**
- Refreshable tokens, fewer browser prompts
- Requires changing the managed-section logic to track the session block and the cache key
- Older SDKs do not understand it

**Leaning toward:** B eventually, once the picker is done. Not blocking.
