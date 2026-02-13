# Confidential Security Hotfix

⚠️ **This repository is confidential.**

This repository contains a **private security hotfix for a critical Cosmos EVM vulnerability** that can lead to **fund loss and chain halts**. The issue is consensus-level, exploitable in practice, and has been locally reproduced.

Details are intentionally limited during the private disclosure window.
Please **do not share, fork, or discuss publicly** until disclosure.

Public disclosure is planned for **March 2nd**.

Thank you for your continued dedication to maintaining a safe and secure ecosystem.

---

## Upgrade Guidance

To reduce the risk of premature disclosure, it is **strongly recommended** that this fix is deployed via **compiled binaries distributed directly to validators**, rather than public source changes, until the disclosure window closes.

Upgrades must be performed in a **coordinated fashion**.

### Hotfix Tags

The following private tag is provided:

- `v0.6.x-papyrus-hotfix`

---

## Applying the Hotfix

### 1. Update your Git config to use private repositories

#### SSH Instructions

First, configure your machine to use SSH for Git. More details can be found here: https://docs.github.com/en/authentication/connecting-to-github-with-ssh.

To use SSH in `go mod` downloads, add these lines to `~/.gitconfig`:
```md
[url "ssh://git@github.com/"]
	insteadOf = https://github.com/
```


#### HTTPS Instructions

If you choose to use HTTPS, please follow the instructions here: https://go.dev/doc/faq#git_https.


### 2. Update `go.mod`

Add a `replace` directive pointing to this repository:

```go
replace github.com/cosmos/evm => github.com/cosmos/evm-sec-papyrus v0.6.x-papyrus-hotfix
```

Upgrade Cosmos SDK to v0.53.6:

```go
github.com/cosmos/cosmos-sdk v0.53.6
```

Then, tidy using the `GOPRIVATE` variable:

```bash
GOPRIVATE=github.com/cosmos/evm-sec-papyrus go mod tidy
```

---

### 2. Build and Deploy

For complete API-breaking changes and instructions, refer to the [migration docs](https://github.com/cosmos/evm-sec-papyrus/blob/release/v0.6.x/docs/migrations/v0.5.x_to_v0.6.0.md).

Rebuild your node binary using your standard process, distribute the compiled binary to validators, and perform a rolling upgrade.

---

## Notes

- Do not mirror this repository to public infrastructure
- Do not copy this repository to a public Github repository
- Do not reference this fix in public changelogs or releases before disclosure
